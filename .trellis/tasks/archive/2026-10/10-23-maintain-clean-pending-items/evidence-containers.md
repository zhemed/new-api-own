# P5 保留期证据：删除两个已停回滚容器 prev-v0044 / prev-v0043

- 任务：Trellis `10-23-maintain-clean-pending-items`（PRD 第 32-34 行 P5 保留期裁定「两个都删」）；共享任务板 `task-4`
- 执行者：worker-containers；执行时间：2026-10-23T16:00Z（UTC）
- 事前依据：PRD `prd.md:32-34`「P5 回滚容器保留期：裁定『两个都删』」
- 写作用域：本文件（唯一被写的路径）
- 红线遵守：无网络动作、未提交、未改仓库其它文件；只执行 `docker ps` / `docker inspect` / `docker rm`（本机 Docker 29.7.2 + Compose v5.4.0，已核达标）

## 1. 环境核验（执行 docker 操作的前置条件）

```console
$ docker version --format 'Client={{.Client.Version}} Server={{.Server.Version}}'
Client=29.7.2 Server=29.7.2

$ docker compose version | head -1
Docker Compose version v5.4.0
```

与 AGENTS.md「Docker 环境标准（29.7.2 + v5.4.0）」一致，达标。

## 2. 删前现状记录

```console
$ docker ps -a --format '{{.Names}} | {{.Image}} | {{.Status}}'
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up About an hour
new-api-prev-v0044 | ghcr.io/zhemed/new-api-own:v0.0.44 | Exited (0) About an hour ago
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 6 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 4 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 3 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 3 days
new-api-prev-v0043 | ghcr.io/zhemed/new-api-own:0.0.9 | Exited (0) 3 hours ago
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 3 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 3 days (healthy)
halodb | postgres:15.4 | Up 3 days (healthy)
```

容器总数：10（在跑 8 + 已停 2）。

两个目标容器的删前状态（单独复核，确认均为已停）：

```console
$ docker ps -a --filter name=^new-api-prev-v0044$ --filter name=^new-api-prev-v0043$ \
    --format '{{.Names}} | {{.Image}} | {{.Status}}'
new-api-prev-v0044 | ghcr.io/zhemed/new-api-own:v0.0.44 | Exited (0) About an hour ago
new-api-prev-v0043 | ghcr.io/zhemed/new-api-own:0.0.9 | Exited (0) 3 hours ago
```

两个都处于 `Exited (0)`，无需 `-f`，未触发强制删除路径。

## 3. 执行删除

```console
$ docker rm new-api-prev-v0044 new-api-prev-v0043
new-api-prev-v0044
new-api-prev-v0043
exit=0
```

命令仅含两个点名容器名，未使用 `-f`、未加 `-v`（不删匿名卷）、未涉及镜像与其它容器。

## 4. 删后验证

### 4.1 prev 容器已不存在

```console
$ docker ps -a --format '{{.Names}} | {{.Image}} | {{.Status}}' | grep -i 'prev'
(无输出)
grep exit=1   # 1 = 无匹配，即无残留
```

### 4.2 全量容器清单（删后）

```console
$ docker ps -a --format '{{.Names}} | {{.Image}} | {{.Status}}'
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up About an hour
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 6 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 4 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 3 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 3 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 3 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 3 days (healthy)
halodb | postgres:15.4 | Up 3 days (healthy)
```

容器总数：10 → 8，减少的正好是两个 prev 容器；其它 8 个容器全部保持原状态（在跑）。

### 4.3 在跑实例未受影响

```console
$ docker ps --format '{{.Names}} | {{.Image}} | {{.Status}}'
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up About an hour
komari-agent-test | ghcr.io/zhemed/komari-agent:latest | Up 6 hours
komari-test | ghcr.io/zhemed/komari:latest | Up 4 hours
picsur | ghcr.io/zhemed/picsur:allinone | Up 3 days (healthy)
support-devdb-1 | postgres:17-alpine | Up 3 days
litepan | ghcr.io/zhemed/litepan:v0.0.49 | Up 3 days
halo | ghcr.io/zhemed/halo:v2.26.1-custom.1 | Up 3 days (healthy)
halodb | postgres:15.4 | Up 3 days (healthy)

$ docker inspect -f '{{.State.Status}}' new-api
running
```

`new-api`（v0.0.45）仍 Up，`State.Status = running`，删除操作没有触碰在跑实例。

### 4.4 镜像未被删除（对照约束「不要删镜像」）

```console
$ docker images --format '{{.Repository}}:{{.Tag}} | {{.ID}} | {{.Size}}' | grep -i 'new-api-own'
ghcr.io/zhemed/new-api-own:v0.0.45 | e114b17532f1 | 302MB
ghcr.io/zhemed/new-api-own:v0.0.44 | 92d066b505ae | 302MB
ghcr.io/zhemed/new-api-own:0.0.9  | f326deaef922 | 302MB
ghcr.io/zhemed/new-api-own:0.0.8  | fdb3883105ce | 302MB
ghcr.io/zhemed/new-api-own:0.0.7  | 68c9842d16fb | 302MB
ghcr.io/zhemed/new-api-own:latest | 68c9842d16fb | 302MB
ghcr.io/zhemed/new-api-own:v0.0.7 | 68c9842d16fb | 302MB
ghcr.io/zhemed/new-api-own:0.0.6  | a74d0a6c0940 | 302MB
ghcr.io/zhemed/new-api-own:0.0.5  | 79f4bdcb712f | 302MB
ghcr.io/zhemed/new-api-own:v0.0.4 | e850cb5dc8b0 | 302MB
ghcr.io/zhemed/new-api-own:0.0.3  | 3d04916fe29a | 302MB
ghcr.io/zhemed/new-api-own:v0.0.3 | 3d04916fe29a | 302MB
ghcr.io/zhemed/new-api-own:0.0.2  | e16437fdc39f | 302MB
ghcr.io/zhemed/new-api-own:v0.0.2 | e16437fdc39f | 302MB
```

被删容器原本使用的两个镜像 `v0.0.44`、`0.0.9` 仍在本地（回滚镜像本体保留，只删了容器外壳）。

### 4.5 数据卷未被删除（对照约束「不要删数据卷」）

```console
$ docker volume ls -q | wc -l
40
```

卷总数 40；`docker rm` 未带 `-v`，未删除任何匿名卷，具名卷（`halo-gradle-cache`、
`picsur_picsur-data`、`support_db-data` 等）与其余 37 个卷均在列。

## 5. 结论

两个指定回滚容器已删除，删除范围严格限定；在跑 `new-api`（v0.0.45）状态 running，
镜像、卷、其它容器零改动。环境与验收条件均满足。
