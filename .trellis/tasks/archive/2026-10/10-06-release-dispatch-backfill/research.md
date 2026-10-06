# 证据：release.yml 手动补发通道 + 并行风险注释

任务 `10-06-release-dispatch-backfill`。仅改 `.github/workflows/release.yml`，未 push、未触发工作流。

## R1 改动内容

`on:` 新增手动补发通道（**同一个 job、同一段脚本**，无重复逻辑）：

```yaml
  workflow_dispatch:
    inputs:
      tag:
        description: 'Version tag to publish a Release for (e.g. v0.0.6). Must already exist and match v[0-9]* or [0-9]*.'
        required: true
        type: string
```

job 仍只有一个 `release`，步骤收敛为三步，两条入口共用：

| 步骤 | 职责 |
|---|---|
| `Check out` (`fetch-depth: 0`) | 取全部 tag（判"最高版本"与校验 tag 存在都要） |
| **`Resolve and validate tag`**（两条入口共用） | 解析 tag（`workflow_dispatch` → 输入；`push` → `GITHUB_REF`），强制版本 tag 契约，校验 tag 真实存在，输出 `steps.tag.outputs.tag` |
| **`Publish release`**（两条入口共用） | `HIGHEST` 判定 → `--latest`/`--latest=false` → 带 `-` 走 `--prerelease` → `gh release view`→`edit` 幂等 → 结果写 run summary |

**脚本注入防护**：`github.event.inputs.tag` 经 `env: INPUT_TAG` 传入，脚本里只用 `$INPUT_TAG`，
不做 `${{ }}` 直接内插（dispatch 输入是调用方可控文本）。

## R2 并行风险注释（已写入文件头）

```
# KNOWN PARALLEL RISK — deliberate, not an oversight:
# This workflow and docker-build.yml both fire on the same `on: push: tags`
# event and therefore run concurrently. If the image push fails while this
# Release succeeds, the panel will advertise a version that cannot actually be
# pulled. Closing that gap completely requires chaining on the image workflow
# (`on: workflow_run: workflows: ["Publish Docker image (Multi-arch)"] ...`) so
# the Release is only created after a successful push. This revision keeps
# `on: push` on purpose, to avoid coupling the two workflows together; revisit
# if a release ever ships without its image.
```

## R3 校验输出

```
=== YAML + structure ===
release.yml parse OK; docker-build.yml parse OK
triggers            : ['push', 'workflow_dispatch']
tags == docker-build: True ['v[0-9]*', '[0-9]*']
dispatch tag input  : required=True type=string
one job / steps     : ['Check out', 'Resolve and validate tag', 'Publish release']
permissions         : {'contents': 'write'}
```

**非法 tag 拒绝（执行的是文件里那段真实脚本，不是重写的副本）：**

```
$ INPUT_TAG=release-notes EVENT_NAME=workflow_dispatch bash -c "<Resolve and validate tag 脚本>"
::error::refusing to publish 'release-notes' (from the workflow_dispatch 'tag' input): not a version tag.
Expected v<digits>... or <digits>..., for example v0.0.6 or 0.0.6. Tags such as 'release-notes',
'nightly-2026' or 'pre-rollback-backup-20260921' are rejected on purpose: publishing one would move
the repository 'latest' flag, which is what the panel reads.
exit=1
```

**完整判定矩阵（同样执行真实脚本）：**

| 场景 | 结果 |
|---|---|
| dispatch `release-notes` | exit=1，`not a version tag` + 明确示例 |
| dispatch 空输入 | exit=1，`no tag resolved ... set the required 'tag' input` |
| dispatch `v9.9.9`（格式合法但 tag 不存在） | exit=1，`tag 'v9.9.9' does not exist ... this workflow never creates tags` |
| dispatch `v0.0.6` | exit=0，`resolved tag 'v0.0.6' from the workflow_dispatch 'tag' input` |
| push `refs/tags/v0.0.6` | exit=0，`resolved tag 'v0.0.6' from the pushed tag` |

**latest / prerelease 决策（真实脚本 + 本仓库真实 tag 列表，`HIGHEST=v0.0.6`）：**

```
TAG=v0.0.6           -> --latest         <none>
TAG=0.0.6            -> --latest         <none>
TAG=v0.0.5           -> --latest=false   <none>
TAG=v0.0.7-alpha.1   -> --latest=false   --prerelease
```

## 自查中发现并修掉的一处缺陷

首轮矩阵暴露出：`TAG=0.0.6` 会判成 `--latest=false`，因为当时比较的是
`"$TAG" = "$HIGHEST"`（`0.0.6` ≠ `v0.0.6`）。本项目**同时发布两种拼写**，
所以改成比较去掉 `v` 前缀后的版本号：

```bash
HIGHEST_VERSION="${HIGHEST#v}"
if [ -z "$PRERELEASE_FLAG" ] && [ "$VERSION" = "$HIGHEST_VERSION" ]; then
  LATEST_FLAG="--latest"
fi
```

修后 `v0.0.6` 与 `0.0.6` 都拿到 `--latest`。

## 范围与未做

- 本任务只改 `release.yml`；`docker-build.yml` 的 ` M` 是**上一轮**已授权改动的残留，本轮未再动它。
- 项目闸门 `scripts/forbid-extra-deploy-methods.sh` → `✅ 部署方式唯一`。
- **未做（等用户定）**：清理旧 `v0.0.5` Release 上的 ~118MB 二进制；存量 `v0.0.6` 的**实际**补发
  （前置条件：本文件先落到默认分支，GitHub 才会出现 "Run workflow" 按钮；且需用户同意提交推送）。

## 补发操作（授权后，供 Lead/用户直接使用）

```
gh workflow run "Publish release (image-only)" -f tag=v0.0.6
# 或 GitHub UI: Actions → Publish release (image-only) → Run workflow → tag = v0.0.6
# 结果核对：curl -s https://api.github.com/repos/zhemed/new-api-own/releases/latest | grep tag_name
#          应返回 v0.0.6
```
