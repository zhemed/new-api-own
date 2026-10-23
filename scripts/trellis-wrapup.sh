#!/usr/bin/env bash
#
# Trellis 收尾一条命令：finish → archive → push → 校验（0 0 + 工作区干净）
#
# 为什么必须一条命令做完：闸门在**工具调用边界**检查"是否有进行中任务"。
# `task.py archive` 会当场产生一个纯 .trellis 的归档提交，此时本会话已无进行中任务，
# 之后再单独调用 `git push` 会被拦掉。把整条收尾链放进同一次调用即可整体通过，
# 不必再建临时任务。背景与标准序列见 .trellis/spec/guides/trellis-gate-guide.md。
#
# 用法:
#   scripts/trellis-wrapup.sh <任务名或slug> [--dry-run]
#
# 退出码: 0 成功；1 守卫拒绝或某一步失败；2 用法错误。
#
# 守卫（拒绝即退出，且不做任何改动）:
#   * 当前不在 git 仓库 / 没有 origin 远端 / 处于 detached HEAD；
#   * 跟踪文件有未提交改动（未跟踪文件不算）；
#   * 任务目录不存在（含"已归档"）或 task.json 的 status 不是 in_progress；
#   * 远端预检失败（git push --dry-run），避免"归档成功、推送失败"后无任务可用。
#
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}" || exit 1

TASKS_DIR=".trellis/tasks"
TASK_PY=".trellis/scripts/task.py"

usage() {
  cat <<'EOF'
用法: scripts/trellis-wrapup.sh <任务名或slug> [--dry-run]

  <任务名或slug>  .trellis/tasks/ 下的目录名（如 10-23-my-task）或 slug（my-task）
  --dry-run       只打印将要执行的命令，不做任何改动（不 finish / 不 archive / 不 push）

步骤: finish → archive --skip-branch-validation → push origin <当前分支> → 校验 0 0 + 工作区干净
EOF
}

title() { printf '\n\033[1m== %s ==\033[0m\n' "$*"; }
ok()    { printf '  \033[32m✓\033[0m %s\n' "$*"; }
warn()  { printf '  \033[33m!\033[0m %s\n' "$*"; }
bad()   { printf '  \033[31m✗\033[0m %s\n' "$*"; }

refuse() {
  title "拒绝执行"
  bad "$*"
  printf '  未做任何改动；处理完上面的问题后重跑本脚本。\n'
  exit 1
}

fail_step() {
  title "执行失败"
  bad "$*"
  exit 1
}

# ---------- 参数 ----------
TASK_ARG=""
DRY_RUN=0
for arg in "$@"; do
  case "${arg}" in
    --dry-run) DRY_RUN=1 ;;
    -h|--help) usage; exit 0 ;;
    -*)
      printf '未知参数: %s\n\n' "${arg}"
      usage
      exit 2
      ;;
    *)
      if [ -n "${TASK_ARG}" ]; then
        printf '只接受一个任务名（多余: %s）\n\n' "${arg}"
        usage
        exit 2
      fi
      TASK_ARG="${arg}"
      ;;
  esac
done
if [ -z "${TASK_ARG}" ]; then
  usage
  exit 2
fi

# ---------- 守卫 1: git 仓库 / origin / 分支 ----------
git rev-parse --show-toplevel >/dev/null 2>&1 || refuse "当前目录不在 git 仓库里：${REPO_ROOT}"

REMOTE_URL="$(git remote get-url origin 2>/dev/null || true)"
[ -n "${REMOTE_URL}" ] || refuse "没有 origin 远端，无法推送（先 git remote add origin <url>）"

BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"
[ -n "${BRANCH}" ] && [ "${BRANCH}" != "HEAD" ] || refuse "当前处于 detached HEAD，无法确定要推送的分支"

# ---------- 守卫 2: 跟踪文件必须已全部提交 ----------
dirty="$(git status --porcelain | grep -v '^??' || true)"
if [ -n "${dirty}" ]; then
  title "拒绝执行"
  bad "存在未提交的跟踪文件改动（先提交或撤销，再收尾）："
  printf '%s\n' "${dirty}"
  printf '  未做任何改动。\n'
  exit 1
fi

# ---------- 守卫 3: 解析任务目录 ----------
TASK_DIR=""
if [ -d "${TASKS_DIR}/${TASK_ARG}" ]; then
  TASK_DIR="${TASKS_DIR}/${TASK_ARG}"
else
  matches="$(find "${TASKS_DIR}" -maxdepth 1 -mindepth 1 -type d -name "*-${TASK_ARG}" 2>/dev/null | sort)"
  match_count="$(printf '%s\n' "${matches}" | grep -c . || true)"
  if [ "${match_count}" -gt 1 ]; then
    refuse "任务名「${TASK_ARG}」匹配到多个目录，请用完整目录名：$(printf '%s' "${matches}" | tr '\n' ' ')"
  fi
  [ "${match_count}" -eq 1 ] && TASK_DIR="${matches}"
fi

if [ -z "${TASK_DIR}" ]; then
  archived="$(find "${TASKS_DIR}/archive" -mindepth 2 -maxdepth 2 -type d -name "*-${TASK_ARG}" 2>/dev/null | sort || true)"
  if [ -n "${archived}" ]; then
    refuse "任务「${TASK_ARG}」已归档（${archived}），没有可收尾的进行中任务"
  fi
  refuse "找不到任务「${TASK_ARG}」（应为 ${TASKS_DIR}/<MM-DD>-${TASK_ARG} 或完整目录名）"
fi

TASK_DIR_NAME="$(basename "${TASK_DIR}")"
TASK_JSON="${TASK_DIR}/task.json"
[ -f "${TASK_JSON}" ] || refuse "任务目录缺少 task.json：${TASK_JSON}"

# ---------- 守卫 4: 任务必须是 in_progress ----------
status="$(grep -oE '"status"[[:space:]]*:[[:space:]]*"[^"]*"' "${TASK_JSON}" | head -1 | sed 's/.*"\([^"]*\)"$/\1/')"
[ "${status}" = "in_progress" ] || refuse "任务 ${TASK_DIR_NAME} 的 status=${status:-未知}，只有 in_progress 才能收尾"

# ---------- 守卫 5: 远端预检（不产生改动） ----------
if [ "${DRY_RUN}" = 1 ]; then
  preflight_note="跳过（--dry-run 不做网络动作）"
else
  if ! preflight_out="$(git push --dry-run origin "${BRANCH}" 2>&1)"; then
    title "拒绝执行"
    bad "远端预检失败（git push --dry-run origin ${BRANCH}），先修好推送通道再收尾："
    printf '%s\n' "${preflight_out}"
    printf '  未做任何改动（任务仍是 in_progress）。\n'
    exit 1
  fi
  preflight_note="通过"
fi

# ---------- 计划 ----------
title "收尾计划"
printf '  仓库:     %s\n' "${REPO_ROOT}"
printf '  任务:     %s（status=%s）\n' "${TASK_DIR_NAME}" "${status}"
printf '  分支:     %s → %s\n' "${BRANCH}" "${REMOTE_URL}"
printf '  远端预检: %s\n' "${preflight_note}"

if [ "${DRY_RUN}" = 1 ]; then
  title "dry-run：以下命令不会被执行"
  printf '  python3 %s finish\n' "${TASK_PY}"
  printf '  python3 %s archive %s --skip-branch-validation\n' "${TASK_PY}" "${TASK_DIR_NAME}"
  printf '  git push origin %s\n' "${BRANCH}"
  printf '  git rev-list --left-right --count origin/%s...HEAD   # 期望 0\\t0\n' "${BRANCH}"
  printf '\n  （dry-run 未改动任何文件、未产生提交、未推送）\n'
  exit 0
fi

# ---------- 1. finish ----------
title "1/4 finish（清除当前任务指针）"
if ! python3 "${TASK_PY}" finish; then
  fail_step "task.py finish 失败；任务目录仍在 ${TASK_DIR}，修正后重跑本脚本"
fi
ok "已 finish"

# ---------- 2. archive ----------
title "2/4 archive（移动到 archive/YYYY-MM/ 并自动提交）"
HEAD_BEFORE="$(git rev-parse HEAD)"
if ! archive_dest="$(python3 "${TASK_PY}" archive "${TASK_DIR_NAME}" --skip-branch-validation)"; then
  fail_step "task.py archive 失败；任务目录仍在 ${TASK_DIR}（finish 已执行，重复执行无害）"
fi
HEAD_AFTER="$(git rev-parse HEAD)"
if [ -n "${archive_dest}" ]; then
  ok "归档到 ${archive_dest}"
else
  warn "archive 未打印归档路径（请检查 ${TASKS_DIR}/archive/）"
fi

if [ "${HEAD_BEFORE}" != "${HEAD_AFTER}" ]; then
  ARCHIVE_COMMIT="$(git rev-parse --short HEAD)"
  ARCHIVE_SUBJECT="$(git log -1 --format=%s HEAD)"
  ok "归档提交 ${ARCHIVE_COMMIT} ${ARCHIVE_SUBJECT}"
else
  ARCHIVE_COMMIT="（无）"
  warn "archive 未产生新提交（session_auto_commit=false 或任务本就已提交？）"
fi

# ---------- 3. push ----------
title "3/4 push origin ${BRANCH}"
if git rev-parse --verify --quiet "refs/remotes/origin/${BRANCH}" >/dev/null; then
  range="$(git log --oneline "origin/${BRANCH}..HEAD" 2>/dev/null || true)"
  range_count="$(printf '%s\n' "${range}" | grep -c . || true)"
  if [ "${range_count}" -gt 0 ]; then
    ok "待推送 ${range_count} 个提交："
    printf '%s\n' "${range}" | sed 's/^/      /'
  else
    ok "本地已与 origin/${BRANCH} 一致，无需推送"
  fi
else
  warn "远端还没有 ${BRANCH} 分支（本次为首次推送）"
fi

if ! push_out="$(git push origin "${BRANCH}" 2>&1)"; then
  title "推送失败"
  bad "git push origin ${BRANCH} 失败（归档已完成，本地提交 ${ARCHIVE_COMMIT} 仍在）："
  printf '%s\n' "${push_out}"
  printf '  恢复：确认网络/凭据后重推（必要时在会话外执行 git push origin %s）；\n' "${BRANCH}"
  printf '  本会话已无进行中任务，后续工具调用可能被闸门拦下。\n'
  exit 1
fi
printf '%s\n' "${push_out}" | sed 's/^/      /'
ok "已推送"

# ---------- 4. 校验 ----------
title "4/4 校验（0 0 + 工作区干净）"
counts="$(git rev-list --left-right --count "origin/${BRANCH}...HEAD")"
behind="${counts%%[[:space:]]*}"
ahead="${counts##*[[:space:]]}"
if [ "${behind}" = "0" ] && [ "${ahead}" = "0" ]; then
  ok "origin/${BRANCH} 与 HEAD 同步（0 0）"
else
  bad "未同步：behind=${behind} ahead=${ahead}"
  exit 1
fi

remaining="$(git status --porcelain | grep -v '^??' || true)"
if [ -n "${remaining}" ]; then
  bad "工作区仍有未提交改动："
  printf '%s\n' "${remaining}"
  exit 1
fi
ok "工作区干净（跟踪文件无未提交改动）"

untracked="$(git status --porcelain | grep -c '^??' || true)"
if [ "${untracked}" != "0" ]; then
  warn "另有 ${untracked} 个未跟踪文件（不计入干净判定，注意别漏提交）"
fi

title "收尾完成"
printf '  任务:     %s → %s\n' "${TASK_DIR_NAME}" "${archive_dest:-${TASKS_DIR}/archive/}"
printf '  归档提交: %s\n' "${ARCHIVE_COMMIT}"
printf '  推送:     %s → origin/%s（0 0）\n' "${BRANCH}" "${BRANCH}"
printf '  提醒:     还在会话内的话，journal 记录（add_session.py）要在这条命令之前或同一次调用内完成。\n'
