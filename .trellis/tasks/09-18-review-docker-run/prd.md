# 核对 docker run 部署命令

## 结论

**命令能启动容器，但不等于现有部署**。最要紧一条：**没有 `SQL_DSN` 会退回 SQLite**
（`model/main.go:148` 打印 `SQL_DSN not set, using SQLite as database`，库文件为 WORKDIR `/data` 下的
`one-api.db`）。线上 compose 用的是 Postgres，因此这条命令会用**另一个库**启动 —— 面板看起来"数据全丢"。

## 逐条核对（对照 `docker-compose.yml` / `Dockerfile` / 代码默认值）

| 项 | 命令现状 | 后果 | 建议 |
|---|---|---|---|
| 数据库 | 无 `SQL_DSN` | 回退 SQLite `/data/one-api.db` | 加 `-e SQL_DSN='postgresql://…@127.0.0.1:5432/new-api'`；若确实要 SQLite 则保持 |
| 缓存 | 无 `REDIS_CONN_STRING` | 退回内存缓存（重启即失效；多节点不共享） | 加 `-e REDIS_CONN_STRING='redis://…'` |
| 时区 | 无 `TZ` | 容器 UTC，日志/面板时间差 8 小时 | `-e TZ=Asia/Shanghai` |
| 日志目录 | 未传 `--log-dir` | 默认 `./logs`（WORKDIR=`/data`）→ 落在 `/data/logs`，混进数据卷 | 保持（简单）或 `--log-dir /app/logs` + 挂 `./logs:/app/logs` |
| 特性开关 | 未传 | `LOGIN_SESSION_NEVER_EXPIRES`（`common/init.go:144`）、`ERROR_LOG_ENABLED`（`:202`）、`BATCH_UPDATE_ENABLED`（`main.go:154`）、`NODE_NAME`（`common/node_identity.go:13`）全走默认 | 按 compose 补上 |
| 限流 | 未传 | 0.0.2 起代码默认已关闭；**旧镜像仍默认开启** | 显式传四个 `*=false`，或用 v0.0.2+ 镜像 |
| 健康检查 | 无 | 容器不会被标记 unhealthy（`--restart always` 只在进程退出时重启） | 加 `--health-cmd/interval/timeout/retries`（镜像内有 wget，`Dockerfile:34`） |
| 镜像标签 | `latest` | 可复现性差；`docker run` 不会自动重新拉取，会用本地缓存 | 建议钉 `:v0.0.3`，或先 `docker pull` |
| 卷路径 | `./data` 相对路径 | 换 cwd 执行会挂到别的目录 → 看似"数据丢了" | 用绝对路径 |
| 容器名 | `--name new-api` | 与既有同名容器（compose 起的）冲突会直接报错 | 先 `docker rm -f new-api` |
| 网络 | `--network host` | 与 compose 一致；但按安全基线面板会监听 `*:3000` 暴露到所有网卡 | 保持 host 网络就必须配 nft 规则（见「部署安全基线」） |
| 数据权限 | 挂载 `data` | 卷内是明文上游 Key | `chmod 700 data data/logs && chmod 600 data/*.db data/logs/*` |

## 建议命令（与 compose 等价）

```bash
cd /root/new-api-own
docker rm -f new-api 2>/dev/null || true
docker pull ghcr.io/zhemed/new-api-own:v0.0.3

docker run -d --name new-api --restart always \
  --network host \
  -v /root/new-api-own/data:/data \
  -e TZ=Asia/Shanghai \
  -e SQL_DSN='postgresql://root:123456@127.0.0.1:5432/new-api' \
  -e REDIS_CONN_STRING='redis://:123456@127.0.0.1:6379' \
  -e LOGIN_SESSION_NEVER_EXPIRES=true \
  -e ERROR_LOG_ENABLED=true \
  -e BATCH_UPDATE_ENABLED=true \
  -e NODE_NAME=new-api-node-1 \
  -e GLOBAL_WEB_RATE_LIMIT_ENABLE=false \
  -e GLOBAL_API_RATE_LIMIT_ENABLE=false \
  -e CRITICAL_RATE_LIMIT_ENABLE=false \
  -e SEARCH_RATE_LIMIT_ENABLE=false \
  --health-cmd "wget -q -O - http://localhost:3000/api/status | grep -o '\"success\":\\s*true' || exit 1" \
  --health-interval 30s --health-timeout 10s --health-retries 3 \
  ghcr.io/zhemed/new-api-own:v0.0.3
```

最小可用版（单机 SQLite、日志进 data/）：

```bash
docker run -d --name new-api --restart always --network host \
  -v /root/new-api-own/data:/data -e TZ=Asia/Shanghai \
  ghcr.io/zhemed/new-api-own:v0.0.3
```

> 升级既有部署更推荐走 compose：`docker compose pull && docker compose up -d`
> （compose 已含全部环境变量、健康检查与依赖服务定义）。

## 盲区

本机没有部署机（内网主机）的 SSH 私钥，无法确认那台机器上 Postgres / Redis 是否在跑、
是否已有同名容器；上面关于"库不同"的结论基于仓库配置与代码默认值，不是对线上状态的实测。

## Acceptance Criteria

- [x] 逐条比对命令与 compose/Dockerfile/代码默认值
- [x] 给出可直接执行的修正命令与最小可用版
- [x] 明确未验证部分（部署机状态、Postgres/Redis 是否在运行）
