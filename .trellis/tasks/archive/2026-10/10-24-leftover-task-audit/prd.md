# 检查工作区残留任务

## Goal

回答用户当前问题：`/root/new-api-own` 工作区现在是否还有残留的 Trellis 任务；若有，给出完整清单与处理建议。

## Requirements

- 只读审计：不得修改、删除、归档任何既有任务目录或文件（仅允许 Trellis 流程自身为本任务写记录）。
- 覆盖范围：
  1. `.trellis/tasks/` 下所有非 `archive/` 的任务目录，逐个读取 `task.json` 的 status（planning / in_progress / completed 等）、createdAt、completedAt。
  2. 判断是否存在 "僵尸任务"：目录存在但未归档、状态长期停在 planning/in_progress、或与当前会话无绑定。
  3. 检查本会话绑定：`.trellis/.runtime/sessions/`（或等价运行态）是否残留旧会话的 current_task 绑定。
  4. 检查任务关联的 git 残留：`git worktree list`、任务 `task.json` 中 `branch` / `worktree_path` 字段非空且对应分支/目录仍存在的情况。
  5. 附带说明工作区未提交改动中是否含任务产物残留（仅陈述，不修改）。
- 输出：先给 "有 / 没有" 的明确结论，再给逐条证据表（slug、状态、创建日期、判定依据、建议动作）。

## Acceptance Criteria

- [x] 明确回答用户问题：是否有残留任务（结论 + 证据）。
- [x] 残留任务清单完整：每个非归档任务目录都被列出或明确说明为何不算残留。
- [x] 未对任何既有任务/代码/配置做写入（`git status` 中除本任务目录与流程记录外无新增改动）。
- [x] 如需后续清理，仅提出问题与建议，不擅自执行。

## 验收结果（2026-10-24）

- 审计前非归档任务：**0 个**；审计期间仅存在本任务自身（结束后归档）。
- 归档仓：188 个 `task.json` 全部 `status=completed`、`completedAt` 非空；无 `worktree_path`；`branch` 字段仅 `main`(177) 与空(11)。
- git：单 worktree（`/root/new-api-own`，main）、无 stash、无 feature 分支残留。
- 非任务残留（gitignored，未处理）：`.trellis/.runtime/update-check-*.marker` 9 个（2026-09-18 ~ 2026-10-23）；`.verify-scratch/` 23 个文件 2.6MB（2026-10-09 ~ 10-19 验证截图）；`__pycache__` 等生成物。
- 结论：工作区无残留 Trellis 任务；无代码/配置改动。

## Notes

- 本任务为轻量任务，PRD-only。
- 审计结束后本任务自身也应 finish + archive，避免审计动作本身制造残留。
