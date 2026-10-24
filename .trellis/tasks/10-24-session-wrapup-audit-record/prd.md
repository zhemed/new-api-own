# 收尾：补齐审计任务归档记录与 journal

## Goal

修复审计任务 `10-24-leftover-task-audit` 归档时 `task.py archive` 自动提交失败造成的未跟踪记录，补写 journal，并把工作区恢复到"无残留任务、无未跟踪记录"的干净状态。

## Background

- `10-24-leftover-task-audit` 已运行 `task.py finish` + `task.py archive --skip-branch-validation`，目录已移入 `.trellis/tasks/archive/2026-10/10-24-leftover-task-audit`。
- 归档脚本自动提交失败：`error: pathspec '.trellis/tasks/10-24-leftover-task-audit' did not match any file(s) known to git`（该任务目录此前未被 git 跟踪）。
- 归档后会话无活动任务，Trellis 闸门拦截 `add_session.py`，故建立本收尾任务来补齐流程记录。

## Requirements

- 提交已归档的 `10-24-leftover-task-audit` 记录（`chore(task): archive ...` 类提交）。
- 用 `add_session.py` 记录本轮审计 session（标题、summary、changes、tests、next steps 完整）。
- 记录完成后将本收尾任务自身归档，保证会话结束后 `.trellis/tasks/` 下无未完成/未跟踪任务。
- 全程不得修改代码、配置、既有任务内容；仅动 `.trellis/` 流程记录与 git 提交。
- 最终 `git status --porcelain` 为空（忽略 gitignored 文件），`git worktree`/分支/stash 无新增残留。

## Acceptance Criteria

- [ ] `10-24-leftover-task-audit` 的归档目录已被 git 跟踪并提交。
- [ ] journal 中新增本轮审计 session 记录（含 188 归档全 completed、非任务残留说明）。
- [ ] 本收尾任务已归档，且其归档目录已提交/无未跟踪残留。
- [ ] `git status --porcelain` 为空（仅可能有 gitignored 条目），无代码/配置改动。
