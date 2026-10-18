# 调查：默认范围为何只能是天数

## 用户观察（2026-10-18）

1. 「模型分析默认设置」里**默认范围 = 30 天**；
2. 图表仍然 `09-19 … 10-18`（= 30 天滚动窗口）；
3. 用户推断：「因为选择的是 30 天，从 1 号到现在根本没有 30 天」→ 想知道怎么办，且指出"0.0.25 还是这样子"。

## 需查证

1. **偏好设置**里「默认范围」的选项来源（是否只有 `TIME_RANGE_PRESETS` 天数档）；
2. 新增的「本月 / 上月」出现在**哪个**对话框（筛选 vs 偏好设置）——用户是否能在偏好里选到；
3. 默认偏好 → `buildDefaultDashboardFilters` → 图表的传导链，确认"改了默认范围才能改图"；
4. 结论与方案：让**默认范围支持自然月**（持久化 kind + 向后兼容 + 迁移），或明确告知只能在筛选里选。

## Acceptance Criteria

- [ ] 用代码行号说明选项来源与传导链
- [ ] 明确"为什么选 30 天就一定是 09-19 起"
- [ ] 给出方案（含持久化兼容做法）与风险

## 调查结论（2026-10-18，只读，已用行号确认）

### 传导链（用户的推断成立 ✓）

1. **偏好设置**的「默认范围」选项来源 = `TIME_RANGE_PRESETS`（只有 `1/7/14/30 天`）
   —— `models-chart-preferences.tsx:92` 与 `:109`；
2. **「本月 / 上月」只出现在筛选对话框** —— `models-filter-dialog.tsx:229`（`NATURAL_MONTH_PRESETS`）；
3. 默认偏好 → `buildDefaultDashboardFilters()`（`lib/filters.ts`）：
   ```ts
   const { start, end } = getCalendarDateRange(preferences.defaultTimeRangeDays)
   ```
   → **只要有默认值（30 天），图表就一定按"最近 30 天滚动窗口"渲染** → 今天 10-18 → **09-19 起**。
4. 持久化类型 `DashboardChartPreferences.defaultTimeRangeDays: number`（`types.ts:201`）——**结构上没有"自然月"的表达能力** ✗。

⇒ 结论：v0.0.25 里"本月/上月"只是**筛选里的临时范围**；**默认范围仍只能选天数**，
而用户想要的是"默认就从 1 号开始" → 上一轮我为了避免改持久化结构而砍掉的正是用户真正需要的东西 ✗。

### 方案（下一步）

- `DashboardChartPreferences` **加法式**扩展：保留 `defaultTimeRangeDays: number`，新增可选
  `defaultRangeKind?: 'days' | 'month-to-date' | 'last-month'`（缺省视为 `'days'`，旧数据零迁移）；
- 偏好设置的「默认范围」下拉改为**两组选项**（天数 + 本月/上月），保存时写 kind（天数档写 kind='days' + days）；
- `buildDefaultDashboardFilters()` 统一走 `resolveRangePreset`（自然月 → 1 号 00:00 → 现在 / 上月整月），粒度按跨度推导；
- 测试：旧数据 `{days:30}` 行为不变；`{kind:'month-to-date'}` → 本月 1 号 → now；29→30 迁移仍通过；保存/读取往返一致。

## 实现结果（2026-10-18，v0.0.26）

用户选"修：默认范围支持本月/上月"：

- `types.ts`：`DashboardChartPreferences` 保留 `defaultTimeRangeDays: number`，**新增可选 `defaultRangeKind`**（`days | month-to-date | last-month`，缺省 `days`）；
- `filters.ts`：`isRangePresetKind` 校验（非法/缺失 → days）；天数仍走 `isTimeRangePresetDays` + **29→30 迁移**；
  `saveChartPreferences` 归一化写显式 kind；`buildDefaultDashboardFilters(prefs, now?)` 统一走 `resolveRangePreset`，粒度按**实际跨度**推导；
- 偏好设置下拉改为**单控件两分组**（`days:<n>` / `<kind>`）：天数 1/7/14/30 + 本月/上月，**无需新增 i18n**；
- 新增 **10** 个测试（旧数据不变、本月含跨年、上月整月、往返一致、非法回落、29→30 迁移仍生效、下拉含本月/上月、触发器显示 Last Month）；`bun test` 327 → **337**；七条门全绿。

### 用户最终怎么用

偏好设置 → 默认范围 → 选「**本月**」→ 保存 → 之后每次打开图表都从 **本月 1 号 00:00 → 现在**（今天 18 号就显示 18 天，不再凑 30 天）。
