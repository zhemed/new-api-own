# 图表粒度按范围自动推导 + 轴标签上限 10 — 交付

## 一、粒度规则落点（唯一来源）

`granularityForRangeDays(days)` 提到 **`dashboard/lib/filters.ts`**（经 `lib/index.ts` 导出），
规则按方案 A：`≤1 → hour`、`≤31 → day`、`>31 → week`（原 `≥29 → week` 已废）。

**共用点（grep 可证只有这一套规则）**：
| 入口 | 位置 |
|---|---|
| 默认偏好初始化 | `filters.ts` `buildDefaultDashboardFilters` |
| 查询用 `time_granularity`（default_time） | `filters.ts` `buildQueryParams` |
| 快捷范围 | `models-filter-dialog.tsx` `handleQuickRange` |
| 自定义区间 | `models-filter-dialog.tsx` `handleChange`（用 `rangeDaysOf(start,end)` 推导） |
| 用户图表 | `user-charts.tsx`（由 `selectedRange` 推导）+ `index.tsx` 初始 state |
| 已保存偏好 | `getSavedChartPreferences` 的 range 决定粒度（粒度不再独立保存） |

**移除的选择器**：设置对话框"默认粒度"、筛选对话框"Time Granularity"、用户图表粒度 Tab。
随之删除的死代码：`TIME_GRANULARITY_OPTIONS`、`getSavedGranularity`、`saveGranularity`（knip 因此保持 exit 0）。

## 二、轴标签限制用的是哪个 vchart 能力

`@visactor/vchart@2.1.4`，依据 `node_modules/@visactor/vrender-components/es/axis/type.d.ts` 的
`AxisLabelOverlap`：`autoHide?: boolean`、`autoHideMethod?: 'parity' | 'greedy' | CustomMethod`、
`autoHideSeparation?`、`autoLimit?: boolean`；`LabelAttributes` 另有
`formatMethod?: (value, datum, index, ...) => string`。

采用**两层**：
1. **确定性等距抽稀**（保证 ≤10 且等距，与容器宽度无关）：
   `axisLabelStride(count, 10) = ceil(count/10)` → 30 桶 stride 3 = **10 个标签**、24 桶 stride 3 = **8 个**、7 桶 = **7 个**；
   `formatMethod(value, _, index) => index % stride === 0 ? value : ''`（**数据桶全部保留**，只稀疏标签）。
2. **vchart 自身重叠保护**作为第二道防线：`autoHide: true` + `autoHideMethod: 'parity'`。

落点：`spec_line` / `spec_area` / `spec_model_line` 新增 `axes: [timeAxisOf(n), {orient:'left',type:'linear'}]`；
`spec_user_trend` 的既有 bottom band 轴加 `label: timeAxisLabelConfig(n)`。

## 三、三种范围的桶数与标签数（now 固定注入）

| 范围 | 粒度 | 数据桶 | 可见标签 |
|---|---|---|---|
| 1 天 | hour | **24** | **8**（stride 3） |
| 7 天 | day | **7** | **7**（stride 1） |
| 30 天 | day | **30** | **10**（stride 3） |
| 31 天 | day | 31 | 10 |
| 60 天 | week | ~9 | 9 |

轴仍为 **范围起点 00:00 → 含 now 的桶**（`getCalendarDateRange` 语义未动）。

## 四、测试（新增 14，合计 282）

`dashboard/lib/__tests__/chart-axis.test.ts` 追加：
- 粒度边界 1/2/7/14/30/31/60 → hour/day/day/day/day/day/week（+NaN→day）；
- 桶数：1 天=24（hour）、7 天=7（day）、30 天=30（day，原有）、`rangeDaysOf` 含首尾（30 天 / 同日 / undefined）；
- 标签上限：`axisLabelStride` 对 7/24/30/31 均 ≤10（30→3、24→3、7→1）；
  `timeAxisLabelConfig` 的 `formatMethod` 抽稀行为（index 0/3 显示、1/2 为空）+ `autoHide===true`；
  **spec 断言**：`processChartData(...).spec_model_line.axes[0]` 是 bottom band 且带抽稀 label。

## 五、七条门

typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **282 pass / 0 fail**（268→+14）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 六、需要 Lead 定夺（超出我的写范围）

`src/features/system-settings/content/dashboard-section.tsx:155` 还有**第三个**"Default time granularity" 选择器
（写范围外）。粒度已改为按范围自动推导，**这个设置现在不再有任何效果**。
建议一并移除（或改成只读"按范围自动"）；同时 `Select time granularity`、`Time Granularity`
两个 i18n 键已无引用（0 refs），可随该改动一起清理。我未触碰该文件与这两个键。

## 七、补充：移除 system-settings 的死粒度控件

- **改动**：`system-settings/content/dashboard-section.tsx` 删除 `<FormField name='DataExportDefaultTime'>`
  整块（"Default time granularity" 选择器）——粒度已由范围自动推导，该控件无任何效果。
  连带删除仅它使用的 `granularityOptions` 与未使用的 `Select*` 导入。
- **后端字段保留**（未改任何 Go 代码）：`DataExportDefaultTime`
  声明 `common/constants.go:30`、持久化 `model/option.go:160,552-553`、对外 `controller/misc.go:85`
  （`data_export_default_time`）。前端**已无任何读取点**（`grep data_export_default_time` 只剩一个
  localStorage 键常量同名，与后端无关）。表单 schema / defaultValues / section-registry 的校验回写
  均**保留** → 保存设置时该值原样回写，不会丢失既有配置。
- **i18n 清理**（逐键 grep 确认 0 引用后删除，5 键 × 7 语言 = 35 条）：
  `Default time granularity`、`Select granularity`、
  `UI granularity only &mdash; data is still aggregated hourly`、`Time Granularity`、`Select time granularity`。
  `Hour`/`Day`/`Week` 仍有引用（66/154/20）→ 保留。`i18n:check` 4166 键七语言齐备。
- **另发现一处"既有"死键**（非本次暴露，HEAD 中就已 0 引用）：
  `Choose the default charts, range, and time granularity for model analytics.` —— 已上报，未删。
- **门**：typecheck 0 ｜ lint 0/0 ｜ `bun test` **282 pass / 0 fail**（基线持平）｜ format:check 0 ｜
  build 0 ｜ knip **exit 0** ｜ i18n:check 0。
