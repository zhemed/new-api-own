# 消耗分布改统计 Token（标题不动）

## 用户要求（原话）
「消耗分布这几个字不动，总计：$0.04。就是统计金额的 $0.04，换成统计 Token」
⇒ **只换数值口径，不改标题**：标题/图标/视图切换保持「消耗分布」；
「总计：$0.04」→「总计：<Token 数>」；y 轴刻度、tooltip（总计 + 各模型值）、柱/面积高度 → 全部 `token_used`。

## 数据面
`/api/data/` 每行已含 `token_used`；前端 `QuotaDataItem.token_used?: number` 已存在；**无 Go 改动**。
格式化参照 flow 图 tokens 口径（K/M）。

## 改动点（`dashboard/lib/charts.ts`）
- 消耗分布（`spec_bar`/`spec_area`，`barData`/`areaData`，`yField:'Usage'`）取值 → `token_used`；
- y 轴 `formatMethod` / tooltip 的 `formatVal` → Token 格式（**不出现 `$`**）；
- `totalQuotaDisplay` → 总计 Token 数（保留「总计」前缀）；
- **不改** `spec_model_line`（次数）与 `spec_rank_bar`（次数排行）。

## 约束
- 标题「消耗分布」及其 i18n 键**不动**；
- `token_used` 缺失回落 0；
- 测试：①value 取 token_used（两行不同 token 断言数值/排序）②tooltip/总计格式不含 `$` ③缺失→0；
- 七条门全绿（基线 **283**）。

## Acceptance Criteria
- [ ] AC1：取值点全部改为 token_used（grep 可证 spec_bar/spec_area 不再用 quota）。
- [ ] AC2：总计/轴/tooltip 均为 Token 格式且无 `$`。
- [ ] AC3：标题与视图切换未变。
- [ ] AC4：三类测试齐备；七条门全绿；`bun test` ≥283。
