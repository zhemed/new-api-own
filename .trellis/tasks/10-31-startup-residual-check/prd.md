# 会话启动与残留任务排查

## Goal

本会话建立 Trellis 新任务，并只读排查 `/root/new-api-own` 的 `.trellis` 工作区是否存在残留任务（未收口、未归档、状态异常）或脏工作区，给出可执行的收口建议。

## Requirements

- R1: 新建并激活本任务 `10-31-startup-residual-check`，状态从 planning 进入 in_progress。
- R2: 只读盘点残留任务，覆盖：
  - R2.1 `.trellis/tasks/*/task.json` 中 status 非 completed 的任务；
  - R2.2 `.trellis/tasks/archive/` 归档目录结构与最新归档；
  - R2.3 `.trellis/.runtime/sessions/` 是否有其它会话绑定的 current_task；
  - R2.4 `.trellis/workspace/` 最近 journal 中记录的未完成事项；
  - R2.5 `git status` 未提交改动、分支、最近提交的任务锚点；
  - R2.6 任务目录异常：空目录、缺 `prd.md`、缺 `implement.jsonl` 等。
- R3: 输出残留判定与建议下一步；本次排查不修改任何代码或配置。
- R4: 按 Trellis 收尾五步留痕：收口、finish、archive、journal、最终答复。

## Constraints

- 纯只读排查，除本任务自身的 Trellis 制品（prd.md 等）与 journal 外不写任何文件。
- 团队事实前置：已建立共享任务 task-1 并派生 `r2-analyst` 执行盘点，Lead 负责编排、审查、落痕与答复。
- 不臆造状态，一切以 `.trellis/tasks/` 与 git 实际输出为准。

## Acceptance Criteria

- [x] AC1: 本任务 `task.json` 状态为 in_progress，且 `prd.md` 内容完整（非 TBD 模板）。
- [x] AC2: 给出残留任务清单，逐条包含路径、状态、判定依据：非归档任务 0 个，归档 193/193 完整，结论无任务残留。
- [x] AC3: 给出 git 工作区状态核实结果：仅本任务目录未跟踪，main 领先 origin/main 7 个提交，无 stash/多 worktree。
- [x] AC4: 明确结论：不存在需要收口的残留 Trellis 任务；非任务残留（marker、.verify-scratch）留待用户授权后另开任务。
- [x] AC5: 任务完成后 finish、archive 并写 journal。

## Notes

- 轻量任务，PRD-only 即可，无需 design.md / implement.md。
- 本会话执行的命令均为只读查看，不涉及网络与工作区外路径。
