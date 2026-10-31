# 残留清理与提交推送

## Goal

按用户当轮授权，处理 Session 123 盘点出的三类残留：推送本地未推送提交到 origin/main、回填两个 10-31 归档任务仍是 TBD 模板的 PRD、清理 gitignored 残留（`.verify-scratch` 与过期 `update-check` marker）。

## Requirements

- R1 推送：把 `origin/main..main` 的 10 个提交推送到 `origin`，保持提交内容不变，不 rebase、不 force。
- R2 回填 PRD：为两个归档任务补写 Requirements 与 Acceptance Criteria，依据只取既有证据，不得臆造：
  - `.trellis/tasks/archive/2026-10/10-31-external-default-quota/prd.md`：依据 journal Session 122 与 `git show 7e9717c` 的调研结论；
  - `.trellis/tasks/archive/2026-10/10-31-archive-commit-cleanup/prd.md`：依据 `git show cd3c633` 与 `git show 9668862` 的归档与团队任务清理事实。
- R3 清理残留：
  - 删除 `.verify-scratch` 下 23 个过期截图文件（约 2.6MB，2026-10-09 至 10-19，gitignored）；
  - 删除 `.trellis/.runtime/` 下 14 个 `update-check-*.marker`（gitignored 运行时缓存）；
  - 保留 `.trellis/.runtime/sessions/` 中当前会话绑定文件。
- R4 复核：清理后 `git status --porcelain` 仅剩本任务自身改动；`git log origin/main..main` 为空；两个回填 PRD 不含 TBD 模板行。
- R5 收尾：本任务 finish、archive 并写 journal。

## Constraints

- 只删除 gitignored 的过期产物，不删除任何被 git 跟踪的文件；删除前逐个确认路径。
- 回填只改 `.trellis/tasks/archive/` 下两份 `prd.md`，不改代码、不改 task.json 状态。
- 推送目标固定 `origin/main`，不新增远端、不改 remote 配置、不 force push。
- 团队分工：r4-operator 执行推送与清理，r2-analyst 执行 PRD 回填，写作用域不重叠；Lead 负责 PRD、复核与归档。

## Acceptance Criteria

- [x] AC1: `git push origin main` 成功，`git log origin/main..main` 输出为空。初始 10 个提交由 r4-operator 推送，`git ls-remote` 双侧一致。
- [x] AC2: 两个归档 PRD 的 Requirements 与 Acceptance Criteria 均回填为具体内容，无 TBD；由 r2-analyst 执行，diff 41 增 10 删。
- [x] AC3: `.verify-scratch` 与 14 个过期 marker 已删除；`.trellis/.runtime/sessions/` 当前会话文件保留，被跟踪文件零改动。
- [x] AC4: 提交前 `git status --porcelain` 仅本任务目录与两份回填 PRD；`git diff --stat` 无代码改动。
- [x] AC5: 本任务 finish、archive、journal 完成，最终答复列明结果。
