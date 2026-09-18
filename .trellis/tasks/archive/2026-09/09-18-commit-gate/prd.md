# 添加 Trellis 提交闸门（.githooks + 审计脚本 + CI 兜底）

## Goal

本仓库当前只有约定、没有机器强制：加 pre-commit/commit-msg 钩子当场拦截缺任务锚点的提交，加 scripts/check-trellis-gate.sh 审计历史（抓 --no-verify 绕过），并接入 CI 兜底；对齐 komari 的三层实现

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
