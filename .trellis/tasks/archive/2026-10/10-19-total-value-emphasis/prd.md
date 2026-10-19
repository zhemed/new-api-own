# 图表总计数值改用正文色

## 问题
图表标题里的「总计：1.68M」数字颜色太浅：两处都把**标签与数值**放在同一个
`text-muted-foreground text-xs` 的 span 里，主指标被当成辅助信息弱化。

## 要改（两处都改，保持一致）
- `consumption-distribution-chart.tsx` 的 `{t('Total:')} {chartData.totalQuotaDisplay}`
- `model-charts.tsx` 的 `{t('Total:')} {chartData.totalCountDisplay}`

改法：标签保持 `text-muted-foreground`，**数值单独包** `<span className='text-foreground font-medium tabular-nums'>`；
**字号不变**（仍 `text-xs`），不动布局/对齐，不硬编码颜色。

## 不要改
其它"总计"（tooltip 行内、排行榜汇总）保持现状（辅助值弱化色合理）。

## 检查
- 组件级轻量断言：总计数值元素带 `text-foreground` 与 `tabular-nums`，标签仍 muted；
- 七条门全绿（基线 **341**，只能增）。

## 写范围
上述两个文件（+ 测试）。不 commit/push。
