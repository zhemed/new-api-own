# 修复：消耗分布 tooltip 模型值为 0

## 用户报告（2026-10-18）

截图：tooltip 显示 `10-09 / 总计: 1.16M / deepseek-flash 0`
→ **总计正确（Token 口径 ✓）**，但**每模型那行是 0** ✗。

## 排查方向

- 「总计」与「每模型」在 tooltip 里由不同分支取值：
  - 总计来自维度合计（`dimension.content` 的合计/`TimeSum` 等）；
  - 每模型来自 `mark.content` 的 `value(datum)`。
- 重点看是否仍在读**已删除的旧字段**（如 `rawQuota`）或读错字段名（`Usage` vs `rawTokens`），
  导致 `Number(undefined) || 0` → 0。
- 同时核对柱状图（`spec_bar`）与面积图（`spec_area`）两套 spec 是否都改到了；以及「Other」合并桶分支。

## Acceptance Criteria

- [ ] 定位到具体行（文件:行号）与错误字段
- [ ] 修复后 tooltip 每模型值 = 该模型该时间桶的 Token 数，与总计自洽
- [ ] 补回归测试：构造两模型两时段数据，断言每个 (time, model) 的 tooltip 值等于其 token，且同桶合计 = 总计
- [ ] 七条门全绿；发版并真机截图验证
