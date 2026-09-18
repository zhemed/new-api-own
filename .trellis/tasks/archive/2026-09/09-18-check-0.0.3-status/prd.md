# 核对 v0.0.3 发布状态

## 结论

**v0.0.3 尚未发布**，所有发布出口都停在 v0.0.2。

## 证据（2026-09-18 核对）

| 出口 | 现状 |
|---|---|
| 仓库 `VERSION` | `0.0.2`（未递增） |
| 远端 tag | 只有 `v0.0.2`（附注 tag `08f40bf` → 提交 `6829e8e`） |
| GitHub Releases | 只有 `v0.0.2`（标记 Latest，发布于 2026-09-18T01:09:43Z） |
| GHCR 镜像标签 | 只有 `v0.0.2` / `0.0.2` / `latest`（+ 各架构标签）；无 `0.0.3` |
| 维护任务 | `.trellis/tasks/09-18-maintenance-0.0.3` 处于 `planning`（尚未 start） |

## 附带发现

- 线上实例 `线上实例/api/status` 的 `version` 仍为空字符串 → 跑的是 2026-09-12 12:30
  （`e1fcb53` 填版本号）之前构建的镜像，**连 v0.0.2 都还没部署**。
- 本机没有 bun（`~/.bun/bin/bun` 不存在），维护清单 B 段的「前端 typecheck / bun test」在本机跑不了，
  需在有 bun 的机器上补跑，或在清单里标注跳过原因。

## Acceptance Criteria

- [x] 逐出口核对（VERSION / 远端 tag / Releases / GHCR / 维护任务状态）
- [x] 记录线上实例版本现状与本机缺少 bun 的限制
- [x] 未做任何写操作
