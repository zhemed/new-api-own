# 收尾 structuredClone 两条 lint 警告

## Goal

处理 `unicorn(prefer-structured-clone)` ×2（`channel-affinity/constants.ts:133`、
`channels/lib/advanced-custom.ts:343`），**在不改变行为的前提下**让 `bun run lint` 归零。
仅改这两个文件。

## 调查结论（决定依据）

两处都是 `JSON.parse(JSON.stringify(x))`，且**各自全仓只有一个调用点**（`grep -rn` 验证）：

| 函数 | 声明 | 唯一调用点 | 传入的数据 |
| --- | --- | --- | --- |
| `cloneTemplate<T>(template: T): T` | **泛型**、导出 | `channel-affinity/index.tsx:216` | `Object.values(RULE_TEMPLATES)` 的值 |
| `cloneAdvancedCustomConfig(config: AdvancedCustomConfig)` | 非泛型、导出 | `advanced-custom.ts:353`（`getAdvancedCustomTemplateConfig` 内） | `ADVANCED_CUSTOM_TEMPLATE_OPTIONS[*].config` |

**两个数据源都是模块级静态字面量，且已逐项核对为 JSON-safe**：

- `RULE_TEMPLATES`（constants.ts:88-117）：只有 `string` / `number`（`0`）/ `boolean` /
  字符串数组 / 纯对象；`param_override_template` 由 `buildPassHeadersTemplate` /
  `buildCodexPassHeadersTemplate` 生成，形如
  `{operations:[{mode:'pass_headers', value:[...strings], keep_origin:true}]}`。
  **没有**值为 `undefined` 的键、`Date`、`NaN`/`Infinity`、函数、`Map`/`Set`、循环引用。
- `ADVANCED_CUSTOM_TEMPLATE_OPTIONS[*].config`（advanced-custom.ts:284-338）：
  全部是字符串/数组/纯对象；`auth` 由 `apiKeyHeaderAuth()`/`geminiQueryAuth()` 生成，
  返回 `{type, name, value}` 三个字符串字段。同样无上述危险值。

⇒ **对"当前实际数据"两者等价**。

**但声明的契约无法保证任意输入 JSON-safe**：

- `cloneTemplate` 的 `T` 无约束（泛型导出），未来调用方可传 `Date`/`Map`/含 `undefined` 值键的对象；
- `AdvancedCustomConfig` 的字段全部 optional，在未启用 `exactOptionalPropertyTypes` 时
  `{incoming_path: undefined}` 合法 ⇒ 与 JSON 往返（丢键）结果不同。

## Requirements

**采用方案 (c)：保留原实现，就地 disable 并附"可验证理由"。** 理由：

1. **方案 (b) 被写范围卡死**：把 `cloneTemplate` 收窄为 `T extends JsonSafe` 需要
   `RuleTemplate` 可赋值给 `JsonSafe`，但 `AffinityRule.param_override_template`
   是 `Record<string, unknown> | null`（`channel-affinity/types.ts:38`），
   `unknown` 不可赋给 `JsonSafe`；修它要改 `types.ts`，**不在本次允许的两个文件内**。
   （调用点 `index.tsx` 亦不可动。）
2. **方案 (a) 只能证明"当前调用点"，不能证明"函数契约"**：两函数都是**导出**的，
   `cloneTemplate` 还是泛型；替换后对"含 `undefined` 值键 / `Date` / `NaN`"的输入会静默改变结果
   （JSON：丢键 / ISO 串 / `null`；`structuredClone`：保留 / `Date` / 原值）。按 Lead
   "不许强行替换"，不采用。
3. 就地 disable 是**零行为变更**且本仓库已有先例
   （`nav-group.tsx`、`mobile-drawer.tsx`、`html-content.tsx`、`risk-acknowledgement-dialog.tsx`）。

disable 注释必须写明：这是"JSON 往返语义即所需语义"、已核对的静态数据源、
以及"能安全替换的前置条件"。**不得**写成无信息量的 silence。

## Acceptance Criteria

- [ ] `bun run lint` = **0 warnings / 0 errors**。
- [ ] `bun run typecheck` exit 0；`bun test` ≥ **224 pass / 0 fail**；`format:check` exit 0。
- [ ] 只改 `channel-affinity/constants.ts` 与 `channels/lib/advanced-custom.ts`。
- [ ] 汇报写清选了 (c) 及为何不选 (a)/(b)，并附四条命令输出。

## Constraints

- 不 commit/push；不碰用户实例；不碰 `pricing/lib/**`（他人并行修改）。
