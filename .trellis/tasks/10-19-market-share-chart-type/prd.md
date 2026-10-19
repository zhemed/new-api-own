# 市场份额卡片跟随偏好图表类型

## 背景
用户要求市场份额卡片与「热门模型」一致 —— **也跟随已保存偏好**（偏好=面积图 → 面积图）。
该图数据层已预归一化（`yField: 'share'`，值域 0–1，y 轴 `min:0/max:1`，标签 `share*100%`），
不依赖 vchart 的 `percent`，所以换 `type:'area'` + 保留 `stack:true` 时 **100% 堆叠语义不变**。

## 要改
1. `type: 'bar' as const` → `rankingChartType(getSavedChartPreferences().consumptionDistributionChart)`（与 `models-section` 同一套）；
2. 保留 `stack: true`；`paddingInner`（柱状专用）→ 按类型决定是否带上；
3. 组件订阅偏好（与 models-section 相同方式）；
4. **不改** y 轴/标签/tooltip 的百分比语义。

## 测试
- 偏好 area → spec `type==='area'`、`stack===true`、`yField==='share'`、y 轴 `max===1`；
- 偏好 bar → `type==='bar'`；
- 断言该卡片也用了 `rankingLabelFilter`。

## 门
七条门全绿（前端基线 **366**，只能增）。

## 写范围
`market-share-section.tsx` + 测试。不 commit/push。
