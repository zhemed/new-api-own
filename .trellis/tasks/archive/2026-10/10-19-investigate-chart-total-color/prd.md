# 调查：消耗分布「总计」数值颜色偏浅

## 用户反馈（2026-10-19）

截图：`消耗分布 总计：1.68M`，用户觉得 **1.68M 这个数字颜色太浅**。

## 只读调查

1. 该数值的渲染位置（哪个组件、哪个 className）；
2. 当前配色（`text-muted-foreground` / `text-muted-foreground/xx` / 透明度）与标题的对比；
3. 是否与仓库其它"总计/汇总"数值的样式一致（例如 tooltip 的 Total、排行榜总计）；
4. 给改法建议（提高对比度、是否加 `tabular-nums`、是否统一到某个语义色）。

## Acceptance Criteria

- [ ] 给出文件:行号与当前 className
- [ ] 给出与同类 UI 的一致性对比
- [ ] 给出 2 个改法建议（含风险：暗色模式）

## 调查结论（2026-10-19，只读）

### 位置与现状

`web/src/features/dashboard/components/models/model-charts.tsx:135-137`：

```tsx
<span className='text-muted-foreground text-xs'>
  {t('Total:')} {chartData.totalCountDisplay}
</span>
```

⇒ **标签和数值共用** `text-muted-foreground`（浅灰）+ `text-xs`（12px）→
值 1.68M 与"总计："同色同级，视觉上被弱化成辅助信息 ✗（而它其实是这一屏最关键的数字）。

### 一致性

- 同一页面其它同类"总计"（tooltip 内的 `Total:`、排行榜/市场份额的汇总）都只是**行内辅助值**，
  用弱化色是合理的 ✓；
- 但**图标题旁的"总计"是主指标**，用同一弱化色属层级不当 ✗。

### 建议改法

1. **推荐（最小改动）**：只把**数值**提亮加粗 —— 标签保持 `text-muted-foreground`，
   数值改 `text-foreground font-medium tabular-nums`（`tabular-nums` 让数字等宽、跳动不抖）；
2. 备选：数值同时 `text-sm`（会更醒目，但可能影响标题行高度，需顺带检查对齐）。

风险低：`text-foreground` 在明暗两种主题下都是正文色，不会出现暗色模式看不清的问题。

## 实现结果（2026-10-19，v0.0.29）

用户选"两处都改"：

- `consumption-distribution-chart.tsx:137` 与 `model-charts.tsx:138`：数值单独包
  `<span data-testid='chart-total-value' className='text-foreground font-medium tabular-nums'>`；
- 标签保持 `text-muted-foreground`、外层仍 `text-xs` → **字号与布局不变**；
- `text-foreground` 随明暗主题（无硬编码色）；`tabular-nums` 让数字等宽不跳动；
- **未动** tooltip 行内总计与排行榜汇总（本就是辅助值）；
- 新增 5 条结构契约断言；`bun test` 341 → **346**；七条门全绿。

### 成员的诚实说明

两处组件会挂载 `@visactor/react-vchart`，需要真实 canvas 2D（happy-dom 无、本机无 `canvas` 包且安装需联网 ✗），
因此断言只能停在**源码结构契约**层面（覆盖：数值单独包裹、类名正确、标签仍弱化、字号未变、未误改其它总计）。
