# 对外模式默认余额由200改为10万

## Goal

调整对外模式对外用户默认余额，将200调整为100000

## Requirements

- R1 需求确认：用户要求把「对外模式」对外用户的默认余额由 200 调整为 100000；动手前需先确认 200 的实际来源，即仓库代码默认值还是实例数据库中的 option 当前值。
- R2 只读确认「对外模式」定义：对外模式即设置向导三选一的 external「对外运营」，选项定义在 `web/src/features/setup/components/usage-mode-step.tsx:48-55`，提交映射为 `SelfUseModeEnabled=false` 且 `DemoSiteEnabled=false`，见 `web/src/features/setup/api.ts:44-52`；后端只落这两个布尔值，见 `controller/setup.go:125-134`。
- R3 默认余额取证：全仓未找到独立的「对外模式默认余额」；新用户默认额度为 `common.QuotaForNewUser`，代码默认 0，见 `common/constants.go:128`；前端表单默认 0，见 `web/src/features/system-settings/billing/index.tsx:28`；写入路径为 `model/option.go:538-539` 与 `model/user.go:597/661`；`grep 对外模式` 仅命中 i18n `web/src/i18n/locales/zh.json:1672-1673`。
- R4 结论与再开工前提：200 更可能是实例数据库中 option 的当前值，例如 `QuotaForNewUser` 已配置为 200，而非仓库代码默认值；再次开工前需用户指明具体页面或配置项，再决定改代码默认值还是改实例 option，不得直接改代码。
- R5 终止口径：用户中途指令「停止一切任务，并归档」，任务在 planning 阶段终止，代码与配置零改动；已执行 `task.py finish` 与 archive，任务移入 `.trellis/tasks/archive/2026-10/10-31-external-default-quota`。
- R6 约束：只读调研，不修改代码、配置与数据库；不臆造代码中不存在的配置项。

## Acceptance Criteria

- [x] 给出「对外模式」的代码定义与提交映射，含权威文件与行号。
- [x] 记录默认余额的代码默认值 0 与前端表单默认值 0，并给出写入路径。
- [x] 给出 200 更可能来自实例 option 当前值的判断与依据。
- [x] 记录任务终止原因：用户中途叫停并归档，代码与配置零改动。
- [x] 记录再次开工前提：需用户指明页面或配置项后再决定改代码默认值或实例 option。
- [x] 归档完成，任务目录位于 `.trellis/tasks/archive/2026-10/10-31-external-default-quota`，task.json 状态为 completed。

## Result

- 结论：仓库不存在「对外模式默认余额」独立配置；`common.QuotaForNewUser` 代码默认 0，前端表单默认 0，200 疑为实例 option 当前值。
- 终止：用户指令「停止一切任务，并归档」，任务在规划阶段终止，未产生代码或配置改动。
- 证据：本次回填依据 `.trellis/workspace/new-api-own/journal-3.md` Session 122 与提交 `7e9717c`。

## Notes

- 任务类型：只读调研，轻量任务，PRD-only，无 design.md 与 implement.md。
- 若继续：先请用户指明「对外模式默认 200 余额」所在页面或配置项，再另开任务决定改代码默认值还是改实例 option。
- task.json 的 commit 字段由归档流程留空，归档提交以 `7e9717c` 为准。
