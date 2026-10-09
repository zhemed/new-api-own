# 收尾 structuredClone 两条 lint 警告 — 结果

## 结论

采纳 **方案 (c)：保留 `JSON.parse(JSON.stringify(x))`，就地 disable 并附可验证理由**。
`bun run lint` 达到 **0 warnings / 0 errors**；`typecheck`、`test`（224 pass / 0 fail）、`format:check` 全绿。
改动仅限授权的两个文件（+15 行，全部是注释 / disable 指令，**零可执行代码变更**）。

## 为什么不是 (a)/(b)

**先做了 Lead 要求的调用点排查**（`grep -rn` 全仓）：

| 函数 | 声明 | 唯一调用点 | 传入数据 |
| --- | --- | --- | --- |
| `cloneTemplate<T>(template: T): T` | **泛型**、导出 | `channel-affinity/index.tsx:216` | `Object.values(RULE_TEMPLATES)` 的值 |
| `cloneAdvancedCustomConfig(config)` | 非泛型、导出 | `advanced-custom.ts:353`（`getAdvancedCustomTemplateConfig` 内） | `ADVANCED_CUSTOM_TEMPLATE_OPTIONS[*].config` |

**两个数据源都已逐项核对为 JSON-safe**（这是"当前等价"的依据）：

- `RULE_TEMPLATES`（`constants.ts:88-117`）：只有 `string` / `number`(`0`) / `boolean` /
  字符串数组 / 纯对象；`param_override_template` 由
  `buildPassHeadersTemplate` / `buildCodexPassHeadersTemplate` 生成，形如
  `{operations:[{mode:'pass_headers', value:[...strings], keep_origin:true}]}`。
  **无**值为 `undefined` 的键、无 `Date`、无 `NaN`/`Infinity`、无函数、无 `Map`/`Set`、无循环引用。
- `ADVANCED_CUSTOM_TEMPLATE_OPTIONS[*].config`（`advanced-custom.ts:284-338`）：全部字符串/数组/纯对象；
  `auth` 来自 `apiKeyHeaderAuth()` / `geminiQueryAuth()`，返回 `{type, name, value}` 三个字符串。

**但 (a)/(b) 都不成立**：

- **(b) 被写范围卡死**：把 `cloneTemplate` 收窄成 `T extends JsonSafe`，需要 `RuleTemplate`
  可赋值给 `JsonSafe`；而 `AffinityRule.param_override_template` 是
  `Record<string, unknown> | null`（`channel-affinity/types.ts:38`），`unknown` 不可赋给 `JsonSafe`。
  要修就得改 `types.ts`，**不在本次允许的两个文件内**（调用点 `index.tsx` 同样不可动）。
- **(a) 只能证明"当前调用点"，证明不了"函数契约"**：两函数都是**导出**的，`cloneTemplate`
  还是泛型；替换后对「含 `undefined` 值键 / `Date` / `NaN`」的输入会**静默改变结果**
  （JSON：丢键 / 转 ISO 串 / 转 `null`；`structuredClone`：保留 / 保留 `Date` / 保留原值）。
  按 Lead「不许强行替换」，不做。

**方案 (c) 因此是唯一能"零行为变更地收掉警告"的选项**，且本仓库已有同款先例
（`nav-group.tsx`、`mobile-drawer.tsx`、`html-content.tsx`、`risk-acknowledgement-dialog.tsx`）。

## 实际改动

`web/src/features/system-settings/general/channel-affinity/constants.ts:132`
与 `web/src/features/channels/lib/advanced-custom.ts:340`：
各自在 `return JSON.parse(JSON.stringify(...))` 上方加了一段说明 + 一条
`// oxlint-disable-next-line unicorn/prefer-structured-clone -- <理由>`。
注释写明了三件事：①这里要的就是 JSON 往返语义；②已核对的当前数据源为何等价；
③要安全替换的前置条件（收窄 T / 处理 optional-undefined 键）。**不是无信息量的 silence。**

## 实测（四命令，全 exit 0）

| 命令 | 结果 |
| --- | --- |
| `bun run lint` | **Found 0 warnings and 0 errors** |
| `bun run typecheck` | 无输出（exit 0） |
| `bun test` | **224 pass / 0 fail**（Ran 224 tests across 42 files） |
| `bun run format:check` | 无问题（exit 0） |

改动范围确认：`git diff --stat` 仅上面两个文件，共 +15 行。

## 需 Lead 知悉：并行写入的瞬时红（非我引入）

工作期间另一位成员在改 `pricing/lib/**` 与 `pricing/components/**`，期间出现过：

- `typecheck`：`src/features/pricing/lib/mock-stats.ts(354,19): Cannot find name 'PROFILE_BY_NAME'`；
- `test`：`src/features/pricing/components/__tests__/sample-data-annotation.test.tsx`
  3 个用例失败（`API tab sample-data annotation` 系列）；
- 更早还出现过 `pricing/components/__tests__/sample-data-annotation.test.tsx`
  找不到 `../components/model-details-api`。

**对方落盘后全部自愈**，上表即最终快照。这几处均在 Lead 明确禁止我触碰的 `pricing/lib/**`
与 `pricing/components/**` 内，我全程未改；我的两个文件在任何时刻都未出现在错误列表中。
