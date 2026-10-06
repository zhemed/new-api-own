#!/bin/sh
set -eu
# 本项目只有一种部署方式（强制）：
#   docker run + 公开镜像 ghcr.io/zhemed/new-api-own（见 README「部署」）
# 禁止在仓库里重新引入第二种编排形态（compose / helm / k8s 清单）。
#
# 为什么做成闸门而不是只写文档：compose 编排曾被清理两次、又被三次回滚带回，
# 每次都靠人记得；这道检查在提交当场与 CI 上拦，回滚也带不回来。
# 规则与 do-not-restore 清单见 .trellis/spec/guides/deployment-single-method.md

fail=0

by_name=$(git ls-files | grep -Ei '(^|/)(docker-)?compose(\.[a-z0-9._-]+)?\.ya?ml$' || true)
if [ -n "$by_name" ]; then
  echo "❌ 出现 compose 编排文件（本项目只允许 docker run + 公开镜像）："
  printf '%s\n' "$by_name" | sed 's/^/   /'
  fail=1
fi

by_dir=$(git ls-files | grep -Ei '(^|/)(charts?|helm|k8s|kubernetes|manifests?)/' || true)
if [ -n "$by_dir" ]; then
  echo "❌ 出现集群编排目录（本项目只允许 docker run + 公开镜像）："
  printf '%s\n' "$by_dir" | sed 's/^/   /'
  fail=1
fi

# 改名/换壳也拦：compose 的特征字段出现在任何 yml/yaml 里
by_content=$(git grep -l -E '^\s*(network_mode|services)\s*:' -- '*.yml' '*.yaml' 2>/dev/null | grep -v '^\.github/' || true)
if [ -n "$by_content" ]; then
  echo "❌ 这些 YAML 带 compose 特征字段（services/network_mode）："
  printf '%s\n' "$by_content" | sed 's/^/   /'
  fail=1
fi

# 裸机 / systemd 也是第二种部署方式：根级 unit 文件，或 systemd|deploy|etc 目录下的 unit
by_unit=$(git ls-files | grep -E '(^|/)(systemd|deploy|etc)/.*\.(service|timer|socket)$|^[^/]+\.(service|timer|socket)$' || true)
if [ -n "$by_unit" ]; then
  echo "❌ 出现 systemd unit（裸机部署形态，本项目只允许 docker run + 公开镜像）："
  printf '%s\n' "$by_unit" | sed 's/^/   /'
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  echo ""
  echo "   规则：.trellis/spec/guides/deployment-single-method.md"
  echo "   要新增部署形态，先改规则并得到仓库所有者明确同意。"
  exit 1
fi

echo "✅ 部署方式唯一：docker run + ghcr.io/zhemed/new-api-own"
