# 帮用户升级实例到 v0.0.8 并启用 200MB 字节预算

## Goal

用户授权由我执行并在遇到问题时直接解决：把其容器从 0.0.7 重建为 0.0.8，带上 LOG_MEMORY_MAX_BYTES=200MB 与次级行数上限/保留天数，且不设 LOG_CLEANUP_INTERVAL（关闭定时清理）；随后按清单验证并处理任何问题。

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
