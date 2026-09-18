# 在 目标主机（名称不记录） 部署 new-api 网关

## 勘察结果（只读，2026-09-18）

- 主机：Debian 12 / 内核 6.1.0-50 / x86_64；**Docker 29.7.2 + Compose v5.4.0**（符合项目强制标准）
- 容器：仅 `new-api`（镜像 ID `094bd74d8d8b`，即 v0.0.2 那份），Up 2 分钟，host 网络，3000 已监听
- 数据：仅 `/root/data`（当天新建的 SQLite）；**无 compose 文件、无 Postgres/Redis 容器**
- 结论：这是一台**单容器 SQLite 部署**，不存在 Postgres 数据；"部署"= 换镜像 + 重建容器，数据与形态保持不变

## 目标

把该机 `new-api` 从旧镜像（v0.0.2）升级到 registry 当前的 `:latest`（= v0.0.3），部署形态与数据目录不变。

## 执行步骤（严格按此，不做多余动作）

1. 该机执行 `docker pull ghcr.io/zhemed/new-api-own:latest`（registry 的 `latest` 已核验为 v0.0.3）
2. `docker rm -f new-api`
3. `docker run -d --name new-api --restart always --network host -v /root/data:/data ghcr.io/zhemed/new-api-own:latest`
   （与用户原命令一致，只把 `./data` 写成绝对路径 `/root/data`）
4. 删除被替换下来的旧镜像（按 ID，先确认无容器引用）
5. 只读核验：容器内 `--version`、`/api/status` 的 version、容器状态、`/root/data` 仍存在

## 不做

- 不装 compose 文件、不引入 Postgres/Redis、不改数据目录与权限
- 不加用户原命令里没有的环境变量（TZ 等如需另说）
- 不动该机其它任何容器 / 配置 / 文件

## 验收标准

- [x] 容器 `new-api` 运行中，镜像为 `sha256:3d04916fe29a…`（= v0.0.3 那份）
- [x] `docker exec new-api /new-api --version` → `v0.0.3`
- [x] `/api/status` 返回 `"version":"v0.0.3"`、`"setup":true`，3000 端口可访问
- [x] `/root/data` 保持存在、未被迁移或覆盖（`backup/`、`logs/`、`one-api.db` 约 7 MB 原样）
- [x] 旧的 v0.0.2 镜像已不在该机；另有一份 6 天前的 `0.0.1` 镜像保留未动（不属于本次替换）

## 结果

- `docker pull …:latest` → 该机已是重建后的镜像（`3d04916fe29a…`）
- 重建容器后：`RestartCount=0`、`Running=true`，日志 `New API v0.0.3 ready in 653 ms`
- 日志含常规提示：`Refresh cookie is not secure … set SESSION_COOKIE_SECURE=true in production`（该机未设该变量，属既有状态，未改动）
- 凭据：仅用于本次 SSH 连接，未写入任何文件；**建议用户轮换该密码**

## 凭据处置

用户提供的密码**不写入任何文件**，仅在 SSH 连接命令中以环境变量传入一次；完成后提醒用户轮换该密码。
