# 盘点工作区未完成任务

## Goal

用户提问：「我们工作区现在还有未完成的任务吗，查一下」。

对工作区做一次**只读**盘点，回答：当前是否还有未完成（未归档 / 未收尾 / 半途停下的）任务，
并列出每一项的标识、状态、所处阶段、残留工作与证据出处。

## Requirements

- R1 盘点范围（全部只读）：
  - `.trellis/tasks/*/` 下**活动任务**（未归档）及其 `task.json` 的 `status`、当前 phase/step、
    `prd.md` 中未勾选的验收项、`journal.md`/`implement.md` 中未完成的步骤。
  - `.trellis/tasks/archive/`（或归档目录）中的任务，作为「已完成」的对照基线，不重复计入未完成。
  - `.trellis/workspace/` 下 journal 里明确写下的「未完成 / 下一步 / TODO」条目。
  - 工作区 git 状态：当前分支、未提交改动、是否有未跟进的分支/提交（用于判断有无半成品改动）。
  - 本轮团队任务板（`team_task_*`）中 `pending` / `in_progress` 的共享任务。
- R2 判定口径必须写清楚：区分
  「已归档 = 完成」、「status=in_progress/planning = 未完成」、「status=completed 但目录未归档 = 待收尾」。
- R3 每一项未完成项必须给出**出处**（文件路径，必要时给行号），不得凭印象下结论。
- R4 只读到底：不修改代码 / 配置 / 服务 / 不提交 / 不推送；仅允许 Trellis 流程留痕
  （建任务、写本 PRD、收尾 journal）。
- R5 若某任务的 `task.json` 不可读或状态自相矛盾（`[workflow-state:task_error]`），如实标注为「状态待修复」，
  不猜测其内容。

## Acceptance Criteria

- [x] AC1 给出「是否还有未完成任务」的明确结论（有 / 无），而不是模棱两可的措辞。
- [x] AC2 未完成项逐条列出：任务目录名、status、所处阶段、残留工作、证据文件路径。
- [x] AC3 明确说明已完成（已归档）任务数量与归档目录位置，作为对照基线。
- [x] AC4 结论中的每一条都能在仓库内被复核（给出的路径真实存在）。
- [x] AC5 全程无越界访问（不读工作区外文件、无网络动作），并在报告中说明这一点。
- [x] AC6 本轮新建的盘点任务本身在收尾时按流程 finish + archive，并写 journal。

## 验收结果（2026-10-23，只读盘点）

- 产出：`.trellis/tasks/10-23-workspace-task-audit/audit-report.md`（侦察员 scout-audit 落盘，171 行六小节）。
- AC1：结论「有 1 个未完成任务 = 本盘点任务自身；无其它未归档任务」。Lead 独立复核通过。
- AC2：明细表见 audit-report.md 表格 A（活动任务）+ 表格 B（journal 挂起事项 5 项 / 6 条记录）。
- AC3：`.trellis/tasks/archive/` 共 **161** 个任务，全部 `status=completed`（2026-09: 8，2026-10: 153）；
  Lead 复跑 `find ... | wc -l` = 161、逐目录解析 status 计数 `{'completed': 161}`，一致。
- AC4：Lead 抽查 `journal-2.md:1059-1061 / 1073 / 1093-1095 / 1143-1147`、git 状态与 archive 计数，均与报告一致。
- AC5：侦察员自述 + Lead 复核：未读工作区外文件、无网络动作、无 docker/systemctl/提交/推送；
  唯一写入为本次任务目录内的 `audit-report.md`。
- AC6：本文件勾选后执行 `task.py finish` + `task.py archive --skip-branch-validation --no-commit`，
  并已用 `add_session.py --no-commit` 写入 Session 94 journal（未提交，用户本轮只要求查，未授权 commit）。

## 附加发现（不在原 AC 内，如实记录）

- 归档任务中 **113 个** `prd.md` 仍留有未勾选 `- [ ]` 验收项（记账漂移，非未完成工作）。
  抽查最近两个：`10-23-improve-self-update` 已交付并发 v0.0.43（提交 813d5c4）后归档，AC 未勾属记账问题；
  `10-23-deepseek-v41-flash-pricing` 的 AC8 是用户当轮明确裁定跳过的回滚演练，属真实遗留。
- 该现象说明「AC 未勾 ≠ 工作未完成」，但建议后续归档前顺手勾选，避免下次盘点误判。

## Notes

- 本任务是**只读调查类**任务，PRD-only（轻量）即可，不需要 `design.md` / `implement.md`。
- 「未完成」的用户语义偏向「有活儿还没干完」，因此除 Trellis 任务状态外，还要看 git 未提交改动与
  workspace journal 里的挂起项，避免只报 Trellis 状态而漏掉真实半成品。
- 交付形式遵循全局规则第 9 条：完整报告直接写进对话正文，不压缩成摘要。
