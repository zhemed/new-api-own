# 动态计费档位名显示英文（peak / offpeak）

## Goal

用户截图（2026-10-24）：面板「动态计费 → 分档价格表」中，列头是中文（输入 / 输出 / 缓存读取），
但**档位名**显示为英文 `peak` / `offpeak`。期望中文（以及其它语言）显示。

## 根因（Lead 已定位）

- 档位名是**计费表达式里的字面量标签**：`tier("peak", p*2 + c*8 + cr*0.04)` / `tier("offpeak", ...)`，
  代码默认值见 `setting/billing_setting/tiered_billing.go:35`（`defaultDeepSeekPeakOffPeakExpr`）。
- 前端「分档价格表」组件 `web/src/features/pricing/components/dynamic-pricing-breakdown.tsx`
  **原样渲染** `tier.label`（:301 移动端卡片、:384 表格列），全仓**没有任何**档位名翻译映射
  （`grep -rn "tierLabel|translateTier" web/src` 无命中）。
- 所以英文来自**表达式里的标签本身**，不是 i18n 漏译某个键。

## Requirements

### R1 显示层映射（不改表达式、不改计费数据）

- 新增档位名 → i18n 键 的映射，覆盖项目自带模板里出现的标签：
  `peak`、`offpeak`、`base`、`standard`、`short`、`mid`、`long`、
  `short_output`、`long_output`、`mid_context`、`long_context`、`discount`
  （来源：`setting/billing_setting/tiered_billing.go` 默认表达式 + `web/src/features/system-settings/models/tiered-pricing-editor.tsx`
  的模板清单）。
- 中文建议译法：高峰 / 低谷 / 基础 / 标准 / 短 / 中 / 长 / 短输出 / 长输出 / 中上下文 / 长上下文 / 优惠；
  其余语言按语义翻译（不要留空、不要机翻式直译成 "Peak" 以外的怪词）。
- **未知标签必须回退为原样显示**（用户可自定义任意标签，不得吞掉或显示成空）；空标签回退为既有 `t('Default')`。
- 只作用于**展示**：`normalizeTierLabel` 的相等比较、`matchedTierLabel` 高亮判断、条件摘要
  （`formatConditionSummary`）等逻辑**必须继续使用原始标签**，不得改成翻译后的字符串。
- 落点建议：`web/src/features/pricing/lib/tier-labels.ts`（新文件：映射表 + `getTierDisplayLabel(label, t)`），
  在 `dynamic-pricing-breakdown.tsx` 的两处渲染点使用。

### R2 i18n

- 新增的键必须补齐 `web/src/i18n/locales/` 下 **en / zh / zh-TW / fr / ru / ja / vi 七个**文件；
  英文源串作为 key。**只插入新键、不重排既有键**；改完必须跑 `bun run i18n:sync`（恢复被 JSON 往返解码的混淆键）；
  `bun run i18n:check` 必须通过。
- 若已有可复用的既有键（例如 `Peak`），优先复用而不是新增，避免重复键。

### R3 测试

- `web/src/features/pricing/components/__tests__/` 下补测试（文件按职责命名，如 `tier-labels.test.ts`）：
  ① 已知标签在 zh 下显示中文、en 下显示英文；
  ② 未知标签（如 `my_custom_tier`）**原样显示**；
  ③ 空标签显示 `Default`；
  ④ 高亮/匹配逻辑仍按原始标签（可用一个含 `peak` 与自定义标签的表达式断言只有匹配项被高亮）。

### R4 门禁

- `cd web && bun run typecheck && bun run i18n:check`，相关测试 `bun test <路径>`，改动文件 `bunx oxlint`。
- 渲染安全自查（无成对美元符号、围栏配平）——本任务主要是代码与 locale。

## Acceptance Criteria

- [ ] AC1 已知档位名在中文界面显示中文；未知标签原样；空标签 `Default`。
- [ ] AC2 匹配/高亮/条件摘要逻辑仍基于原始标签（有测试证明）。
- [ ] AC3 七语言键齐备、无重排、`i18n:sync` 后混淆键完好、`i18n:check` 通过。
- [ ] AC4 门禁全过，有输出。
- [ ] AC5 独立验证通过（非实现者自证）。

## Notes

- **不改** `setting/billing_setting/tiered_billing.go` 的默认表达式（改标签等于改计费配置文本，
  会让已上线实例与日志里的 tier 名对不上；显示层翻译才是正确位置）。
- 本轮**不发版**（用户明确：目前没必要那么频繁发版）。
