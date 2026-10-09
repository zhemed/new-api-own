# 清理前端 lint 警告（零行为变更）— 结果

## 结论

`bun run lint`：**21 warnings / 0 errors → 2 warnings / 0 errors**。
修掉 18 条 + 1 条（`react/no-danger`）以**带理由的就地 disable** 收敛；
**2 条主动跳过**（无法证明全输入等价，按"宁留 warning，不改行为"处理）。
`bun test` **221 pass / 0 fail**（任务下达时基线 213，未下降；增量来自并行成员新增的测试文件）。

## 已修（19 条，逐条语义等价）

| # | 文件 | 规则 | 等价依据 |
| --- | --- | --- | --- |
| 1–4 | `hooks/use-system-config.ts` | prefer-add-event-listener ×4 | `preloadImage` 里 `img.onload/onerror` → `addEventListener('load'/'error')`，cleanup 改成按同一函数引用 `removeEventListener`；`img` 是本函数新建的局部实例，不存在其它监听者 ⇒ 等价 |
| 5 | `components/turnstile.tsx` | prefer-add-event-listener ×1 | 新建 `<script>` 上的 `s.onload = () => render()` → `addEventListener('load', () => render())`；无 cleanup（原代码也没有置空）⇒ 等价 |
| 6 | `components/ai-elements/prompt-input.tsx` | prefer-add-event-listener ×1 | `reader.onerror = reject` → `addEventListener('error', reject)`；两种写法都以事件对象为唯一实参调用 `reject`，`FileReader` 为新实例 ⇒ 等价（相邻的 `onloadend` 未被规则命中，故未动） |
| 7 | `components/ai-elements/prompt-input.tsx` | prefer-add-event-listener ×1 | `speechRecognition.onerror` → `addEventListener('error', …)`。**注意**：`addEventListener` 会把事件类型放宽为 `Event`，故在回调内 `event as SpeechRecognitionErrorEvent` 取回 `error` 字段——纯类型层收窄，运行时对象与原来同一个 ⇒ 等价 |
| 8–9 | `features/dashboard/lib/stats.ts` | prefer-number-properties ×2 | `result = value / divisor`，两参均声明为 `number` ⇒ 实参静态即 number，`Number.isNaN/Number.isFinite` 与全局版等价（不会发生强制转型差异） |
| 10–11 | `system-settings/integrations/amount-options-visual-editor.tsx` | prefer-number-properties ×2 | ①`isNaN(Number(item))` 实参已是 number；②`Number.parseFloat(...)` 返回值是 number ⇒ 等价 |
| 12–13 | `system-settings/integrations/amount-discount-visual-editor.tsx` | prefer-number-properties ×2 | `item.amount`（`Number.parseInt`）与 `item.discountRate`（number 分支 / `Number.parseFloat`）**均为 number** ⇒ 等价 |
| 14 | `system-settings/utils/json-parser.ts` | prefer-string-slice ×1 | `substring(0, position)`，`position` 由 `/at position (\d+)/i` + `parseInt` 得来 ⇒ 恒 ≥0；非负时 `slice(0,p)` ≡ `substring(0,p)`；且 `getLineAndColumn` 是模块私有、仅一个调用点 ⇒ 等价 |
| 15 | `system-settings/integrations/utils.ts` | prefer-string-slice ×1 | 同上（`/\d+/` + `parseInt`）⇒ 等价 |
| 16 | `system-settings/models/utils.ts` | prefer-string-slice ×1 | 同上（`/\d+/` + `parseInt`）⇒ 等价 |
| 17 | `features/dashboard/lib/text.ts` | prefer-string-slice ×1 | `getPreviewText` 是**导出**函数、`maxLength` 由调用方给，可能为负；故写成 `slice(0, Math.max(0, maxLength))` —— 对**任意**数值都与 `substring(0, maxLength)` 完全一致（含负数钳到 0、超长钳到 length）⇒ 等价 |
| 18 | `components/data-table/toolbar/bulk-actions.tsx` | prefer-array-index-of ×1 | `findIndex(b => b === document.activeElement)` → `indexOf(document.activeElement)`：元素是 DOM 节点（不可能是 NaN），`indexOf` 严格相等 ≡ `===` ⇒ 等价（补了 `as HTMLButtonElement` 纯类型断言） |
| 19 | `components/layout/components/footer.tsx` | react/no-danger ×1 | **未改代码**。`sanitizedFooterHtml` 已由 DOMPurify 消毒，且带 fail-safe（`isSupported ? sanitize(html) : ''`），另有 3 个消毒测试；就地加 `// eslint-disable-next-line react/no-danger -- …` 并写明依据，与既有先例 `components/html-content.tsx:203` 同款 ⇒ 行为零变化 |

## 主动跳过（2 条，按 Lead "宁留 warning，不改行为"）

| 文件:行 | 规则 | 跳过原因 |
| --- | --- | --- |
| `system-settings/general/channel-affinity/constants.ts:133` | prefer-structured-clone | `export function cloneTemplate<T>(template: T): T { return JSON.parse(JSON.stringify(template)) }` —— **泛型 T**，等价性取决于调用方类型。`JSON.parse(JSON.stringify(x))` 与 `structuredClone(x)` 在以下情况**不等价**：值为 `undefined` 的自有键（JSON 丢弃 / clone 保留）、`Date`（JSON→ISO 字符串 / clone 保留 Date）、`NaN`/`Infinity`（JSON→`null` / clone 保留）、函数（JSON 丢弃 / clone 抛异常）、`Map`/`Set`（JSON→`{}` / clone 保留）、循环引用（JSON 抛异常 / clone 可用）。当前唯一调用点传的是 JSON 安全的模板，但函数签名对 T 无约束，无法证明"全输入等价" |
| `channels/lib/advanced-custom.ts:343` | prefer-structured-clone | 同上（`cloneAdvancedCustomConfig(config: AdvancedCustomConfig)`）。`AdvancedCustomConfig` 的字段全部 optional，在未开启 `exactOptionalPropertyTypes` 下允许显式写 `undefined`，此时两者对"键是否存在"的处理不同 ⇒ 不能证明等价 |

> 若要收掉这两条，需先收窄契约（例如把 `cloneTemplate` 改为 `structuredClone` 并把 T 约束为 JSON-safe，或确认所有调用方都不含上述值）——那属于**契约变更**，不在"纯机械"范围内，交 Lead 决定。

## 实测（并行成员落盘完成后的最终快照）

| 命令 | 前 | 后 |
| --- | --- | --- |
| `bun run lint` | 21 warnings / 0 errors | **2 warnings / 0 errors**（exit 0） |
| `bun run typecheck` | 0 | **0**（exit 0） |
| `bun test` | 213 pass / 0 fail | **224 pass / 0 fail**（exit 0） |
| `format:check` | — | **0**（exit 0） |

改动文件（12 个，全部在授权范围内，**未碰 `pricing/components/**`**）：
`hooks/use-system-config.ts`、`components/turnstile.tsx`、`components/ai-elements/prompt-input.tsx`、
`components/data-table/toolbar/bulk-actions.tsx`、`components/layout/components/footer.tsx`、
`features/dashboard/lib/stats.ts`、`features/dashboard/lib/text.ts`、
`features/system-settings/utils/json-parser.ts`、`features/system-settings/integrations/utils.ts`、
`features/system-settings/models/utils.ts`、
`features/system-settings/integrations/amount-options-visual-editor.tsx`、
`features/system-settings/integrations/amount-discount-visual-editor.tsx`。

## 并行写入干扰（曾出现瞬时红，已自愈，非我引入）

工作期间 `review-frontend` 在并发写 `pricing/components/**` 与 dashboard 测试，仓库级命令出现过**中间态红**：

- 一度 `typecheck` 报 `src/features/dashboard/__tests__/throughput-kpi-color.test.tsx` 的
  `Cannot find name 'seedQueue'` / 未使用的 `render`、`seedSummary`；
- 一度 `typecheck` 报 `pricing/components/__tests__/sample-data-annotation.test.tsx`
  找不到 `../components/model-details-api`（对方重构中途）；
- 一度 `test` 出现 `221 pass / 1 fail / 1 error`；`lint` 一度显示 4 warnings。

**对方落盘完成后全部恢复**：lint 2/0、test 224 pass / 0 fail、typecheck 0、format:check 0。
**我的 12 个文件在上述任何时刻都未出现在错误/警告列表中**；`pricing/components/**` 按红线全程未触碰。
