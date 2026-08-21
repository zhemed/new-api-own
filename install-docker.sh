#!/bin/bash
# 一键安装项目标准 Docker 环境：Docker 29.7.2 + Compose v5.4.0（Debian/Ubuntu）
# 用法：curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash
#    或：bash install-docker.sh
set -euo pipefail

REQUIRED_DOCKER="29.7.2"
REQUIRED_COMPOSE="v5.4.0"

if [ "$(id -u)" -ne 0 ]; then
  echo "请用 root 运行（sudo bash install-docker.sh）" >&2
  exit 1
fi

if ! grep -qiE 'debian|ubuntu' /etc/os-release 2>/dev/null; then
  echo "仅支持 Debian/Ubuntu，其他系统请手动安装 Docker $REQUIRED_DOCKER + Compose $REQUIRED_COMPOSE" >&2
  exit 1
fi

echo "==> 安装 Docker $REQUIRED_DOCKER + Compose $REQUIRED_COMPOSE"

# 安装 Docker 官方源（若未安装 docker）
if ! command -v docker >/dev/null 2>&1; then
  echo "未检测到 docker，安装官方源..."
  curl -fsSL https://get.docker.com | sh
fi

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y --allow-downgrades \
  docker-ce=5:${REQUIRED_DOCKER}* \
  docker-ce-cli=5:${REQUIRED_DOCKER}* \
  docker-compose-plugin=${REQUIRED_COMPOSE#v}* \
  docker-ce-rootless-extras docker-buildx-plugin containerd.io 2>&1 | tail -5

apt-mark hold docker-ce docker-ce-cli docker-ce-rootless-extras docker-buildx-plugin docker-compose-plugin containerd.io >/dev/null 2>&1 || true

systemctl enable --now docker >/dev/null 2>&1 || service docker start >/dev/null 2>&1 || true

DOCKER_VER=$(docker --version 2>/dev/null | grep -oP "Docker version \K[0-9.]+" || echo "missing")
COMPOSE_VER=$(docker compose version 2>/dev/null | grep -oP "Docker Compose version \K\S+" || echo "missing")

if [ "$DOCKER_VER" = "$REQUIRED_DOCKER" ] && [ "$COMPOSE_VER" = "$REQUIRED_COMPOSE" ]; then
  echo "✅ 完成：Docker $DOCKER_VER + Compose $COMPOSE_VER 已就绪（已 hold 锁定版本）"
else
  echo "⚠️ 安装后版本仍不符：Docker $DOCKER_VER（期望 $REQUIRED_DOCKER），Compose $COMPOSE_VER（期望 $REQUIRED_COMPOSE）" >&2
  echo "请检查 apt 源或手动执行：apt-get install -y docker-ce=5:${REQUIRED_DOCKER}* docker-ce-cli=5:${REQUIRED_DOCKER}* docker-compose-plugin=${REQUIRED_COMPOSE#v}*" >&2
  exit 1
fi
