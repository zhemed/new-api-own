# 前端一致性核对：日志设置区块 vs `LOG_MEMORY_MAX_BYTES` 默认

## Goal

后端日志上限机制改为**按字节预算**（新增 `LOG_MEMORY_MAX_BYTES`，且**配置它时清理定时器默认关闭**）。
核对「系统设置 → 系统维护 → 日志设置」区块（`log-settings-section.tsx`）的文案与行为是否仍与后端默认一致；
仅当它**暗示"会自动每 5 分钟清理"**时才改，否则不动。

## Requirements

### R1 现状核对（只读）
- 说清该区块**展示**什么、**触发**什么：是纯手动入口（`POST /api/system-task/log-cleanup`，按时间点清），
  还是也展示/暗示"自动清理"状态。
- 结论必须落到具体文件:行 + 文案原文。

### R2 有条件修改
- **仅当**区块在**任何地方**暗示"会自动每 5 分钟清理"时，才把文案/行为改成与后端实际默认一致
  （有字节预算 → 定时器关闭）。
- 若为纯手动入口：**不改**，只回报结论。

### R3 内存占用展示（只评估，不实现）
- 评估在界面加"内存日志占用 / 上限"只读展示的可行性：需哪些后端字段、是否需要新外呼。
- **本轮不实现**；不新增任何浏览器外呼。

### R4 质量门
- `bun run typecheck | lint | test | build | knip | i18n:check` 全绿；`knip` 基线红照旧说明。
- 如需新增/修改文案，走临时 `scripts/add-missing-keys.mjs` + `node scripts/sync-i18n.mjs`（禁止手改 locale JSON）。

## Constraints

- 只改 `web/**`；不 commit / push；**不碰用户实例（3000 端口那台是用户的）**。
- 不新增依赖、不引入浏览器外呼。
- 不确定的标"待确认"，不臆造后端行为——以 `10-06-log-byte-budget-write-trigger` 的 PRD 与后端代码为准。

## Acceptance Criteria

- [ ] AC1：给出区块现状结论（展示什么 / 触发什么 / 对应文件:行与文案原文）。
- [ ] AC2：明确结论「改了 / 没改」；若改了，说明改的是哪几处、为什么、如何与后端默认对齐。
- [ ] AC3：给出"内存占用只读展示"的可行性评估（所需字段 + 是否需外呼），且**未实现**。
- [ ] AC4：六条命令输出齐备（knip 基线红有对照证据）。
- [ ] AC5：改动仅限 `web/**`（若无需改动，则工作区无本次新增改动）；未 commit/push、未触碰 3000 实例。

## Notes

- 参考：`.trellis/tasks/10-06-log-byte-budget-write-trigger/prd.md`（后端侧）。
- 结论与命令输出留 `findings.md`。
