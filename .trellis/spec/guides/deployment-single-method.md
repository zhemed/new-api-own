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
