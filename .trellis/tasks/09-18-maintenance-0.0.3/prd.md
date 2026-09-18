# v0.0.3 维护轮次

## Goal

下一轮例行维护：复查依赖与安全基线、确认零同步上游、复跑质量门禁，然后把版本推进到 `0.0.3` 并发版
（GitHub Release + `v0.0.3` / `0.0.3` 双标签镜像）。

## 边界（不在本轮做）

- 不引入上游 QuantumNous/new-api 的任何改动（铁律 1：绝不同步上游）
- 不做与维护无关的功能开发；维护中发现的独立问题另开任务，不混进本轮

## 清单

### A. 事实复查（先看再改）

- [ ] `git status` 与 `git log --oneline origin/main..HEAD`：起始点干净、无未推送提交
- [ ] `git remote -v` 只有 `origin`：确认没有引入 upstream、没做过 fetch/merge/rebase
- [ ] 限流开关仍为有意关闭：`common/init.go` 四处 `*_ENABLE` 默认 `false` + `docker-compose.yml` 四个 env
- [ ] 部署安全基线（MAINTENANCE.md「部署安全基线（必读）」）逐条对照线上实例
- [ ] 数据与日志权限：`chmod 700 data data/logs && chmod 600 data/*.db data/logs/*`

### B. 依赖与工具链

- [ ] 后端：`GOWORK=off go vet ./...`、`go build ./...`、`cd relaykit && GOWORK=off go build ./...`、`make test`
- [ ] 前端（需 bun）：`cd web && bun install --frozen-lockfile && bun run typecheck && bun test`
- [ ] 记录 Go / Bun / Docker（须为 29.7.2 + Compose v5.4.0）实际版本，与 MAINTENANCE.md 记载比对
- [ ] 依赖安全扫描（如 `govulncheck ./...`、`bun audit`）结果可接受；不可接受的升级另开任务处理

### C. 文档与流程

- [ ] MAINTENANCE.md / README.md 与实际行为一致（限流默认、双标签镜像、发版流程、提交闸门）
- [ ] `./scripts/check-trellis-gate.sh` 通过（闸门已装 + 提交可追溯）
- [ ] `.trellis/spec/backend/` 五个规范仍与代码一致，必要时更新

### D. 发版 0.0.3

- [ ] `VERSION` 改为 `0.0.3` 并提交（提交消息带 `[task:maintenance-0.0.3]`）
- [ ] `git tag -a v0.0.3 -m "v0.0.3"`，推送 `main` 与 tag
- [ ] 确认 CI 全绿：`Release`（Linux/macOS/Windows 二进制）与 `Publish Docker image (Multi-arch)`
- [ ] `docker run --rm ghcr.io/zhemed/new-api-own:0.0.3 --version` 输出 `v0.0.3`；`:v0.0.3` 同样可拉
- [ ] 本任务 `task.py finish` + `archive`，并写 journal

## Acceptance Criteria

- [ ] 以上清单全部勾选；未勾选项必须写明原因与后续任务编号
- [ ] `main` 与 `origin/main` 同步，tag `v0.0.3` 已推送且在 Releases 页可见
- [ ] GHCR 同时存在 `v0.0.3` 与 `0.0.3`，镜像内版本号为 `v0.0.3`

## Notes

- 维护者开始时：`python3 .trellis/scripts/task.py start 09-18-maintenance-0.0.3`（状态 `planning` → `in_progress`）。
- 本轮内如需改代码，闸门允许（本任务处于 in_progress），提交消息统一带 `[task:maintenance-0.0.3]`。
- 维护中发现的新问题不要顺手做：`task.py create` 另开任务，保持本轮范围清晰。
