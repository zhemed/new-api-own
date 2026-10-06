# 复查后端与文档面（团队）

## Goal

对 commit `208483c` / `97fd7d6` / `5ad97b7`（相对基线 `0941a4f`）的改动做**独立复核**，
确认 Go 侧未被污染、全仓库无指向已删前端文件的残留引用、`MAINTENANCE.md` / `docs/FILE_INVENTORY.md`
描述与代码现状一致、`.gitignore` 新增规则合理。默认态度：怀疑，所有结论必须实测取数。

## Requirements

1. **Go 侧未受影响**
   - `git diff 0941a4f..HEAD --name-only` 中不出现 `.go` 文件（`.trellis/**` 与文档除外）。
   - 实测四条命令：`GOWORK=off go vet ./...`、`GOWORK=off go build ./...`、
     `cd relaykit && GOWORK=off go build ./...`、`make test`。
2. **残留引用扫描**
   - 以 `git show --stat 5ad97b7` 与 `git log --diff-filter=D --name-only -3` 得出「已删文件清单」。
   - 扫描全仓库（`*.md`、脚本、注释、`.trellis/spec/**`、`.github/**`、`README*`、`MAINTENANCE.md`）
     是否点名这些文件或 `Dockerfile.dev`；有则修（限写范围）。
3. **文档准确性**
   - `MAINTENANCE.md`「前端工具链（2026-10-06 起本机可用）」：每条命令可跑通、路径存在。
   - `MAINTENANCE.md`「已知不一致（刻意未改）」：逐条与代码现状比对；重点抽查两条——
     3 个 api-key 测试失败的成因、ClickHouse 行数上限的行为。
   - `docs/FILE_INVENTORY.md` 文件计数是否因本轮删除失真；失真则按实测更新。
4. **`.gitignore` 的 `.verify-*/`**：是否合理、是否误伤正常文件（例如真实以 `.verify-` 开头的目录）。

## Constraints

- 禁止网络动作、禁止访问生产、禁止 `git commit/push/tag`。
- **禁止 `git add -A`**、**禁止在仓库内建 git worktree**。
- 只改 Go 代码与 `.md` / `.trellis/spec/` / `.gitignore`；**不动 `web/**`**；
  不动 `.local-instance/`；不重启任何服务。

## Acceptance Criteria

- [ ] 四项各自给出「结论 + 证据（`文件:行`）」。
- [ ] 四条 Go 命令的实测结果（通过/失败）与关键输出已记录。
- [ ] 已修清单（若修改）与「需 Lead 处置清单」分列。
- [ ] 明确列出「无法验证的项」及原因。
- [ ] 汇报给 Lead：中文，结论先行，≤25 行。

## Notes

- 共享任务：`task-6`（本任务对应）。
- 独立复核，不得采信前一位 agent 的自述结论。
