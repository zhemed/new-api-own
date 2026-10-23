# 唯一部署方式（强制）

> 用户 2026-10-06 定调：「我们只有一种部署方式，也是唯一一种，必须强制约束了。」
> 本文件是这条约束的规则源；配套闸门脚本 `scripts/forbid-extra-deploy-methods.sh`。

## 规则

**本项目只有一种部署方式：`docker run` + 公开镜像 `ghcr.io/zhemed/new-api-own`。**

```bash
docker run -d --name new-api --restart always \
  --network host \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:latest
```

- 唯一形态：单容器 + `docker run`（`--network host`，数据挂 `./data:/data`）。
- 数据：默认 SQLite，落在 `./data`。
- 上面四条 `LOG_*` 与 `--log-opt` 是**当前在用的日志形态**（内存用量日志 + Docker 日志轮转上限），
  与 README / MAINTENANCE 的部署、升级、回滚命令**完全一致**；改这一组参数时三处必须同步。
- 自建镜像（`docker build -t new-api-own .`）后仍是**同一个** `docker run` 形态，
  不算第二种部署方式；**第二种编排形态**才算。

## 禁止

- `docker-compose*.yml` / `compose.y*ml`：**曾两删三回**，见下"事故记录"；
- Helm chart、K8s/Kustomize 清单（`charts/`、`helm/`、`k8s/`、`kubernetes/`、`manifests/`）；
- 任何带 compose 特征字段（`services:`、`network_mode:`）的 YAML（改名换壳也算）；
- **裸机 / systemd 形态**：根级 `*.service` / `*.timer` / `*.socket`，或 `systemd/`、`deploy/`、`etc/`
  目录下的 unit 文件（2026-10-06 用户定调："一并删掉"）。

要新增部署形态，必须**先改本规则并取得仓库所有者明确同意**，再动文件。

## 强制手段（三层，与 Trellis 闸门同构）

| 层 | 位置 | 作用 |
|---|---|---|
| 提交当场 | `.githooks/pre-commit` | 暂存区里出现 compose 形态 → 直接拒绝提交 |
| 本地自检 | `scripts/forbid-extra-deploy-methods.sh` | 按文件名 + 目录名 + 文件内容三种方式判定 |
| 远程兜底 | `.github/workflows/trellis-gate.yml` | CI 上再跑一遍，`--no-verify` 绕过的也会红 |

## do-not-restore 清单（回滚后必须核验）

被用户点名移除、**任何回滚都不得带回**的交付物：

| 交付物 | 移除时间 | 依据 |
|---|---|---|
| `docker-compose.yml` | 2026-09-20（`843f988`） | "移除生产 compose 编排，改 docker run" |
| `docker-compose.dev.yml` | 2026-10-06 | 同一条"唯一部署方式"约束 |
| `new-api.service`（systemd unit）| 2026-10-06 | 裸机/systemd 视为第二种部署方式，闸门一并拦 |
| `v0.0.4`–`v0.0.8` 的镜像版本与 tag（旧批次）| 2026-10-06 | 用户要求的全套回滚（此条为历史批次，勿与后续同名 tag 混淆）|

**新增条目**：任何被用户点名"删掉/不要了"的东西，当轮就要登记进本表。

## 回滚核验（强制动作）

**每次执行回滚（`git reset --hard`、`git revert`、强推、恢复备份 tag）之后，必须做两件事**：

1. 对照本表逐项核验：**被点名移除的交付物有没有被带回来**（`git ls-files | grep -i compose` 一条命令即可）；
2. 把核验结果写进当轮汇报——**回滚不等于可以把手点删过的东西还原**。

## 事故记录（为什么要有这道闸门）

`docker-compose.yml` 的完整轨迹：

| 时间 | 提交 / 任务 | 对该文件做了什么 |
|---|---|---|
| 2026-08-13 | `4a24792` | 注释里带入内网地址（后于 09-18 清理） |
| 2026-09-18 | `purge-deploy-details` / `release-v0.0.2` | 清掉注释里的内网地址 |
| 2026-09-20 | `8bcd139` `review-compose-deploy-findings` | 改名 `docker-compose.yml` → `compose.yaml` |
| 2026-09-20 | `770e26c` `revert-to-previous-deploy` | **加回来**（第 1 次被回滚带回） |
| 2026-09-20 | **`843f988` `drop-compose-and-release-004`** | **删除**（"移除生产 compose 编排，改 docker run"） |
| 2026-09-20 | `a29cd82` `rollback-to-002` | **加回来**（第 2 次被回滚带回） |
| 2026-10-06 | `rollback-to-0-0-3`（强推 main → `8a99114`，09-18 的 v0.0.3）| 09-20 的删除**不在其祖先里** → **加回来**（第 3 次） |
| 2026-10-06 | 本文件 + 闸门脚本 | 删除并加上三层拦截 |

教训：**"文档写了规则"挡不住回滚，"提交/CI 拦得住"才挡得住。**

## 数据目录必须位于工作区内（2026-10-23 定调）

> 用户 2026-10-23 定调：「实例数据目录必须在工作区内。」本机实例的数据目录已从工作区外路径迁到
> 仓库根下的 `./data`（容器挂载等价于 `-v ./data:/data`）。README / MAINTENANCE / 本文件不再引用
> 工作区外的数据目录（workspace journal 里 2026-10-23 的迁移记录作为历史保留，不受此限）。

**规则：实例数据目录一律落在仓库工作区内，按 `-v ./data:/data` 挂载；禁止挂到工作区外的绝对路径。**
容器内路径仍是 `/data`，命令形态仍是「规则」一节那一条 `docker run`——本约定不引入第二种部署方式。

- 数据：SQLite 库、应用日志、上传文件，全部在 `./data` 下。
- 忽略规则：`.gitignore:27` 的 `/data/` 已覆盖它，数据不入库。
- 与 MAINTENANCE「升级 / 回滚」的关系：那里写明「`./data` 是相对路径，换目录＝换了数据目录」，
  三处（README / MAINTENANCE / 本文件）口径必须一致。

### 理由

1. **维护动作只能在允许的范围内做。** 维护会话（含 AI agent）的读写范围只有工作区：数据在工作区内，
   文件数、sha256、SQLite 计数这类只读盘点可以直接做；放在工作区外，同一次盘点就变成越界读取——
   按全局安全红线属禁止动作，校验只能降级成"人在主机上凭记忆做"。
2. **忽略规则只覆盖工作区内。** `/data/` 是锚定仓库根的规则：放在根下，数据自动不进 Git；
   放到工作区外则既不受这条规则保护，也不随工作区备份走——容易出现"仓库很干净，数据没备份"。
3. **回滚点删除后就是单点。** 迁移完成后旧目录只是临时回滚点；一旦按"清干净"删掉，`./data` 就是唯一副本，
   此后的误挂载、误删都没有第二条退路（见下方风险表）。

### 正确迁移 / 换挂载步骤

| 步 | 动作 | 命令要点 |
|---|---|---|
| 1 | 停容器，别删 | `docker stop new-api`；容器留着可 `docker inspect` 对照原参数 |
| 2 | 复制数据 | `mkdir -p ./data && cp -a <旧目录>/. ./data/`；**不要 `mv`**，旧目录先当回滚点 |
| 3 | 校验 | 文件数 + sha256 摘要，且**必须在容器停止后**复制 |
| 4 | 删容器 | `docker rm new-api`，**不带 `-v`** |
| 5 | 按原参数重建 | 照抄「规则」一节的 `docker run`，只改 `-v` 的源路径 |
| 6 | 验证 | `docker inspect` 看挂载源 + HTTP 200 + 版本与业务计数 |
| 7 | 观察后再删旧目录 | 计数、登录、用量页都正常，才删回滚点 |

第 3 步在旧、新两个数据目录**内部**各跑一次（下面用子 shell `cd` 把这一点写死），输出必须一致：

```bash
( cd <旧数据目录> && find . -type f | wc -l ) ; ( cd ./data && find . -type f | wc -l )
( cd <旧数据目录> && find . -type f -exec sha256sum {} + | sort | sha256sum )
( cd ./data        && find . -type f -exec sha256sum {} + | sort | sha256sum )
```

在仓库根直接跑 `find . -type f` 会把整个仓库算进去，文件数与摘要都不对。

第 6 步：

```bash
docker inspect -f '{{range .Mounts}}{{.Source}}{{"\n"}}{{end}}' new-api   # 应等于 <部署目录>/data
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status  # 期望 200
curl -s http://127.0.0.1:3000/api/status | grep -o '"version":"[^"]*"'     # 实例自报版本
```

### 反例与风险

| 反例 | 后果 | 正确做法 |
|---|---|---|
| 挂工作区外的路径 | 只读盘点、凭据与数据核查都要越界，备份与迁移容易漏项 | 一律 `./data` |
| 容器运行中 `cp -a` | SQLite `-wal` / `-shm` 与主库不一致，拷出的库可能损坏或丢最近事务 | 先 `docker stop` |
| `docker rm -v` / `docker volume prune` | 本项目是 bind mount，现在无损；一旦换成命名卷就会连数据一起删，不能靠记忆区分 | 删容器一律不带 `-v` |
| 迁移后忘改 `-v`，仍挂旧路径 | 面板看起来"数据全没了"（其实在旧目录），随后误删旧目录才是真事故 | 先按第 6 步核对挂载源与计数，再动旧目录 |
| 验证没过就删回滚点 | `./data` 成单点，无退路 | 保留旧目录直到计数与页面都正常 |

> **现状（2026-10-23 起）**：迁移完成后旧目录已按裁定删除，`./data` 是唯一数据副本，且**尚无异地备份**
> ——备份策略本轮未做，属已知单点风险。凡涉及数据目录的清理/换挂载动作（`rm -rf`、卷与镜像清理、
> 改 `-v`），动手前先确认这条，并优先补备份。
