# 调查：Token 格式化在 ≥1e9 时是否卡在 M 单位

## 用户担心（2026-10-18）

「1.68M 我感觉后边会出问题，如果后边用到 1000M 是不是单位一直是 M 啊」

## 只读调查项

1. `formatTokens`（及 `formatTokenValue`、`renderQuotaCompat` 的 tokens 分支）的档位实现：有没有 B / T 档，≥1e9 时输出什么；
2. 所有消费点（消耗分布总计/轴/tooltip、模型/用户图表、其它 token 展示）；
3. 本实例数据规模与增长潜力（当前 token 量级、30 天口径最大值、1000M 需要多久）；
4. 若确实卡在 M，给出最小补法（B/T 档、或统一循环档位）与测试建议，并注意与既有 K/M 文案风格一致、不要影响费用口径。

## Acceptance Criteria

- [ ] 给出格式化函数实现与行号，明确 ≥1e9 的实际输出
- [ ] 列出消费点清单
- [ ] 给出补法与风险评估（不改代码，等用户决定）

## 调查结论（2026-10-18，只读）

### 确认：**没有 B 档，≥1e9 会一直挂在 M** ✗

`web/src/lib/format.ts:193`：

```ts
export function formatTokens(tokens: number): string {
  if (tokens === 0) return '-'
  if (tokens < 1000) return tokens.toString()
  if (tokens < 1000000) return `${(tokens / 1000).toFixed(1)}K`
  return `${(tokens / 1000000).toFixed(2)}M`   // ← 再无分支
}
```

实际输出：

| 原始值 | 现在显示 | 应有 |
|---|---|---|
| 999 | `999` | 999 ✓ |
| 1_682_465 | `1.68M` | ✓ |
| 999_999_999 | `1000.00M` ✗ | `1.00B` |
| 1_000_000_000 | `1000.00M` ✗ | `1.00B` |
| 1_234_567_890_123 | `1234567.89M` ✗ | `1.23T` |

### 另外发现：**同名函数有两份**

- `web/src/lib/format.ts:193`（`formatTokens`）——排行榜/详情/图表等多处使用；
- `web/src/features/rankings/lib/format.ts:19`（`formatTokens`）——同名、同样缺 B 档；
- 另有 `charts.ts:59` 的 tokens 分支直接用 `toLocaleString()`（显示 `1,682,465`，无单位问题 ✓）。

### 消费点（受影响的）

排行榜 `model-leaderboard` / `models-section`（轴、tooltip、总计）/ `market-share-section`、
用量详情对话框（audio/text token）、`dashboard/lib/charts.ts:193`（消耗分布总计/轴/tooltip）等 **14+ 处**。

### 规模判断

本实例 30 天累计约 **1.68M** token → 到 1e9 需约 **595 倍**增长；但排行榜是**平台级**统计，
长期（或流量大的部署）完全可能越过 1e9，届时显示 `1000.00M` 会显得很不专业 ✗。

### 建议补法

- 统一到一个共享 formatter：档位 `K → M → B → T`（或循环 `['', 'K', 'M', 'B', 'T']`），
  K 一位小数、M/B/T 两位小数；删除 rankings 里的重复实现（避免两份各改一次）；
- 边界测试：`999 / 1000 / 999_999 / 1_000_000 / 999_999_999 / 1_000_000_000 / 1e12`；
- 不改费用口径（`renderQuotaCompat` 的货币分支保持原样）。

## 实现结果（2026-10-18，v0.0.24）

- `src/lib/format.ts`：`formatTokens` 改为 `TOKEN_UNITS` 表驱动，档位 **K/M/B/T**；
  进位处理：先按原档 `toFixed`，结果 ≥1000 则升一档并沿用原档小数位重算
  （`999_999 → 1.0M`、`999_999_999 → 1.00B`、`999_999_999_999 → 1.00T`）；
  非有限值 → `'-'`（旧实现会吐 `NaNM`）；负值保留符号；
- **删除 rankings 侧重复实现**，`grep "function formatTokens"` 全仓仅 1 处；3 个调用方改 import `@/lib/format`；
- 新增 **15** 个测试（含进位边界 999_949/999_950）；`bun test` 300 → **315**；七条门全绿。

### 调用方显示变化（需知悉）

- rankings 页：`0` 由 `'0'` 变 `'-'`；`<1000` 不再带千分位（`1,234 → 1234`）；小数位固定化。
- `charts.ts:193`（消耗分布）不受影响（它自己把 0 转成 `'0'` ✓）。
