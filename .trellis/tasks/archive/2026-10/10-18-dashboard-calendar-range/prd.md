# 仪表盘时间范围改日历制 + x 轴按范围铺满 + 29→30 天

## 背景（Lead 已诊断，用户原话："应该是 00:00 到实际时间"）

1. **范围是滚动窗口**：`getRollingDateRange()`（`src/lib/time.ts:50`）实现为 `start = now - days*24h`，
   被 `dashboard/lib/filters.ts:146`、`models-filter-dialog.tsx:137,159`、`user-charts.tsx:90` 消费。
   → 选 29 天时起点是"29×24 小时前"，不是某个 00:00。
2. **x 轴被数据拽**：`dashboard/lib/charts.ts:278 fillTimePoints()` 用 `max(created_at)` 当终点往前补桶，
   标签由 `formatChartTime()` 生成 → 轴上首条=最早数据、末条=最后数据，刻度不规则。

## Requirements

### R1 范围改日历制（4 个消费点统一）
- 新增明确命名的 helper（如 `getCalendarDateRange(days, from)`）：
  **起点 = 范围内第一天 00:00（本地时区）**，**终点 = 当前时刻**。
  - 1 天 → 今天 00:00 → now；
  - 30 天 → 29 天前的 00:00 → now（共 30 个日历日）。
- **必须把 4 个消费点统一改过去**（filters.ts、models-filter-dialog.tsx ×2、user-charts.tsx），
  不允许只改一处。
- 保留"含当前小时"的既有意图（现有 `computeTimeRange` 的 +1h buffer 注释，别把当前桶切掉）。
- 跨月/跨年必须正确（用日期算术，不要 `now - n*24h`）。

### R2 x 轴按范围铺满
- `fillTimePoints` 接收 `range`（start/end），由 `processChartData` 及调用链把 range 传进去；
  **不再**用 `max(created_at)` 当终点。
- 从范围起点 → 当前时刻，按当前粒度（hour/day/week）**逐桶生成标签**；真实数据按 bucket 落到对应标签，
  **缺失桶补 0/空**；刻度规则化。
- 桶数过多时按既有 `TIME_RANGE_BY_GRANULARITY` 的 range↔granularity 配对处理，不得生成上万点。

### R3 29 天 → 30 天
- `dashboard/constants.ts:46 TIME_RANGE_PRESETS`：`{ label: '29 Days', days: 29 }` → `{ label: '30 Days', days: 30 }`。
- 7 个 locale 的 `"29 Days"` → `"30 Days"`：en `30 Days`、zh `30 天`、zh-TW `30 日`、ja `30日`、
  ru `30 дней`、fr `30 jours`、vi `30 ngày`。
- **必须走 i18n 脚本流程**（临时 `scripts/add-missing-keys.mjs` + `node scripts/sync-i18n.mjs`），
  禁止手改乱序；`i18n:check` 无残留键。

### R4 测试（必须有）
- 范围 helper：1 天 = 今天 00:00 → now；30 天 = 29 天前 00:00 → 30 个日历日；**跨月/跨年**；本地时区。
- 轴填充：给定 start/end + 粒度，桶序列**首=范围起点、末=当前桶**、数量正确、缺失补空、真实数据落位正确。
- 断言"轴终点随 now 变化"：**注入 `now` 参数或 mock 时间**，禁止依赖真实时钟。

## Constraints
写范围：`web/src/features/dashboard/**`、`web/src/lib/time.ts`、`web/src/i18n/locales/*.json`。
不改后端；不 commit/push；不碰用户实例。

## Acceptance Criteria
- [ ] AC1：4 个消费点全部改用日历制 helper（grep 可证 `getRollingDateRange` 无残留调用或已重命名）。
- [ ] AC2：x 轴首=范围起点、末=当前桶，缺失桶补空，数据落位正确（有测试）。
- [ ] AC3：29→30 天，7 语言同步，`i18n:check` 干净。
- [ ] AC4：六条命令全绿；`bun test` ≥247 pass / 0 fail（只能增）；`format:check` 干净；`knip` exit 0。
- [ ] AC5：回报含诊断确认、改动清单、1 天与 30 天的前后行为对照、测试与命令输出。

## Notes
- `getRollingDateRange` 若仍被非 dashboard 代码使用，保留其定义但确保 dashboard 不再用；否则一并改造。
- 结论留 `findings.md`。
