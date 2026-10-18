# 消耗分布改统计 Token — 交付

## 一、改了哪些取值点（`dashboard/lib/charts.ts`，均在 `processChartData` 内）

| 位置 | 改动 |
|---|---|
| 格式化器 | 删掉 `formatQuotaValue`/`formatQuotaTotal`（`renderQuotaCompat` 会跟随货币显示设置），新增 **`formatTokenValue`**：`>0` 走共享 `formatTokens`，`0 → '0'`。**不读任何货币配置** |
| 柱状图取值 `lineValues` | `rawQuota` → **`rawTokens = stats.tokens`**，`Usage = rawTokens`（不再 `/quotaPerUnit` 转 USD） |
| 面积图取值 `areaValues` | 同上；"Other" 合并桶同样按 token 累加 |
| 面积图 Top-Model 选择 | **改按 token 排名**（原先按 quota 排名 → 显示 tokens 时会出现"小 token 的模型占了大色块"） |
| y 轴（`spec_line`/`spec_area` 的 left linear 轴） | 原先无 formatMethod（自动数字），现加 `label.formatMethod = formatTokenValue` |
| tooltip | `mark.content` 与 `dimension.content` 的取值 → `formatTokenValue(rawTokens)`；`updateContent` 的合计/溢出桶同理 |
| 总计 | `totalQuotaRaw(quota 求和)` → **`totalTokens(token_used 求和)`**；`totalQuotaDisplay = formatTokenValue(totalTokens)`；空数据分支 → `'0'` |
| 移除 | `processChartData` 里的 `getCurrencyDisplay()/quotaPerUnit`（已无用途） |

**未改**：`spec_pie`（次数占比）、`spec_model_line`（`yField:'Count'`）、`spec_rank_bar`（次数排行）、
`processUserChartData`（用户图表仍按 quota）、以及**标题「消耗分布」与其 i18n 键**（卡片标题在
`consumption-distribution-chart.tsx`，本次未触碰；`{t('Total:')}` 前缀保留）。

## 二、新格式示例（共享 `formatTokens`：`<1000` 原样、`<1e6` 一位小数 K、`≥1e6` 两位小数 M）

| 原始值 | 改前（USD 口径） | 改后 |
|---|---|---|
| `1682465` | `$3.36…` 之类的金额 | **`1.68M`** |
| `1000` | 金额 | **`1.0K`** |
| `42` | 金额 | **`42`** |
| `0` | `$0.00` | **`0`** |
| 总计 `1_683_465` | `总计：$0.04` | **`总计：1.68M`** |

tooltip 单模型行同样为 `1.68M` / `1.0K`；维度合计为 `1.68M`；y 轴刻度 `0 / 42 / 1.0K / 1.68M`。**任何位置都不再出现 `$`**。

## 三、测试（新增 3 个）

`dashboard/lib/__tests__/consumption-tokens.test.ts`：
1. **取值口径**：构造两行（quota 500/900 但 token 1_682_465/1_000）→ 断言 bar 与 area 的 `Usage`
   分别为 token 值，且**排序按 token**（若仍按 quota，gpt-5 会排在 claude 之后）；
2. **无货币符号**：`totalQuotaDisplay === '1.68M'` 且不含 `$`；y 轴 `formatMethod(1_682_465) === '1.68M'`、
   `(42) === '42'`、`(0) === '0'`；tooltip `value({rawTokens:1_682_465}) === '1.68M'` 且不含 `$`；
3. **缺失回落**：行内没有 `token_used` → 所有 `Usage === 0`、总计 `'0'`、tooltip `({}) === '0'`（图表不变空、不报错）。

## 四、七条门

typecheck **0 错误** ｜ lint **0 warnings / 0 errors** ｜ `bun test` **286 pass / 0 fail**（283 → +3）｜
format:check exit 0 ｜ build exit 0 ｜ **knip exit 0** ｜ i18n:check exit 0。

## 五、边界
只改 `dashboard/lib/charts.ts` + 新增测试；无 Go 改动、未动标题/i18n 文案、未动其它图表。
