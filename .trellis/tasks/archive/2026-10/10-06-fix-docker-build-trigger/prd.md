# 修复 docker-build 触发条件并清理探针标签

## Goal

1. **修掉自己发现的真实失效模式 #6**：`.github/workflows/docker-build.yml` 的 `on.push.tags: '*'`
   让**任何** tag 都能构建并移动 `:latest`。本机存在非版本 tag `pre-rollback-backup-20260921`，
   若推送会把 `:latest` 指到非发布快照。
2. **清理本机探针标签**：`ghcr.io/zhemed/new-api-own:zz-local-stale-probe`（只读调查实验遗留的假标签）。

## Requirements

### R1 触发条件收窄（`.github/workflows/docker-build.yml`）

- 把 `on.push.tags` 从 `'*'`（+ 现已失效的 `'!nightly*'`）收窄为**只匹配版本 tag**。
- 必须**保留** workflow 既有的"带 `v` 与不带 `v` 双 tag"设计（`TAG_ALIAS` 逻辑，见步骤
  `Resolve tag & write VERSION` 与 `Create & push manifest (alias without "v")`）——
  即 `v0.0.6` 与 `0.0.6` 都应继续触发。
- 必须**排除** `pre-rollback-backup-20260921` 这类非版本 tag。
- `workflow_dispatch` 保持可用（人工发版入口）。
- 不得改动 `create_manifests` 的步骤顺序、provenance（`provenance: mode=max`）、`sbom: true`、
  cosign 签名步骤的语义。

### R2 YAML 与语义校验

- 用 `python3 -c "import yaml, ..."` 解析校验改后文件是合法 YAML。
- 贴出**触发条件前后对照**。
- 用 Python 按 GitHub Actions 的 glob 语义（`*`、`[]`、`!`）自证：新条件对
  `v0.0.6` / `0.0.6` 命中，对 `pre-rollback-backup-20260921` / `nightly-2026` 不命中。

### R3 清理探针标签

- `docker rmi ghcr.io/zhemed/new-api-own:zz-local-stale-probe`，只删该别名，不动 `hello-world` 本体。
- 证据：`docker images | grep zz-local-stale-probe` 输出为空。
- 若会连带删除不应删的对象，**先停下来说明，不硬删**。

## Constraints（硬约束）

- 只读网络：仅允许 ghcr.io 只读；**不 push、不触发 workflow**（无网络授权）；
- 不 commit / 不 push；禁止 `git add -A`；不建 worktree；
- 不碰用户远程实例；不碰 `.local-instance/`；
- 写范围仅限 `.github/workflows/docker-build.yml` + 本任务目录。

## Acceptance Criteria

- [ ] `.github/workflows/docker-build.yml` 触发条件只接受版本 tag，且解析为合法 YAML
- [ ] 版本 tag（含带/不带 `v`）命中、非版本 tag 不命中的判定有可复现输出
- [ ] `create_manifests` / provenance / sbom / cosign 未改变语义（副作用为零，diff 可核）
- [ ] 探针标签已删除，`docker images | grep zz-local-stale-probe` 为空
- [ ] 未 push、未触发任何工作流、未动 `.local-instance/`

## Notes

- 来源：`10-06-team-hunt-deploy-chain` 失效模式 #6（真实风险）+ 遗留探针标签。
- 指派：Team Lead `lead`（要求由发现者本人修复）。
- **遗留未修**（不在本任务范围，需单独决策）：`workflow_dispatch` 重建**旧** tag 仍会覆盖全局
  `:latest-{arch}` 从而让 `:latest` 静默回退（失效模式 #5）；需要额外的"仅允许最新版本移动 latest"保护。
