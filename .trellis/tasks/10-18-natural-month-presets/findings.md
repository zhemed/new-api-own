# 自然月快捷范围 + 轴末标签对齐 — 交付

## 一、预设结构改成了什么（`dashboard/constants.ts`）

```ts
export type DashboardRangePresetKind = 'days' | 'month-to-date' | 'last-month'

export interface DashboardRangePreset {
  label: string
  kind: DashboardRangePresetKind
  /** Only meaningful when `kind === 'days'`. */
  days?: number
}

export const TIME_RANGE_PRESETS: readonly DashboardRangePreset[] = [
  { label: '1 Day', kind: 'days', days: 1 },
  { label: '7 Days', kind: 'days', days: 7 },
  { label: '14 Days', kind: 'days', days: 14 },
  { label: '30 Days', kind: 'days', days: 30 },
]

export const NATURAL_MONTH_PRESETS: readonly DashboardRangePreset[] = [
  { label: 'This Month', kind: 'month-to-date' },
  { label: 'Last Month', kind: 'last-month' },
]
```

## 二、四处消费点如何统一

唯一解析函数（`dashboard/lib/filters.ts`，经 `lib/index.ts` 导出）：

```ts
export function resolveRangePreset(preset, now = new Date()): ResolvedRange
// month-to-date → start = 本月 1 号 00:00, end = now,        days = now.getDate()
// last-month    → start = 上月 1 号 00:00, end = 本月 1 号 - 1ms, days = 上月末日
// days          → getCalendarDateRange(days, now)（语义未变：起始日 00:00 → now）
```

| 消费点 | 用法 |
|---|---|
| 默认偏好初始化 | 仍由 `defaultTimeRangeDays`（数字）→ `buildDefaultDashboardFilters` 内部的日历制 helper；等价于 `resolveRangePreset({kind:'days', days})`（有测试） |
| 查询参数 | 跟随 filters 的 `start_timestamp/end_timestamp/time_granularity`，无需改 |
| **快捷范围** | 对话框两组按钮都走 `resolveRangePreset`：`{...filters, start, end, time_granularity: granularityForRangeDays(range.days)}` |
| 用户图表 | 仍只用天数预设；粒度同样由 `granularityForRangeDays(selectedRange)` 推导 |

**取舍（你允许的退路，我选了它）**：自然月**只作为筛选对话框的快捷按钮**，默认范围仍是天数。
理由：把"默认范围"改成自然月需要把持久化的 `defaultTimeRangeDays: number` 扩成判别联合，
并同步改 localStorage 校验、默认值、迁移三处；对"永久默认"这种低频设置，收益不抵迁移风险，
且会污染 29→30 迁移的契约。默认值仍可精确表达 1/7/14/30 天。

## 三、向后兼容怎么保证

- `DashboardChartPreferences.defaultTimeRangeDays` **仍是 number**，存储键与校验（`isTimeRangePresetDays` 只看 `days`）不变；
- 旧预设对象形状新增 `kind`，`days` 必填于天数档 → 旧逻辑读取 `preset.days` 仍正确；
- 上一轮 **29 → 30 迁移**测试原样通过（`migrateLegacyTimeRangeDays` 未改）；
- 新增测试断言 `{kind:'days', days:30}` 解析为 **09-19 00:00 → now**，与旧行为逐字节一致。

## 四、末标签规则（`lib/charts.ts` `timeAxisLabelConfig`）

```ts
const indexes = [0, stride, 2*stride, …]
if (indexes.at(-1) !== data.length - 1) {
  indexes.push(data.length - 1)              // 追加轴末桶
  if (indexes.length > MAX_AXIS_LABELS) indexes.splice(indexes.length - 2, 1)  // 丢掉倒数第二个
}
```
- **首项恒为 0，末项恒为 n-1**（轴末端桶 = 今天）；
- 总数仍 ≤ `MAX_AXIS_LABELS`(10)；
- 30 桶 → `[0,3,6,9,12,15,18,21,24,29]`（10 个标签，最后一个 = 今天）；
- 24 桶 → stride 3 → `[0,3,…,21]` 已含末项 23？→ 实为 `[0,3,6,9,12,15,18,21,23]`（9 个），末项 23 ✓；
- 整除情形（如 20 桶 stride 2）不会重复追加。

## 五、测试（新增 12，另更新 2 条旧断言）

`lib/__tests__/range-presets.test.ts`：
- 本月：10-18 → 10-01 00:00~now（days 18）；11-01 00:05 → 11-01 00:00（days 1）；跨年 01-03 → 01-01（days 3）；
- 上月：10-18 → 09-01 00:00 ~ 09-30 23:59:59.999（days 30）；跨年 → 2025-12（days 31）；二月非闰年 → days 28；
- 天数档语义不变：30 天 → 09-19 00:00 → now；
- 粒度：本月/上月 → `day`；
- 末标签：24/30/31 桶均满足「首=0、末=n-1、总数 ≤10」，且整除时不重复。

更新旧断言：预设形状（`kind/days/label`）、抽稀序列（现含末桶：`… d24, d29`）。

## 六、七条门

typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **327 pass / 0 fail**（315 → +12）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。
新文案 `This Month` / `Last Month` 已按脚本补 7 语言（zh 本月/上月、ja 今月/先月、ru Этот/Прошлый месяц、fr Ce mois-ci/Le mois dernier、vi Tháng này/Tháng trước）。

## 七、边界
只改 `dashboard/**` + 7 locale；`lib/time.ts` 的 `getCalendarDateRange` **未改动**（仅新增自然月解析，位于 dashboard 内）。未 commit。
