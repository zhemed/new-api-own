# 图表粒度按范围自动推导 + 轴标签上限 10

## 背景（Lead 已查清）
"刻度密密麻麻"的根因是**范围↔粒度错配**：`DEFAULT_TIME_GRANULARITY='hour'` 且粒度独立保存；
`granularityForRangeDays()` 只在点快捷范围按钮时才跑 → 30 天 × hour = 720 桶。
旧版疏是因为轴最多 7 点（该行为已废弃）。

## Requirements

### R1 桶粒度由范围自动决定（唯一来源）
- 规则：`≤1 天 → hour`、`2–31 天 → day`、`>31 天 → week`。
- 把现藏在 `models-filter-dialog.tsx` 的 `granularityForRangeDays` 提到**共享位置并导出**，
  **所有入口统一**：默认偏好初始化、快捷范围、自定义区间、**查询用的 `time_granularity`**。
- 设置里移除"默认粒度"选择器、筛选对话框移除粒度选择器（或改只读"按范围自动"）；
  涉及 i18n 键按流程清理并跑 `i18n:check`。
- 桶数上限仍用 `MAX_CHART_AXIS_BUCKETS` 兜底；30 天应为 **30 个日桶**。

### R2 轴标签数量上限（视觉疏朗关键）
- 类别轴（`xField:'Time'`）默认每类都画标签 → 显式限制：
  优先 vchart 官方 `axes[].label.autoHide` + `autoLimit`（**先查 `@visactor/vchart` 类型定义确认支持**）；
  不支持则用 step 类配置或**只对标签等距抽稀**（数据桶保持，标签 ≤10）。
- 目标：**任何范围可见标签 ≤10 且等距**；1 天（24 桶）也 ≤10。

## 验收
- 30 天：30 个日桶、可见标签 ≤10、轴仍 00:00 → now；
- 1 天：24 个小时桶、可见标签 ≤10；
- 7 天：7 个日桶；
- 查询 `time_granularity` 与图表桶粒度一致（同一函数）；
- 测试：粒度推导（1/2/7/14/30/31/60 天边界）+ 桶数（30 天=30、1 天=24）+ 标签上限（断言 spec 字段）。

## Constraints
写范围：`web/src/features/dashboard/**`（可动 `constants.ts`/`lib/filters.ts`/`lib/charts.ts`/两个 dialog）。
**不要动** `lib/time.ts` 的 `getCalendarDateRange` 语义。
不 commit/push；不碰实例。

## Acceptance Criteria
- [ ] AC1：粒度统一由共享函数推导（grep 可证无第二套规则）。
- [ ] AC2：粒度/标签选择器已移除或只读。
- [ ] AC3：三种范围桶数与标签数达标（测试可证）。
- [ ] AC4：七条门全绿；`bun test` ≥268 pass。
- [ ] AC5：回报含落点、vchart 能力依据、桶数/标签数、命令输出。
