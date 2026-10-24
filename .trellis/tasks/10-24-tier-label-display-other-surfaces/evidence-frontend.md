# evidence-frontend：档位名翻译补齐全部展示入口（3 处）

- 任务板：`task-63`（Trellis 任务 `10-24-tier-label-display-other-surfaces`）
- 执行者：`worker-docs`；基线 HEAD `43bc237`（2026-10-24）
- 改动：3 个生产文件（+3 行 import/调用、details-dialog 6→4 行）+ 2 个新测试文件；**未改 locale**（复用既有键）
- 未提交、未推送；未动 Go 文件、`data/`、实例

## 一、结论

全仓 **4 个展示入口**现已全部走 `getTierDisplayLabel(label, t)`：定价面板（上轮已修）+ 日志列表「详情」列 +
日志详情弹窗「命中阶梯」+ 模型详情价格表「档位」列。**逻辑路径**（row key、`normalizeTierLabel` 比较、
`matchedTierLabel` 高亮、`formatConditionSummary`、表达式编辑/回写、导出/复制）**全部继续用原始标签**，
并有反向测试证明。未知标签原样、空标签 `Default`。**无第 5 个展示入口**（见第四节）。

## 二、`.label` 消费点判定表

| # | 位置 | 用途 | 判定 | 处置 |
|---|---|---|---|---|
| 1 | `pricing/components/dynamic-pricing-breakdown.tsx:302`（移动端卡片） | 渲染档位徽标 | **展示** | 已用 helper（上轮） |
| 2 | `pricing/components/dynamic-pricing-breakdown.tsx:385`（表格列） | 渲染档位徽标 | **展示** | 已用 helper（上轮） |
| 3 | `usage-logs/components/columns/common-logs-columns.tsx:171` | 「详情」列的 `档位 · 价格` 摘要 | **展示** | **本轮改用 helper** |
| 4 | `usage-logs/components/dialogs/details-dialog.tsx:244` | 弹窗「命中阶梯」字段值 | **展示** | **本轮改用 helper**（并去掉空标签时省略该行的旧判断） |
| 5 | `pricing/components/model-details.tsx:1002` | 价格表「档位」列单元格 | **展示** | **本轮改用 helper** |
| 6 | `pricing/components/model-details.tsx:994` | 表格 row key（group 前缀 + 原始 label 或索引） | **逻辑** | 保持原始 |
| 7 | `pricing/components/dynamic-pricing-breakdown.tsx:287` | 移动端高亮 `tier.label === matchedTierLabel` | **逻辑** | 保持原始 |
| 8 | `pricing/components/dynamic-pricing-breakdown.tsx:202-203、357、375-377` | `normalizeTierLabel` 比较（表格高亮） | **逻辑** | 保持原始 |
| 9 | `usage-logs/lib/format.ts:252-253` | `resolveMatchedTier`：用 `normalizeTierLabel` 找命中档位 | **逻辑** | 保持原始 |
| 10 | `pricing/lib/billing-expr.ts:264` | 解析表达式时写入 `tier.label` | **逻辑**（数据源） | 保持原始 |
| 11 | `pricing/lib/tier-expr.ts:56、123、134` | 回写表达式（`tier.label ?? ''`、空值回退 `default` / `tier_N`） | **逻辑** | 保持原始 |
| 12 | `system-settings/models/tiered-pricing-editor.tsx:629` | 档位名 `<Input value={tier.label}>`（操作员编辑，会写回表达式） | **逻辑/编辑** | 保持原始（翻译会把译文写进表达式） |
| 13 | `dynamic-pricing-breakdown.tsx` `formatConditionSummary` | 条件摘要（变量/操作符，不含档位名） | **逻辑** | 未改 |

## 三、改动清单

| 文件 | 改动 |
|---|---|
| `web/src/features/usage-logs/components/columns/common-logs-columns.tsx` | +1 import；`tieredSummary.tier.label \|\| t('Default')` → `getTierDisplayLabel(...)`（1 行） |
| `web/src/features/usage-logs/components/dialogs/details-dialog.tsx` | +1 import；「命中阶梯」行由「有标签才显示」改为始终推入、值取 helper（6→4 行） |
| `web/src/features/pricing/components/model-details.tsx` | +1 import；「档位」列 `tier.label \|\| t('Default')` → `getTierDisplayLabel(...)` |
| `web/src/features/usage-logs/components/__tests__/tier-label-display.test.tsx` | **新增** 4 项测试（日志列表 / 详情弹窗 / 反向高亮） |
| `web/src/features/pricing/components/__tests__/model-details-tier-label.test.tsx` | **新增** 3 项测试（模型详情档位列 / 未知 / 空） |

说明：`getTierDisplayLabel` 内部已处理「未知原样、空 → `t('Default')`」，三处**没有**另写回退逻辑；
本轮**未新增 i18n 键**（复用 `Peak hours` / `Off-peak hours` 等既有键），因此 locale 文件零改动。

## 四、是否还有第 5 个展示入口

全仓 grep（`tier.label` / `tierLabel` / `matchedTierLabel` / `getTieredBillingSummary` / `matched_tier`）
后确认：

- `getTieredBillingSummary` 的消费点只有 2 处（日志列表列、详情弹窗），本轮都已处理；
- `matched_tier` 的两处消费：`resolveMatchedTier`（逻辑）与 `DynamicPricingBreakdown` 的 `matchedTierLabel`（逻辑+展示均已正确）；
- 日志导出/复制：usage-logs 内**没有**导出档位名的路径（CSV/剪贴板不涉及 tier label）；
- 表达式编辑器里的 `<Input value={tier.label}>` 是**编辑**入口（写回表达式），必须保持原文，不属展示入口。

结论：**无第 5 个展示入口**。

## 五、测试

`bun test src/features/usage-logs/ src/features/pricing/` → **31 pass / 0 fail**（8 文件）。

新增用例与断言：

| 文件 | 用例 | 断言 |
|---|---|---|
| usage-logs `tier-label-display.test.tsx` | 日志列表「详情」段 | zh 下 `offpeak` → `低谷`；原始 `offpeak` 不出现 |
| 同上 | 未知档位名 | `my_custom_tier` 原样显示 |
| 同上 | 详情弹窗「命中阶梯」 | 字段名 `命中阶梯` + 值 `低谷`；原始 `offpeak` 不出现 |
| 同上 | **反向测试** | `matched_tier='offpeak'` 时有「已命中」徽标；`matched_tier='低谷'`（译文）时**0 个** |
| pricing `model-details-tier-label.test.tsx` | 价格表档位列 | `offpeak`/`peak` → `低谷`/`高峰`；原始标签不出现 |
| 同上 | 未知档位名 | 原样显示 |
| 同上 | 空档位名 | 显示 `默认`（helper 的 `Default` 回退） |

说明：「空标签 → Default」在**两个日志入口不可达**——`resolveMatchedTier` 要求命中标签的归一化值非空
（`l1 !== ''`），空标签时 `getTieredBillingSummary` 返回 null，摘要本身不渲染。因此该项在
模型详情入口 + helper 单测（`tier-labels.test.ts`）覆盖，未在日志入口伪造不可达状态。

## 六、门禁输出

| 命令 | 结果 |
|---|---|
| `cd web && bun run typecheck` | `TYPECHECK_EXIT=0` |
| `cd web && bun run i18n:check` | `I18N_EXIT=0`：`4036 literal t() keys + 467 static keys = 4197 referenced keys across 7 locales`，全语言齐备 |
| `bunx oxlint -c .oxlintrc.json src/features/usage-logs src/features/pricing` | `Found 0 warnings and 0 errors.`（74 文件） |
| `cd web && bun run format:check` | `FORMAT_EXIT=0`（1049 文件） |
| `bun test src/features/usage-logs/ src/features/pricing/` | **31 pass / 0 fail** |
| `cd web && bun test`（全量） | **427 pass / 0 fail**（65 文件） |

## 七、风险与未完成

- 详情弹窗的「命中阶梯」行现在**总是渲染**（值为 helper 结果）；此前空标签时整行省略。因为空标签在
  `tieredSummary` 存在的前提下不可达，实际渲染结果不变，仅代码路径更直白。
- 未做端到端浏览器截图（需实例 + 真实日志）；本轮为组件级渲染测试。
- 若将来新增展示入口，必须继续走 `getTierDisplayLabel`（或直接渲染 `TierLabel` 类组件）——建议在
  `tier-labels.ts` 顶部注释里保留这一约束（已有）。

## 八、建议下一步

1. 实例上用一条 `tiered_expr` 日志核对：列表「详情」列应显示「低谷 · ¥…/M」、弹窗显示「命中阶梯：低谷」。
2. 若后续还有新的档位展示位（例如导出 CSV 增加档位列），按本表规则先判定「展示 / 逻辑」再接入 helper。
