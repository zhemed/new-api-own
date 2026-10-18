# 默认范围支持自然月（本月/上月）

## 背景
用户把「默认范围」设成 30 天，图表永远从 09-19 起（30 天滚动窗口），但他要"默认从本月 1 号开始"。
根因：偏好下拉只列 `TIME_RANGE_PRESETS`（1/7/14/30 天）；自然月只做在筛选对话框；
`buildDefaultDashboardFilters()` 只用 `getCalendarDateRange(days)` → 永远滚动 N 天。

## 要做（加法式，**不破坏**老数据与 29→30 迁移）
1. `types.ts`：保留 `defaultTimeRangeDays: number`，**新增可选** `defaultRangeKind?: 'days' | 'month-to-date' | 'last-month'`（缺省视为 `'days'`）。
2. `lib/filters.ts`：
   - `getSavedChartPreferences()` 校验 kind（非法 → `'days'`），天数继续 `isTimeRangePresetDays` + 29→30 迁移；
   - `saveChartPreferences()` 同时写 `defaultRangeKind`（天数档 → `'days'` + days；自然月 → 对应 kind）；
   - `buildDefaultDashboardFilters()` 统一走 **`resolveRangePreset({kind, days})`**，粒度用**实际跨度天数** `granularityForRangeDays(range.days)`。
3. `models-chart-preferences.tsx`：默认范围下拉分**两组** —— 天数（1/7/14/30）+ 本月 / 上月（复用 `NATURAL_MONTH_PRESETS` 与既有 i18n）；draft 同时持有 kind。
4. `detectQuickRangeDays` 等既有行为不回归（自然月不匹配天数档属预期）。

## 测试（必须）
- 旧数据 `{defaultTimeRangeDays: 30}`（无 kind）→ 仍 **09-19 00:00 → now**；
- `{defaultRangeKind:'month-to-date'}` → 本月 1 号 00:00 → now（跨月/跨年边界各一例）；
- `{defaultRangeKind:'last-month'}` → 上月整月；
- 保存→读取**往返**一致；非法 kind → 回落 days；
- 29→30 旧迁移测试仍通过；
- 偏好对话框选项包含"本月/上月"（组件级断言）。

## 门
七条门全绿（基线 **327**，只能增）；i18n 无需新键（复用既有 This Month / Last Month）。

## 写范围
`web/src/features/dashboard/**`。不 commit/push。
