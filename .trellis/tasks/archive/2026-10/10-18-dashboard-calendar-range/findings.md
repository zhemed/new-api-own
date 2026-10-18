# 仪表盘时间范围改日历制 + x 轴按范围铺满 + 29→30 天 — 交付

## 一、诊断确认

Lead 的两条根因**均确认**，无修正：
1. `getRollingDateRange()` 是 `now - days*24h` 滚动窗口 → 被 4 处消费（filters.ts:146、
   models-filter-dialog.tsx:137/159、user-charts.tsx:90）。
2. `charts.ts` 的 `fillTimePoints()` 用 `max(created_at)` 当终点反向补 7 个桶 → 轴首尾被数据拽。

补充发现：`processChartData` 的时间聚合键原本是**标签字符串**（`formatChartTime` 的输出），
所以 week 粒度下同周不同天的数据会落进**不同**标签 —— 改成数值桶键后一并修正。

## 二、改动清单

| 文件 | 改动 |
|---|---|
| `src/lib/time.ts` | 用 **`getCalendarDateRange(days, from)`** 取代 `getRollingDateRange`：起点=范围第一天本地 00:00（`setDate` 日历算术，跨月/跨年/DST 正确），终点=now；非法/≤0 → 按 1 天 |
| `dashboard/lib/filters.ts` | `buildDefaultDashboardFilters` 改用日历制（API 查询区间随之变） |
| `models-filter-dialog.tsx` ×2 | 同上 |
| `user-charts.tsx` | 同上（查询区间） |
| `dashboard/lib/charts.ts` | 新增 `ChartTimeRange`、`toBucketStart()`、`buildTimeBuckets()`；`processChartData(..., range?)`；聚合键改为**桶起点(ms)**；轴由 range 生成（首=起点桶、末=含 now 的桶），无 range 时保留旧行为 |
| `dashboard/constants.ts` | `TIME_RANGE_PRESETS` 29→**30 天**；新增 `MAX_CHART_AXIS_BUCKETS = 1000`（最大组合 30 天×小时=720，仅防病态范围） |
| `model-charts.tsx` / `consumption-distribution-chart.tsx` | 新增 `range?: ChartTimeRange` prop 并透传 |
| `dashboard/index.tsx` | 由 `modelFilters` 派生 `modelChartRange` 传给两张图 |
| 7 个 locale | `29 Days` → `30 Days`（en/zh/zh-TW/ja/ru/fr/vi），走临时脚本 + `sync-i18n`，旧键已移除 |

## 三、前后行为对照（示例：now = 2026-01-15 13:45）

| | 改前 | 改后 |
|---|---|---|
| **1 天** | 2026-01-14 13:45 → 01-15 13:45（滚动 24h） | **2026-01-15 00:00 → now** |
| 1 天·轴 | 数据空的桶不显示；末点=最后一条数据（如 22:00） | **00:00, 01:00 … 13:00 共 14 桶**，无数据桶补 0 |
| **30 天** | 2025-12-16 13:45 → now | **2025-12-17 00:00 → now**（30 个日历日） |
| 30 天·轴（day） | 末点=最后一条数据 | **12-17 … 01-15 共 30 桶**，逐日一格 |
| 30 天·轴（hour） | — | 720 桶（< 1000 上限） |

## 四、测试（新增 16 个）

- `src/lib/__tests__/calendar-date-range.test.ts`（5）：1 天=今天 00:00→now；
  30 天=29 天前 00:00 且跨年（2026-01-15 → 2025-12-17）、跨月（3-3 → 2-25）、
  月初 00:05 保持当月 1 日、非法/≤0 按 1 天。全部用**本地时间构造**，不依赖真实时钟。
- `src/features/dashboard/lib/__tests__/chart-axis.test.ts`（11）：
  `toBucketStart`（hour/day/week-Monday）、`buildTimeBuckets`（首=起点桶、末=含 now 的桶、
  30 天=30 桶、末桶**随 now 变化**、week 按周一跳）、
  `processChartData`（range 下轴铺满 + 空桶补 0 + 真实数据落位；无 range 保持旧行为）、
  预设（末档=30 天、`[1,7,14,30]`、最宽组合 < 1000 桶）。

## 五、六条命令

| 命令 | 结果 |
|---|---|
| `bun run typecheck`（tsgo -b --force） | **0 错误** |
| `bun run lint` | **0 warnings / 0 errors** |
| `bun test` | **263 pass / 0 fail**（基线 247 → +16） |
| `bun run build` | exit 0 |
| `bun run knip` | **exit 0** |
| `bun run i18n:check` | exit 0（4171 键 × 7 语言齐备） |
| 附 `format:check` | exit 0 |

## 六、边界

- 只改：`dashboard/**`、`lib/time.ts`、`i18n/locales/*.json`（+ 2 个新测试目录）。
- **user charts 的 x 轴仍未按范围铺满**（`processUserChartData` 没有 `fillTimePoints`，
  其轴本就由数据桶构成）；本次只统一了它的**查询区间**。如需同样铺满，可作为后续小任务。

## 七、补充：29→30 的存量偏好迁移

**问题**：`getSavedChartPreferences()` 用 `isTimeRangePresetDays()` 校验存档值，预设 29→30 后
用户**已存的 29 变成非法值 → 静默回退默认 1 天**（用户截图正是选了 29 天）。

**做法**：
- 新增 `migrateLegacyTimeRangeDays(value)`（`lib/filters.ts`），**只**把 `29` 归一成 `30`，
  其余值原样返回交给 `isTimeRangePresetDays` 判定（不猜测其它值）；函数注释写明"29 已改为 30，兼容旧存量"。
- 在 `getSavedChartPreferences()` 的解析里先迁移再校验。
- **未做**回写（Lead 列为可选）：getter 会在渲染期间被调用，此时写 localStorage 属于
  "渲染期副作用"，收益（存档值从 29 变 30）不抵风险；下次用户点选/保存设置时会自然写成 30。

**测试**（`lib/__tests__/chart-preferences-migration.test.ts`，5 个，happy-dom localStorage）：
①存 29 → 读 30；②存 30 → 30；③存 5 → 回退默认；④未存 → 默认；
⑤`migrateLegacyTimeRangeDays` 只转 29（30/7/undefined/'29' 均原样）。

**门**：typecheck 0 ｜ lint 0/0 ｜ `bun test` **268 pass / 0 fail**（263→+5）｜ format:check 0 ｜
knip **exit 0** ｜ i18n:check 0 ｜ build exit 0。
