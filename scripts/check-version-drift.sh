#!/usr/bin/env bash
#
# 版本对齐核查（机械执行，不靠人记）
#
# 比对四处版本来源，给出一句话结论与严格退出码：
#   1) 仓库 VERSION 文件
#   2) 最新 git tag（v 前缀可有可无）
#   3) ghcr 上 latest / <版本> / v<版本> 三个标签的镜像摘要（三者必须同摘要）
#   4) 可选：运行实例 /api/status 的 version（--instance <url>）
#
# 退出码：0 = 全部一致 / 1 = 发现不一致（drift）/ 2 = 无法判定（查不到、网络不可用、命令缺失）
# **任何"查不到"一律按 2 处理，绝不当成"一致"。**
#
# 用法：
#   scripts/check-version-drift.sh
#   scripts/check-version-drift.sh --instance http://127.0.0.1:3000
#   scripts/check-version-drift.sh --no-network            # 离线：只核对仓库内部，整体判定为 UNKNOWN
#   scripts/check-version-drift.sh --registry ghcr.io/user/repo
#   scripts/check-version-drift.sh --version 9.9.9         # 覆盖仓库版本（自测/CI 用）
#
# 只读：仅执行 `docker buildx imagetools inspect` 与 `curl /api/status`，不 push、不写任何状态。
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

REGISTRY_DEFAULT="ghcr.io/zhemed/new-api-own"
REGISTRY="${REGISTRY_DEFAULT}"
INSTANCE=""
VERSION_OVERRIDE=""
NO_NETWORK=0
TIMEOUT_SECS="${CHECK_VERSION_TIMEOUT:-30}"

usage() {
	cat <<'EOF'
版本对齐核查：比对 仓库 VERSION / 最新 git tag / ghcr 标签摘要 / 运行实例版本

用法: scripts/check-version-drift.sh [选项]

  --instance <url>     额外核对运行实例的 /api/status version（例如 http://127.0.0.1:3000）
  --registry <ref>     镜像仓库，默认 ghcr.io/zhemed/new-api-own
  --version <v>        用 <v> 代替 VERSION 文件的值（自测/CI 覆盖用）
  --no-network         跳过 registry 与实例检查（离线自测）；整体判定为 UNKNOWN(2)
  -h, --help           显示本帮助

退出码: 0=一致  1=不一致(drift)  2=无法判定(查不到/离线/命令缺失)
EOF
}

die_usage() {
	echo "参数错误: $1" >&2
	usage >&2
	exit 2
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--instance)
		[[ $# -ge 2 ]] || die_usage "--instance 需要 URL"
		INSTANCE="$2"
		shift 2
		;;
	--registry)
		[[ $# -ge 2 ]] || die_usage "--registry 需要镜像引用"
		REGISTRY="$2"
		shift 2
		;;
	--version)
		[[ $# -ge 2 ]] || die_usage "--version 需要版本号"
		VERSION_OVERRIDE="$2"
		shift 2
		;;
	--no-network)
		NO_NETWORK=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*) die_usage "未知参数 $1" ;;
	esac
done

with_timeout() {
	if command -v timeout >/dev/null 2>&1; then
		timeout "${TIMEOUT_SECS}" "$@"
	else
		"$@"
	fi
}

DRIFT=0
UNKNOWN=0
CHECKED=0
SKIPPED=0

echo "== 版本对齐核查 $(date -Is) =="
echo "仓库: ${REPO_ROOT}"
echo

# ---------------------------------------------------------------------------
# 1) 仓库 VERSION
# ---------------------------------------------------------------------------
VERSION_FILE="${REPO_ROOT}/VERSION"
if [[ ! -f "${VERSION_FILE}" ]]; then
	echo "❌ 找不到 ${VERSION_FILE}，无法判定"
	exit 2
fi
REPO_VERSION="$(tr -d '[:space:]' <"${VERSION_FILE}")"
if [[ -n "${VERSION_OVERRIDE}" ]]; then
	echo "• 仓库 VERSION=${REPO_VERSION}（本次用 --version ${VERSION_OVERRIDE} 覆盖参与比对）"
	REPO_VERSION="${VERSION_OVERRIDE}"
fi
if [[ -z "${REPO_VERSION}" ]]; then
	echo "❌ VERSION 文件为空，无法判定"
	exit 2
fi

# ---------------------------------------------------------------------------
# 2) 最新 git tag
# ---------------------------------------------------------------------------
LATEST_TAG=""
if git -C "${REPO_ROOT}" rev-parse --git-dir >/dev/null 2>&1; then
	if tags=$(git -C "${REPO_ROOT}" tag --sort=-v:refname 2>/dev/null); then
		LATEST_TAG="$(printf '%s\n' "${tags}" | sed -n '1p')"
	fi
fi
if [[ -z "${LATEST_TAG}" ]]; then
	echo "⚠️  仓库里没有可用的 git tag → 仓库内部一致性无法判定"
	UNKNOWN=1
	SKIPPED=$((SKIPPED + 1))
else
	TAG_VERSION="${LATEST_TAG#v}"
	CHECKED=$((CHECKED + 1))
	if [[ "${TAG_VERSION}" == "${REPO_VERSION}" ]]; then
		echo "✅ 仓库: VERSION=${REPO_VERSION} == 最新 tag ${LATEST_TAG}"
	else
		echo "❌ 仓库: VERSION=${REPO_VERSION} != 最新 tag ${LATEST_TAG}"
		DRIFT=1
	fi
fi

# ---------------------------------------------------------------------------
# 3) registry 标签摘要（latest / <版本> / v<版本>）
# ---------------------------------------------------------------------------
DOCKER_OK=1
if [[ ${NO_NETWORK} -eq 0 ]]; then
	if ! command -v docker >/dev/null 2>&1; then
		DOCKER_OK=0
		echo "⚠️  未安装 docker → registry 无法判定"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
	fi
fi

FIRST_TAG=""
FIRST_DIGEST=""
for tag in latest "${REPO_VERSION}" "v${REPO_VERSION}"; do
	if [[ ${NO_NETWORK} -eq 1 ]]; then
		echo "⏭  registry: 跳过 ${REGISTRY}:${tag}（--no-network）"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
		continue
	fi
	if [[ ${DOCKER_OK} -eq 0 ]]; then
		SKIPPED=$((SKIPPED + 1))
		continue
	fi

	if ! out=$(with_timeout docker buildx imagetools inspect "${REGISTRY}:${tag}" 2>&1); then
		first_line="$(printf '%s\n' "${out}" | sed -n '1p')"
		echo "⚠️  registry: ${REGISTRY}:${tag} 查询失败（网络/权限/标签不存在）：${first_line:-无输出}"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
		continue
	fi

	digest="$(printf '%s\n' "${out}" | grep -m1 '^Digest:' | awk '{print $2}' || true)"
	if [[ -z "${digest}" ]]; then
		echo "⚠️  registry: ${REGISTRY}:${tag} 输出里没有 Digest 行，无法判定"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
		continue
	fi

	CHECKED=$((CHECKED + 1))
	echo "• registry: ${REGISTRY}:${tag} → ${digest}"
	if [[ -z "${FIRST_DIGEST}" ]]; then
		FIRST_TAG="${tag}"
		FIRST_DIGEST="${digest}"
	elif [[ "${digest}" != "${FIRST_DIGEST}" ]]; then
		echo "❌ registry: ${tag} 摘要 ${digest} 与 ${FIRST_TAG} 摘要 ${FIRST_DIGEST} 不一致"
		DRIFT=1
	fi
done

# ---------------------------------------------------------------------------
# 4) 运行实例 /api/status
# ---------------------------------------------------------------------------
extract_version() {
	if command -v jq >/dev/null 2>&1; then
		jq -r '.data.version // empty' 2>/dev/null || true
		return
	fi
	if command -v python3 >/dev/null 2>&1; then
		python3 -c 'import json,sys
try:
    data = json.load(sys.stdin)
except Exception:
    sys.exit(0)
if isinstance(data, dict):
    inner = data.get("data") or {}
    if isinstance(inner, dict):
        print(inner.get("version") or "")
' 2>/dev/null || true
		return
	fi
	sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | sed -n '1p' || true
}

if [[ -z "${INSTANCE}" ]]; then
	echo "⏭  实例: 未指定（--instance <url> 可核对运行实例版本）"
	SKIPPED=$((SKIPPED + 1))
else
	STATUS_URL="${INSTANCE%/}/api/status"
	if [[ ${NO_NETWORK} -eq 1 ]]; then
		echo "⏭  实例: 跳过 ${STATUS_URL}（--no-network）"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
	elif ! command -v curl >/dev/null 2>&1; then
		echo "⚠️  实例: 未安装 curl → 无法判定 ${STATUS_URL}"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
	elif ! body=$(with_timeout curl -sS --max-time "${TIMEOUT_SECS}" "${STATUS_URL}" 2>&1); then
		first_line="$(printf '%s\n' "${body}" | sed -n '1p')"
		echo "⚠️  实例: ${STATUS_URL} 不可达（${first_line:-无输出}）→ 无法判定"
		UNKNOWN=1
		SKIPPED=$((SKIPPED + 1))
	else
		instance_version="$(printf '%s' "${body}" | extract_version)"
		if [[ -z "${instance_version}" ]]; then
			echo "⚠️  实例: ${STATUS_URL} 响应里读不到 version → 无法判定"
			UNKNOWN=1
			SKIPPED=$((SKIPPED + 1))
		else
			CHECKED=$((CHECKED + 1))
			if [[ "${instance_version}" == "${REPO_VERSION}" ]]; then
				echo "✅ 实例: ${STATUS_URL} version=${instance_version} == VERSION=${REPO_VERSION}"
			else
				echo "❌ 实例: ${STATUS_URL} version=${instance_version} != VERSION=${REPO_VERSION}"
				DRIFT=1
			fi
		fi
	fi
fi

# ---------------------------------------------------------------------------
# 判定
# ---------------------------------------------------------------------------
echo
echo "已核对 ${CHECKED} 项，跳过/无法判定 ${SKIPPED} 项"
if [[ ${DRIFT} -eq 1 ]]; then
	echo "==> 判定: DRIFT（三处版本不一致，按事故处理）"
	exit 1
fi
if [[ ${UNKNOWN} -eq 1 ]]; then
	echo "==> 判定: UNKNOWN（有检查项查不到/被跳过，不能当作一致；联网或修正参数后重跑）"
	exit 2
fi
echo "==> 判定: OK（仓库 VERSION、最新 tag、registry 标签摘要、实例版本一致）"
exit 0
