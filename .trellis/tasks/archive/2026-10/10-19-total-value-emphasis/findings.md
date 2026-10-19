# 图表总计数值改用正文色 — 交付

## 一、两处 diff 摘要（结构相同）

`consumption-distribution-chart.tsx`（`totalQuotaDisplay`）与 `model-charts.tsx`（`totalCountDisplay`）：

```diff
 <span className='text-muted-foreground text-xs'>
-  {t('Total:')} {chartData.totalQuotaDisplay}
+  {t('Total:')}{' '}
+  <span
+    data-testid='chart-total-value'
+    className='text-foreground font-medium tabular-nums'
+  >
+    {chartData.totalQuotaDisplay}
+  </span>
 </span>
```

- 标签仍是 `text-muted-foreground`，外层仍是 `text-xs` → **字号与布局不变**；
- 数值用 `text-foreground`（正文色，随明暗主题自动切换，未硬编码颜色）+ `font-medium` + `tabular-nums`（等宽数字，避免跳动）；
- `{' '}` 显式保留原有空格（JSX 换行会被折叠）。

## 二、未改动（按要求）
- tooltip 行内总计、排行榜汇总的"总计"：仍为辅助弱化样式（`charts.ts` 无任何 `chart-total-value` 元素，测试有断言）；
- 未新增/删除 i18n 键。

## 三、断言（5 个）

`components/models/__tests__/chart-total-emphasis.test.tsx`：
1. 两个组件各自：源码中存在 `data-testid='chart-total-value'` 且类名含 `text-foreground` / `font-medium` / `tabular-nums`；
2. 两个组件各自：`Total:` 仍包在 `text-muted-foreground text-xs` 的外层 span 中，且**没有**把 `text-foreground` 与更大的字号组合（字号不变）；
3. `lib/charts.ts` 不含该 testid → tooltip / 排行汇总未被误改。

**为什么是源码契约断言而不是渲染断言**：这两块都会挂载 VChart（`@visactor/react-vchart`），
需要真实 canvas 2D 上下文；happy-dom 没有，本机也没有 `canvas` 包（安装需联网，红线禁止）。
我确实先尝试了组件渲染，失败在 `new VChart(...)`，因此改为**对两处源码标记的结构断言**（仍覆盖"数值单独包 + 类名正确 + 标签仍弱化 + 字号未变"四条契约）。

## 四、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **346 pass / 0 fail**（341 → +5）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 五、边界
只改两处组件 + 1 个测试；未 commit。
