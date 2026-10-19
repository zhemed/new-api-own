# 市场份额卡片跟随偏好图表类型 — 交付

## 一、diff 摘要

### `rankings/lib/chart-preference.ts`（新增 `rankingChartShape`）
```ts
export function rankingChartShape(preference: string | undefined): {
  type: 'area' | 'bar'; stack: true; paddingInner?: number
} {
  const type = rankingChartType(preference)
  return {
    type,
    stack: true,                                       // 预归一化 share → 面积图仍是 100% 堆叠
    ...(type === 'bar' ? { paddingInner: 0.12 } : {}),  // 柱状专用，面积图下无意义 → 不出现该键
  }
}
```

### `components/market-share-section.tsx`
```diff
     return {
-      type: 'bar' as const,
+      ...rankingChartShape(getSavedChartPreferences().consumptionDistributionChart),
       data: [{ id: 'vendor-share', values: orderedPoints }],
       xField: 'label',
       yField: 'share',
       seriesField: 'vendor',
-      stack: true,
-      paddingInner: 0.12,
```
**未改动**：`yField: 'share'`（0–1 预归一化）、y 轴 `min: 0` / `max: 1`、标签 `share*100%`、
tooltip 的 `xx.x% · tokens`、bottom 轴 `dataFilter: rankingLabelFilter(bucketCount)`（未重复添加）。

## 二、`paddingInner` 取舍
**按类型条件带上，而不是直接删**：
- 它只影响柱状的类目间距（视觉留白），对面积图无意义 → 面积图下**不出现该键**；
- 但柱状仍是默认回退（偏好非 area），删掉会让柱状贴得比现在更紧 → 保留 `0.12` 只在 `type==='bar'` 时出现。

## 三、偏好订阅方式（与 models-section 完全一致）
两处都在 `useMemo` 内调用 `getSavedChartPreferences()`（localStorage 读取）。
所以：在仪表盘改完偏好 → 进入排行页（组件挂载）即按新偏好渲染，**不需要刷新页面**。
⚠️ 边界：如果**停留在排行页**不动、在别处改了偏好，本页不会热更新（models-section 也一样）；
需要"同页热更新"得引入偏好事件/订阅，属额外设计，本次未做。

## 四、测试（新增 3，`preferences-and-labels.test.ts` 现 16 个）
1. `rankingChartShape('area')` → `type==='area'`、`stack===true`、**无 `paddingInner` 键**（`Object.hasOwn` 断言）；
2. `rankingChartShape('bar')` → `type==='bar'`、`stack===true`、`paddingInner===0.12`；
3. 市场份额 spec 的源码契约：`...rankingChartShape(`、`yField: 'share'`、`min: 0`、`max: 1`、
   `dataFilter: rankingLabelFilter(bucketCount)` 全部在。

> 为什么第 3 条是源码契约：该卡片挂载 VChart，需真实 canvas（happy-dom 无、本机无 `canvas` 包、禁联网安装），
> 无法渲染断言；纯几何规则已由 1、2 两条真单测覆盖。

## 五、七条门
typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **369 pass / 0 fail**（366 → +3）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 六、边界
只改 `market-share-section.tsx` + `chart-preference.ts`（+ 测试）。未 commit。
