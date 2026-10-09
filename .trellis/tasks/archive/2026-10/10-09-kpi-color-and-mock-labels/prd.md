# 前端：KPI 阈值上色 + mock 区块标注

## Goal

两件独立但同批交付的前端一致性工作：
① 让控制台两处汇总 KPI（`avgTps`）与日志/详情页**共用同一套阈值配色**；
② 给仍然使用**按模型名哈希生成的假数据**的区块加**清晰、克制**的「示例数据」标注，做认知纠偏。

## Requirements

### ① KPI 上色（同一套阈值，不得另起一套）
- 位置：`web/src/features/dashboard/components/models/performance-overview.tsx:156` 与
  `web/src/features/dashboard/components/overview/performance-health-panel.tsx:126`，
  两处都渲染 `formatThroughput(summary.avgTps)`。
- 必须**复用** `web/src/features/performance-metrics/lib/format.ts` 的 `getTpsTextClass()`
  （阈值 50/100、**左闭右开**、无数据 → 中性灰）。
- **不得**新增第二套阈值或配色常量。
- 若这两个组件不支持 `valueClassName` 之类的类名注入，用**最小改动**支持（或就地包一层 `span`）。

### ② mock 区块标注（只标真的那块）
- `web/src/features/pricing/lib/mock-stats.ts` 的数据是**按模型名哈希生成**的假值，被三处消费：
  `model-details-charts.tsx`（延迟/吞吐图表）、`model-details-uptime-sparkline.tsx`（30 天可用率）、
  `model-details-api.tsx`。
- 要求：给这些区块中**确实使用 mock 的部分**加标注（小字或徽标），文案形如「示例数据，非实测」。
- **必须逐一核对数据来源**：区块内来自真实接口的字段**不得误标**；标注范围要能被复核（写明哪些字段是 mock）。
- **不得**给出"自动/实时"之类会加剧误解的措辞。

### ③ i18n
- 新文案必须走临时 `scripts/add-missing-keys.mjs` + `node scripts/sync-i18n.mjs`，**禁止手改 `locales/*.json`**；七语言齐备，
  `i18n:check` 通过。

### ④ 回归测试
- KPI 颜色边界：50 / 100 的**左闭右开**语义 + 无数据中性灰（在阈值函数与组件两层至少覆盖其一，优先组件可见类名）。
- 标注：在**真正 mock** 的区块出现；在**真实数据**区块（如 API 区块里来自接口的字段）不出现。

### ⑤ 质量门
- `typecheck / lint / test / build / knip / i18n:check` 全绿 + `format:check` 通过。
- `bun test` 基线 **213 pass / 0 fail**，**不得下降**（新增测试应使其上升）。
- `knip` 若仍为基线红，需照旧给出对照说明。

## Constraints

- 只改：`web/src/features/pricing/components/**`、`web/src/features/performance-metrics/**`、
  ① 明确点名的两个 dashboard 组件文件、以及新增的 i18n 文案与本次新增的测试文件。
- **不动** lint 命中的其它文件（由 `fix-frontend-open-items` 同时修改）——开工前先核对 lint 文件清单，若与①点名的文件冲突，先向 Lead 确认再动手。
- 不 commit / push；不碰用户实例；不新增依赖；不引入浏览器外呼。

## Acceptance Criteria

- [ ] AC1：两处 KPI 使用 `getTpsTextClass()` 的类名渲染，且仓库内无第二套 TPS 阈值/配色。
- [ ] AC2：给出"哪些字段是 mock / 哪些是真实"的逐项核对结论（文件:行）。
- [ ] AC3：mock 区块可见「示例数据」标注；真实字段所在区块无该标注。
- [ ] AC4：新增文案七语言齐备，`i18n:check` 通过；未手改 locale JSON（临时脚本用后删除）。
- [ ] AC5：新增回归测试覆盖颜色边界与标注有无；`bun test` ≥ 213 pass / 0 fail。
- [ ] AC6：六条命令 + `format:check` 输出齐备；改动仅限允许范围；未 commit/push。

## Notes

- 结论文档留 `findings.md`。
- 写法参考：`.trellis/spec/` 与 `web/AGENTS.md`（i18n、测试、可读性约定）。
