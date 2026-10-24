# 档位名显示：补齐全部展示入口（用量日志 / 详情弹窗 / 模型详情）

## Goal

用户截图：用量日志「详情」列仍显示 `offpeak · ¥1/¥4/M`，而定价面板已显示「低谷」。
上一轮（Session 115，v0.0.50）只把 `dynamic-pricing-breakdown.tsx` 一处改成展示层翻译，**漏了其它展示入口**。

## 既定约定（Lead 已查证）

1. **Session 115 的决定**：档位名是**表达式里的字面量**（`tier("peak", …)`），
   **只在展示层翻译**；表达式本身不改、`normalizeTierLabel` 相等比较与高亮判断一律使用**原始标签**。
2. **项目硬规则**（`web/AGENTS.md` 3.1）：所有面向用户的文案都必须走 i18n —— 因此**每一个**渲染档位名的地方都要用同一映射，
   不能只改一处。

按这两条，上一轮的实现**不完整**：全仓共 4 个展示入口，只改了 1 个。

## 展示入口盘点（Lead 已 grep 全仓）

| # | 入口 | 位置 | 状态 |
|---|---|---|---|
| 1 | 定价面板「分档价格表」（移动端卡片 + 表格） | `features/pricing/components/dynamic-pricing-breakdown.tsx:302/:385` | 已修 |
| 2 | 用量日志列表「详情」列 | `features/usage-logs/components/columns/common-logs-columns.tsx:170` | **待修** |
| 3 | 日志详情弹窗「档位」字段 | `features/usage-logs/components/dialogs/details-dialog.tsx:241-244` | **待修** |
| 4 | 模型详情价格表「档位」列 | `features/pricing/components/model-details.tsx:1001` | **待修** |

> 说明：`model-details.tsx:993` 用 `tier.label` 拼 row key、`format.ts:252` 用 `normalizeTierLabel` 做匹配 —— 这些属**逻辑**，必须继续用原始标签，不得改成翻译串。

## Requirements

- R1 复用既有 helper `getTierDisplayLabel(label, t)`（`features/pricing/lib/tier-labels.ts`），
  在 2/3/4 三处**仅替换展示值**；若跨 feature 引用需要，沿用既有先例（`usage-logs/lib/format.ts` 已在用 `features/pricing/lib` 的
  `normalizeTierLabel`）。
- R2 **逻辑不得被污染**：row key、`normalizeTierLabel` 比较、`matchedTierLabel` 高亮、`formatConditionSummary`、
  CSV/导出/复制文本（若有）中的标签处理需逐一确认——**修改前先 grep 出所有 `.label` 消费点并列进证据**，
  逐个判定「展示」还是「逻辑」。
- R3 未知标签仍原样显示、空标签回退 `t('Default')`（与既有 helper 行为一致，不得在这些入口另写一套回退）。
- R4 测试：
  - 三处入口各补/改一条渲染断言：zh 下 `peak`/`offpeak` 显示「高峰 / 低谷」；未知标签原样；空标签 `Default`；
  - 至少一条**反向测试**：日志列表或详情里的匹配/高亮仍按原始标签（例如 `matched_tier='offpeak'` 时高亮成立，
    而传入「低谷」不成立）。
- R5 门禁：`cd web && bun run typecheck && bun run i18n:check`、相关测试 `bun test`、改动文件 `bunx oxlint`、
  `bun run format:check`。若需新增 i18n 键 → 七语言齐备、只插不排、`i18n:sync`。
- R6 红线：**不发版**（用户明确不想频繁发版）、不提交不推送、不动 Go 文件、不动 `data/` 与实例。

## Acceptance Criteria

- [ ] AC1 4 个展示入口全部使用 `getTierDisplayLabel`，无遗漏（证据里给 grep 前后对照）。
- [ ] AC2 逻辑路径仍用原始标签（有反向测试证明）。
- [ ] AC3 未知标签原样、空标签 Default（三处均覆盖）。
- [ ] AC4 门禁全过，有输出。
- [ ] AC5 独立验证通过（非实现者自证）。

## Notes

- 这是「上一轮修复不完整」的补齐；根因不是没做映射，而是**只做了一个入口**。
- 若发现还有第 5 个入口（例如导出/复制摘要），一并列出并同批修。
