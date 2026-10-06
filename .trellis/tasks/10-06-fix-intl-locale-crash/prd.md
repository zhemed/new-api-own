# 修复 v0.0.8 回归：日志页因语言标签崩溃

## Goal

i18n.language 为 zhCN 时 Number.toLocaleString 抛 RangeError，整页错误边界；改用 toIntlLocale 归一化并补回归测试；发 v0.0.9 并更新用户实例。

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
