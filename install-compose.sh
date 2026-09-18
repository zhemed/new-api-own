#!/bin/bash
#
# new-api-own · docker compose 一条命令部署
#
#   curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-compose.sh | sudo bash
#
# 它做的事：预检 Docker/Compose → 建项目目录（默认 /opt/docker/new-api-own，750）
#   → 生成或沿用 .env（随机口令，600）→ 拉取同 ref 的 docker-compose.yml
#   → docker compose up -d → 等 new-api 健康检查通过 → 收紧数据目录权限 → 打印运维信息。
#
# 参数（也可用环境变量；括号内为对应环境变量）：
#   --dir <路径>          项目目录（默认 /opt/docker/new-api-own；NEW_API_COMPOSE_DIR）
#   --ref <git ref>       取 compose 与 VERSION 的 ref（默认 main；NEW_API_COMPOSE_REF）
#   --raw-base <URL>      取文件的来源基址（默认官方 raw 地址；NEW_API_RAW_BASE）——fork 或镜像源时用
#   --tag <镜像 tag>      镜像 tag（默认按 ref 上 VERSION 推导 v<版本>；NEW_API_IMAGE_TAG）
#   --project-name <名>   compose 项目名，决定 pg_data 卷名（默认 new-api-own；NEW_API_PROJECT_NAME）
#   --name-prefix <前缀>  三个容器名统一加前缀（默认无；NEW_API_NAME_PREFIX）
#   --tz <时区>           容器时区（默认取宿主机 /etc/timezone；NEW_API_TZ）
#   --allow-docker-mismatch  在非标准 Docker 环境上强行部署（本仓库标准：29.7.2 + v5.4.0）
#   --force               已有部署时覆盖 compose（默认拒绝；旧文件先备份成 .bak-<时间戳>-<微秒>）
#   --no-start            干跑：只写文件，不启动，也不做容器名预检
#   -h, --help            显示本帮助
#
# 不变量（改这个脚本时不要破坏它们）：
#   * 已有部署默认拒绝覆盖；--force 也**绝不**删除或覆盖 ./data、./logs 与 pg_data 命名卷；
#   * .env 已存在则沿用既有口令，**绝不重新生成** —— Postgres 口令只在数据目录为空时生效，
#     重生成会让应用连不上自己的库，现场表现是"重装后面板里数据全丢"，极难自查；
#   * compose 项目名不变更（它决定 pg_data 的卷名，改了等于换一个新空卷）；
#   * 口令不打印到输出、不写进仓库。
#
set -euo pipefail

RAW_BASE="${NEW_API_RAW_BASE:-https://raw.githubusercontent.com/zhemed/new-api-own}"
REQUIRED_DOCKER="29.7.2"
REQUIRED_COMPOSE="v5.4.0"
HEALTH_TIMEOUT=120   # 等健康检查的上限（秒）
BACKUP_KEEP=3        # compose / .env 备份各保留最近几份

DIR="${NEW_API_COMPOSE_DIR:-/opt/docker/new-api-own}"
REF="${NEW_API_COMPOSE_REF:-main}"
TAG="${NEW_API_IMAGE_TAG:-}"
PROJECT="${NEW_API_PROJECT_NAME:-new-api-own}"
PREFIX="${NEW_API_NAME_PREFIX:-}"
TZ_VALUE="${NEW_API_TZ:-}"
FORCE=0
START=1
ALLOW_MISMATCH=0

log()  { printf '%s\n' "$*"; }
ok()   { printf '\033[32m%s\033[0m\n' "$*"; }
warn() { printf '\033[33m[警告] %s\033[0m\n' "$*"; }
die()  { printf '\033[31m[错误] %s\033[0m\n' "$*" >&2; exit 1; }

usage() {
  # curl | bash 形态下 $0 是 bash、读不到文件，此时给出可自取的等价做法。
  if [ -r "$0" ] && [ "$(head -c 2 "$0" 2>/dev/null || true)" = "#!" ]; then
    sed -n '2,/^set -euo pipefail/p' "$0" | sed '$d'
  else
    log "参数：--dir --ref --tag --project-name --name-prefix --tz --allow-docker-mismatch --force --no-start --help"
    log "完整用法（含不变量说明）：curl -fsSL ${RAW_BASE}/${REF}/install-compose.sh | sed -n '1,40p'"
  fi
}

# 生成 32 位十六进制随机串。
# 只用 hex：口令要拼进 postgresql:// 与 redis:// 形式的 DSN，非 URL 安全字符会引入解析歧义。
# 用 head 直接读 /dev/urandom（不是 tr ... | head，那会让 tr 收到 SIGPIPE，在 pipefail 下中断脚本）。
rand_hex() { head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'; }

# 从 .env 取值：只做行匹配，不 eval，避免执行 .env 里的内容。
read_env_value() { sed -n "s/^$1=//p" "$2" 2>/dev/null | head -n 1; }

# 唯一到微秒的备份名；同一秒内连跑多次也不会互相覆盖（只到秒时"保留最近 N 份"会形同虚设）。
unique_backup_name() {
  local base="$1" ts ns candidate
  ts="$(date -u +%Y%m%d-%H%M%S)"
  ns="$(date -u +%N 2>/dev/null || true)"
  case "$ns" in
    '' | *[!0-9]*) ns="$$-$RANDOM" ;;
    *) ns="${ns:0:6}" ;;
  esac
  candidate="$DIR/$base.bak-$ts-$ns"
  while [ -e "$candidate" ]; do
    candidate="$DIR/$base.bak-$ts-$ns-$RANDOM"
  done
  printf '%s' "$candidate"
}

# 备份一个已存在的文件，并在必要时清理过旧的备份。
# 只清理 <名字>.bak-*（本脚本自己生成的），不碰其它任何文件。
backup_existing() {
  local name="$1" path="$DIR/$1"
  [ -f "$path" ] || return 0
  local target
  target="$(unique_backup_name "$name")"
  mv "$path" "$target"
  warn "已把 $name 备份为 $(basename "$target")"
  local old
  # 这里的文件名全部由本脚本生成（<已知文件名>.bak-<时间戳>-<微秒>），不含空格与换行。
  # shellcheck disable=SC2012
  old="$(ls -1t "$DIR/$name".bak-* 2>/dev/null | tail -n +$((BACKUP_KEEP + 1)) || true)"
  if [ -n "$old" ]; then
    while IFS= read -r f; do
      rm -f "$f" && warn "清理较旧的备份 $(basename "$f")（只保留最近 $BACKUP_KEEP 份 $name 备份）"
    done <<< "$old"
  fi
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dir) DIR="${2:?--dir 需要路径}"; shift 2 ;;
    --ref) REF="${2:?--ref 需要 git ref}"; shift 2 ;;
    --raw-base) RAW_BASE="${2:?--raw-base 需要 URL}"; shift 2 ;;
    --tag) TAG="${2:?--tag 需要镜像 tag}"; shift 2 ;;
    --project-name) PROJECT="${2:?--project-name 需要名字}"; shift 2 ;;
    --name-prefix) PREFIX="${2:?--name-prefix 需要前缀}"; shift 2 ;;
    --tz) TZ_VALUE="${2:?--tz 需要时区}"; shift 2 ;;
    --allow-docker-mismatch) ALLOW_MISMATCH=1; shift ;;
    --force) FORCE=1; shift ;;
    --no-start) START=0; shift ;;
    -h | --help) usage; exit 0 ;;
    *) die "未知参数：$1（用 --help 看用法）" ;;
  esac
done

[ "$(id -u)" -eq 0 ] || die "需要 root：curl -fsSL ${RAW_BASE}/${REF}/install-compose.sh | sudo bash"
[ -n "$PROJECT" ] || die "--project-name 不能为空"
case "$PREFIX" in
  *[!A-Za-z0-9._-]*) die "--name-prefix 只允许字母数字与 . _ -" ;;
esac

if [ -z "$TZ_VALUE" ]; then
  TZ_VALUE="$(cat /etc/timezone 2>/dev/null || true)"
  TZ_VALUE="$(printf '%s' "$TZ_VALUE" | tr -d '[:space:]')"
  [ -n "$TZ_VALUE" ] || TZ_VALUE="Asia/Shanghai"
fi
[ -n "$REF" ] || die "--ref 不能为空"

# ---------- 1. Docker 环境预检 ----------
command -v docker >/dev/null 2>&1 || die "未找到 docker：先装 Docker Engine + Compose v2（见 README / https://docs.docker.com/engine/install/）"
docker compose version >/dev/null 2>&1 || die "未找到 docker compose（v2）：需要 docker-compose-plugin"

DOCKER_VER="$(docker --version 2>/dev/null | grep -oP 'Docker version \K[0-9.]+' || echo missing)"
COMPOSE_VER="$(docker compose version 2>/dev/null | grep -oP 'Docker Compose version \K\S+' || echo missing)"
if [ "$DOCKER_VER" != "$REQUIRED_DOCKER" ] || [ "$COMPOSE_VER" != "$REQUIRED_COMPOSE" ]; then
  if [ "$ALLOW_MISMATCH" = 1 ]; then
    warn "Docker 环境非标准：实测 Docker $DOCKER_VER / Compose $COMPOSE_VER，标准为 Docker $REQUIRED_DOCKER + Compose $REQUIRED_COMPOSE（已按 --allow-docker-mismatch 继续）"
  else
    die "Docker 环境未达标：实测 Docker $DOCKER_VER / Compose $COMPOSE_VER，本仓库标准是 Docker $REQUIRED_DOCKER + Compose $REQUIRED_COMPOSE。
      装标准环境：curl -fsSL ${RAW_BASE}/${REF}/install-docker.sh | bash
      确实要在非标准环境部署：加 --allow-docker-mismatch"
  fi
fi

# ---------- 2. 解析版本（不写字面量，避免与 VERSION 漂移）----------
if [ -z "$TAG" ]; then
  version_raw="$(curl -fsSL "${RAW_BASE}/${REF}/VERSION" 2>/dev/null || true)"
  version_trimmed="$(printf '%s' "$version_raw" | tr -d '[:space:]')"
  [ -n "$version_trimmed" ] || die "取不到 ${REF} 上的 VERSION 文件；请显式指定版本：--tag <镜像 tag>"
  TAG="v${version_trimmed}"
fi

# ---------- 3. 先拉 compose（坏 ref 要在动文件系统之前失败）----------
tmp_compose="$(mktemp)"
tmp_env="$(mktemp)"
trap 'rm -f "$tmp_compose" "$tmp_env"' EXIT
curl -fsSL "${RAW_BASE}/${REF}/docker-compose.yml" -o "$tmp_compose" \
  || die "拉取 ${REF} 的 docker-compose.yml 失败：检查 ref 是否存在、网络是否可达"
[ -s "$tmp_compose" ] || die "拉取到的 docker-compose.yml 是空文件"
docker compose -f "$tmp_compose" config >/dev/null 2>&1 \
  || die "拉取到的 docker-compose.yml 不是合法的 compose 文件（ref=$REF）；这通常意味着 ref 下没有我们这个文件"

# ---------- 4. 项目目录与 .env ----------
mkdir -p "$DIR" "$DIR/data" "$DIR/logs"
chmod 750 "$DIR" 2>/dev/null || true
# data/ 与 logs/ 含明文上游 Key，创建时就收紧（不等启动完），--no-start 干跑也生效。
chmod 700 "$DIR/data" "$DIR/logs" 2>/dev/null || true

ENV_FILE="$DIR/.env"
if [ -f "$ENV_FILE" ]; then
  PG_PW="$(read_env_value POSTGRES_PASSWORD "$ENV_FILE")"
  RD_PW="$(read_env_value REDIS_PASSWORD "$ENV_FILE")"
  if [ -z "$PG_PW" ] || [ -z "$RD_PW" ]; then
    die "已有 $ENV_FILE，但缺少 POSTGRES_PASSWORD 或 REDIS_PASSWORD。
      本脚本不会覆盖它——口令一旦重生成，Postgres 会认证失败、应用连不上自己的库。
      请手工补上这两个键，或删除该文件后重跑（前者更安全）。"
  fi
  existing_project="$(read_env_value COMPOSE_PROJECT_NAME "$ENV_FILE")"
  if [ -n "$existing_project" ] && [ "$existing_project" != "$PROJECT" ]; then
    die "既有 .env 的项目名是 \"$existing_project\"，本次要写成 \"$PROJECT\"。
      compose 项目名决定 pg_data 的**卷名**，改名等于指向一个新的空卷——数据看起来会全部消失。
      要沿用既有数据：加 --project-name $existing_project
      确实要换（旧数据已迁移，或本就打算重来）：删除 $ENV_FILE 后重跑"
  fi
  ok "沿用既有口令（未重新生成）"
else
  PG_PW="$(rand_hex)"
  RD_PW="$(rand_hex)"
fi

cat > "$tmp_env" <<EOF
# 由 install-compose.sh 生成（$(date -u +%Y-%m-%dT%H:%M:%SZ)）
# 本文件含数据库口令：权限 600，不要提交进任何仓库。
# 重跑安装脚本会沿用这里的口令；换项目名会让 pg_data 指向新卷。
COMPOSE_PROJECT_NAME=$PROJECT
POSTGRES_PASSWORD=$PG_PW
REDIS_PASSWORD=$RD_PW
NEW_API_IMAGE_TAG=$TAG
NEW_API_CONTAINER_NAME=${PREFIX}new-api
REDIS_CONTAINER_NAME=${PREFIX}redis
POSTGRES_CONTAINER_NAME=${PREFIX}postgres
EOF

# ---------- 5. 写文件（已有部署默认拒绝覆盖）----------
COMPOSE_FILE="$DIR/docker-compose.yml"
LEGACY_NAMES=()
for f in compose.yaml compose.yml docker-compose.yaml; do
  [ -f "$DIR/$f" ] && LEGACY_NAMES+=("$f")
done
EXISTS=0
[ -f "$COMPOSE_FILE" ] && EXISTS=1
[ "${#LEGACY_NAMES[@]}" -gt 0 ] && EXISTS=1

if [ "$EXISTS" = 1 ] && [ "$FORCE" != 1 ]; then
  if [ "${#LEGACY_NAMES[@]}" -gt 0 ]; then
    die "项目目录里已有 compose 文件：${LEGACY_NAMES[*]}
      本脚本用的是 docker-compose.yml；多个 compose 文件名并存会让每条 compose 命令都告警。
      迁移方式（只留一个；改名后容器上的配置标签仍指向旧路径，需要一次重建）：
        mv $DIR/${LEGACY_NAMES[0]} $COMPOSE_FILE
        cd $DIR && docker compose up -d --force-recreate
      确实要覆盖：加 --force（会把已有文件逐个改名成 .bak-<时间戳> 再写新的）"
  else
    die "$COMPOSE_FILE 已存在（可能是现有部署）。要覆盖请加 --force（会先备份）；
      只想启停请直接在该目录跑 docker compose up -d / down。"
  fi
fi

if [ "$EXISTS" = 1 ] && [ "$FORCE" = 1 ]; then
  for f in docker-compose.yml "${LEGACY_NAMES[@]}"; do
    [ -f "$DIR/$f" ] || continue
    backup_existing "$f"
  done
fi
[ -f "$ENV_FILE" ] && backup_existing ".env"

mv "$tmp_compose" "$COMPOSE_FILE"
mv "$tmp_env" "$ENV_FILE"
chmod 640 "$COMPOSE_FILE" 2>/dev/null || true
chmod 600 "$ENV_FILE" 2>/dev/null || true
ok "已写入 $COMPOSE_FILE 与 .env（口令已生成或沿用）"

log "== new-api-own compose 部署 =="
log "  项目目录:   $DIR"
log "  项目名:     $PROJECT（决定 pg_data 卷名：${PROJECT}_pg_data）"
log "  镜像:       ghcr.io/zhemed/new-api-own:$TAG"
log "  容器名:     ${PREFIX}new-api / ${PREFIX}redis / ${PREFIX}postgres"
log "  网络:       host（new-api 监听 *:3000）"
log "  数据:       $DIR/data + $DIR/logs，Postgres 在命名卷 ${PROJECT}_pg_data"

if [ "$START" != 1 ]; then
  log ""
  log "（--no-start）下一步：cd $DIR && docker compose up -d"
  exit 0
fi

# ---------- 6. 容器名冲突预检（必须在 up 之前，起一半再报 Conflict 极难收拾）----------
# 必须用 `docker container inspect`：`docker inspect` 同时匹配容器与**镜像**，而我们的容器名
# redis / postgres 与 compose 自己要拉的镜像同名 —— 用 `docker inspect` 会把本地镜像当成容器，
# 于是镜像一旦存在就永远误报"已有同名容器"，脚本从第二次起不可再运行（2026-09-18 实装时踩到）。
for c in "${PREFIX}new-api" "${PREFIX}redis" "${PREFIX}postgres"; do
  docker container inspect "$c" >/dev/null 2>&1 || continue
  existing_dir="$(docker container inspect --format '{{index .Config.Labels "com.docker.compose.project.working_dir"}}' "$c" 2>/dev/null || true)"
  [ "$existing_dir" = "$DIR" ] && continue
  die "已有同名容器 \"$c\"（项目目录：${existing_dir:-未知/非 compose}）。
      解法：加 --name-prefix <前缀> 让三容器与既有部署共存，或先自行处理该容器：
        docker container inspect $c --format '{{.Config.Image}}'
        docker rm -f $c        # 仅在确认它不再需要时执行"
done

# ---------- 7. 启动并等健康 ----------
cd "$DIR"
log "拉取镜像..."
docker compose pull || die "拉取镜像失败：检查网络与 ghcr.io 可达性（公开镜像无需登录）"
log "启动容器..."
docker compose up -d || die "compose up 失败：cd $DIR && docker compose logs"

log "等待健康检查通过（上限 ${HEALTH_TIMEOUT}s）..."
status=""
for _ in $(seq 1 $((HEALTH_TIMEOUT / 2))); do
  status="$(docker container inspect --format '{{.State.Health.Status}}' "${PREFIX}new-api" 2>/dev/null || echo starting)"
  [ "$status" = "healthy" ] && break
  sleep 2
done

chmod 600 "$DIR/data"/*.db "$DIR/logs"/* 2>/dev/null || true

if [ "$status" = "healthy" ]; then
  ok "new-api 健康（healthy）"
else
  warn "健康检查尚未通过（status=$status）。下一步：cd $DIR && docker compose logs new-api --tail=100"
fi

cat <<EOF

------------------------------------------------------------------------
部署完成
  面板地址:   http://<主机>:3000   （首次访问完成初始化）
  项目目录:   $DIR
  数据目录:   $DIR/data 与 $DIR/logs（含明文上游 Key，权限已收紧为 700）
  数据库:     命名卷 ${PROJECT}_pg_data
  常用命令:   cd $DIR && docker compose logs -f / ps / restart / down

升级（钉 tag 部署，回滚有落点）:
  cd $DIR && docker compose pull && docker compose up -d
  换版本:     改 .env 的 NEW_API_IMAGE_TAG，或重跑本脚本加 --tag vX.Y.Z
  回滚:       把 NEW_API_IMAGE_TAG 改回旧版本后 pull + up -d（数据在卷里，不受影响）

安全基线（MAINTENANCE.md「部署安全基线」，本脚本不自动改网络形态）:
  * new-api 用 host 网络，监听 *:3000，/api/status 无需认证即可读；反代不会关掉这个直连端口。
    加固二选一：① nft 拦 3000 入站；② 改绑 127.0.0.1:3000 走反代（需同时把 redis/postgres
    迁到容器网络并改内部地址，属独立改动）。
  * data/ 与 logs/ 已置 700；备份文件同样含密钥，复制出去时同样要收紧权限。
  * 容器仍以 root 运行（Dockerfile 未设 USER），沿用现状。
------------------------------------------------------------------------
EOF
