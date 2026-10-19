# 市场份额卡片也跟随偏好：面积图

## 用户要求（2026-10-19）

截图：市场份额卡片（100% 堆叠柱状，中文日期，最近 7 天）。
「这个能不能匹配一下，也变成面积图」→ 与「热门模型」一致，**跟随已保存偏好**（偏好=面积图 → 面积图）。

## 只读调查

1. 现有 spec：`market-share-section.tsx` 的 `type/stack/percent` 与 100% 归一化方式；
2. 面积图是否支持 100% 堆叠（vchart：`type:'area'` + `stack:true` + `percent`?），若不支持则前端**预归一化**为百分比；
3. 标签抽稀（上一轮新加的 `rankingLabelFilter`）是否已覆盖此卡片。

## Acceptance Criteria

- [ ] 采用偏好（area/bar），默认 area
- [ ] 100% 语义不变（每桶合计 100%），y 轴仍 0–100%
- [ ] 标签仍 ≤10 且首尾必在
- [ ] 七条门全绿；发版；Lead 真机截图核对

## 实现结果（2026-10-19，v0.0.34）

- 新增 `rankingChartShape(pref)`：`{ type, stack: true, ...(bar ? { paddingInner: 0.12 } : {}) }`；
  `market-share-section.tsx` 用它替换写死的 `type:'bar' / stack / paddingInner`；
- **未动**：`yField:'share'`（数据层已归一化到 0–1）、y 轴 `min:0/max:1`、标签 `share*100%`、tooltip 百分比、
  `dataFilter: rankingLabelFilter(bucketCount)` —— **100% 语义与标签抽稀均不变** ✓；
- 新增 3 个测试（area/bar 形态、100% 轴与抽稀契约）→ `bun test` 366 → **369**；七条门全绿；发 v0.0.34。

### Lead 真机验证（升级到 v0.0.34 后）

- `?period=week`（单桶）：面积图退化为一个点 @100% ✓；
- `?period=month`（多桶）：**填充面积 @100%**、y 轴 0–100%、中文日期 `10月7日/10月9日/10月18日` ✓；
- 截图：`.verify-scratch/v0034-market-share-3.png`、`v0034-market-share-month.png`。

### 已知边界

停留在排行榜页时切换偏好不会热更新（需离开再回来）；`models-section` 亦然，本次未做偏好订阅。
