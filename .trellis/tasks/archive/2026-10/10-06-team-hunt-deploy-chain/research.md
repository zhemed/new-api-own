# 研究：交付链失效模式（只读调查）

> 任务 `10-06-team-hunt-deploy-chain` ／ 共享任务 `task-11` ／ 结论日期 2026-10-06
> 环境：Docker 29.7.2（符合项目标准）、buildx v0.37.1。全程只读，未 push、未 rm/rmi、未触碰 `.local-instance/`。

## 0. 结论

用户停在 `0.5` **最可能是客户端陈旧标签组合**，不是 registry 故障：

> 本地 `:latest` 陈旧 → `docker run :latest` 默认 `--pull=missing` **不联网**（本地有就用）→
> 容器把镜像**按内容 ID 固定** → `--restart always` 只重启不换镜像 → 永久停在旧版本。

registry 侧**当前健康**：`latest` = `0.0.6` = `v0.0.6` = `sha256:a74d0a6c…`（同一 index）。

## 1. 链路图

```
git tag v0.0.6 / workflow_dispatch
        │
        ▼  .github/workflows/docker-build.yml
   ┌──────────────────────────────────────────────┐
   │ build_single_arch  (matrix, fail-fast: false)│
   │   amd64 @ubuntu-latest   arm64 @ubuntu-24.04-arm │
   │   push:  :<TAG>-{arch}  :<ALIAS>-{arch}      │  ← 全局可变、每次构建都覆盖
   │          :latest-{arch}                       │  ← 同上，latest 的唯一来源
   │   + provenance(max) + sbom + cosign sign      │
   └──────────────────────────────────────────────┘
        │ needs (成功才继续；任一矩阵腿失败=整 job 失败)
        ▼
   ┌──────────────────────────────────────────────┐
   │ create_manifests                             │
   │   1) imagetools create -t :<TAG>             │  ← 步骤顺序：版本 tag 先
   │   2) imagetools create -t :latest            │  ← latest 后
   │   3) imagetools create -t :<ALIAS>（可跳过）  │
   │   4) cosign sign 三个 tag                     │
   └──────────────────────────────────────────────┘
        │
        ▼  registry：多架构 OCI index（amd64 + arm64 + 2 个 attestation）
   :v0.0.6 ─┐
   :0.0.6  ─┼─→ sha256:a74d0a6c…（同一 index）
   :latest ─┘
        │
        ▼  部署机： docker pull（唯一会联网比对的命令）
   本地标签（可陈旧）
        │
        ▼   docker run  ← 默认 --pull=missing：本地有标签就不联网
   容器（把镜像 ID 固化进 Config.Image/Image）
        │
        ▼   --restart always：只重启同一个容器
   运行实例 ← 版本权威只存在于「容器创建时那个镜像 ID」
```

## 2. 实测证据（本机，可复现）

探针：`docker tag hello-world:latest ghcr.io/zhemed/new-api-own:zz-local-stale-probe`
（该 tag **在 registry 不存在**，所以"报 not found"就是"确实联网了"的探测器）

| 实验 | 命令 | 退出码 | 判读 |
|---|---|---|---|
| EXP1 | `docker run --rm <probe>` | **0** | 默认=missing，本地有 → **不联网**，用旧镜像 |
| EXP2 | `docker run --rm --pull=never <probe>` | **0** | 明确禁止拉取，成功 |
| EXP5 | `docker run --rm --pull=missing <probe>` | **0** | 与默认一致 |
| EXP3 | `docker run --rm --pull=always <probe>` | **125** | `not found` → **确实联网** |
| EXP4 | `docker pull <probe>` | **1** | `not found` → `docker pull` **总是联网** |
| 对照 | `docker pull ghcr.io/…:latest`（已最新） | 0 | `Status: Image is up to date` → pull 会比对远端 |

**结论**：`docker run` 在本地标签存在时（哪怕陈旧）**完全不访问 registry**；只有 `docker pull`
或 `--pull=always` 才比对远端。**`--pull=never` 与默认 `missing` 在"本地有标签"时行为相同。**

容器按 ID 固定（`docker inspect litepan`）：
```
Config.Image(tag) = ghcr.io/zhemed/litepan:v0.0.49
Image(pinned ID)  = sha256:6ddd6e7973da665109eedb3b2588df1a032b4d14407470f04e25a1b27c7d14a4
RestartPolicy     = unless-stopped
```
→ 容器记录的是**tag 字符串（仅供显示）**与**不可变镜像 ID（真正执行的）**。pull 新镜像、
`docker restart`、`--restart always` 都不会改变后者；**必须 `docker rm` + `docker run`**。

镜像标签证据：
| tag | `org.opencontainers.image.version` | created |
|---|---|---|
| 0.0.2 | `v0.0.2` | 2026-09-18T01:07Z |
| v0.0.3 / 0.0.3 | **`main`** | 2026-09-18T07:29Z |
| v0.0.4 | `v0.0.4` | 2026-09-20T01:14Z |
| latest / 0.0.6 | `v0.0.6` | 2026-10-06T12:32Z |

`0.0.3` 的 label 是 `main` 而**不是** tag → `docker/metadata-action` 从分支 ref 取值 →
**该次构建走的是 `workflow_dispatch`，不是 tag push**。这是"历史 run 里确实用过手动触发"的实证。

## 3. 失效模式表

| # | 失效模式 | 判定 | 会不会真的让用户停在旧版 | 证据 |
|---|---|---|---|---|
| 1 | 本地 `:latest` 陈旧 + `docker run` 默认不拉 | **真实（主因）** | **会**。本地标签一旦陈旧，重跑部署命令永远跑旧镜像 | EXP1/EXP3/EXP4；本机 `latest` 曾=0.0.3（已确认事实） |
| 2 | `--restart always` / `docker restart` ≠ 换镜像 | **真实** | **会**。pull 完不 rm+run，容器仍固定旧 ID | `docker inspect` 的 `Image`=sha256 ID |
| 3 | 矩阵部分成功 → 推了镜像但 `create_manifests` 被跳过 | **真实** | **会**。`:latest` 不更新（停旧版），但 `:latest-{arch}` 已被覆盖成新版 | `fail-fast: false` + `needs: [build_single_arch]`（默认 success()）；cosign 步骤失败也会触发 |
| 4 | 上述残留态再"Re-run failed jobs" → 拼出**混合架构清单** | **真实** | **会**。amd64=新 + arm64=旧，pull 成功但某个架构版本静默错误 | `imagetools create` 只做拼接、不校验两端同版本 |
| 5 | `:latest-{arch}` 是全局可变标签 → 任何构建都回退 `:latest` | **真实** | **会**。`workflow_dispatch` 重建旧 tag 会让 `:latest` 静默**回退** | 步骤 2 用 `latest-{arch}`；0.0.3 label=`main` 证明手动触发被用过；本机 `latest` 曾=0.0.3 |
| 6 | 非版本 tag 也会发布 | **理论** | 会（若推送） | 触发条件 `tags: '*'`（仅排除 `nightly*`）；本机存在 tag `pre-rollback-backup-20260921` |
| 7 | 重建时相对卷路径 / 环境变量丢失 | **真实** | 间接：数据目录变空、`-e` 丢失 → 表现为"升级后配置没了" | README 升级三步用 `-v ./data:/data`（相对路径）；顶部部署命令无 `-e` |
| 8 | registry 拉取鉴权失败 | **非风险** | 不会 | 匿名只读可用：探针返回 `not found` 而非 401/403 |
| 9 | 别名标签漂移（latest vs vX.Y.Z） | **当前非风险** | 不会 | `latest`/`0.0.6`/`v0.0.6` 同一 index；`0.0.4`=`v0.0.4`=d159a4df；`0.0.5`=`v0.0.5`=79f4bdcb |
| 10 | 多架构结构本身 | **非风险** | 不会 | index 含 amd64 `e0ad97dd` + arm64 `a8b41eae` + 2 attestation；`latest-amd64`(3fe08818) 是 provenance index，其子 `e0ad97dd` 与 `latest` 内一致 → **无漂移** |

## 4. 复现命令

```bash
# 远端 vs 本地标签是否一致
docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:0.0.6 --format '{{.Manifest.Digest}}'
docker images --digests | grep new-api-own

# 容器到底在跑哪个镜像 ID（而不是哪个 tag）
docker inspect <容器名> --format '{{.Image}} {{.Config.Image}}'

# 拉取语义自证（探针 tag 在 registry 不存在）
docker tag hello-world:latest ghcr.io/zhemed/new-api-own:zz-local-stale-probe
docker run --rm            ghcr.io/zhemed/new-api-own:zz-local-stale-probe   # exit 0，不联网
docker run --rm --pull=always ghcr.io/zhemed/new-api-own:zz-local-stale-probe # exit 125, not found
docker pull                ghcr.io/zhemed/new-api-own:zz-local-stale-probe   # exit 1,  not found
```

## 5. 诚实边界

- **未验证 GitHub Actions run 历史**：本任务只获授权对 `ghcr.io/zhemed/new-api-own` 做只读查询，
  没有 GitHub API 访问授权。第 3/4/5 条是基于 workflow 配置的**结构性论证** + `main` label 实证，
  而非 run 日志实证。要坐实需查 `Publish Docker image (Multi-arch)` 的历史 run。
- **未验证用户远程实例**（硬约束禁止）。"0.5" 的现场确认需用户在自己的部署机上执行第 4 节命令。

## 6. 遗留物

本机新增了一个**探针标签**（不是新镜像，仅给 `hello-world:latest` 加别名）：

```
ghcr.io/zhemed/new-api-own:zz-local-stale-probe -> sha256:5e2309035332…
```

按硬约束「不要 docker rm/rmi 任何东西」**未清理**。如需清理：
`docker rmi ghcr.io/zhemed/new-api-own:zz-local-stale-probe`（仅解除该别名，`hello-world` 本体保留）。
