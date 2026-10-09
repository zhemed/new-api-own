# JSON 克隆语义显式化：移除两处 lint 抑制

## Goal

把上一轮留在 `cloneTemplate` / `cloneAdvancedCustomConfig` 上的两条
`oxlint-disable unicorn/prefer-structured-clone` **真正移除**（不是换个地方抑制），
做法是让「JSON 往返」这一语义**显式化**为一个带契约文档的公共 helper。

## 方案选择（采用**方案 2：显式递归实现，零抑制**）

Lead 给了两条路，选 **方案 2**，理由：

1. **目标就是"移除抑制"**：方案 1（helper 内包 `JSON.parse(JSON.stringify(x))` + 唯一一处 disable）
   只是把抑制集中，仍是抑制；方案 2 让 lint 抑制**彻底归零**。
2. **契约可被单测钉死**：显式实现把隐含在 `JSON.stringify` 里的语义写成代码 + JSDoc + 边界用例，
   以后任何人改动都会撞到测试，比"一行黑盒往返"更可维护。
3. **风险已被工程手段覆盖**：显式实现的主要风险是"与 `JSON.stringify` 语义有偏差"，
   对策是①严格按规范算法实现（`toJSON` → 装箱原语 → 类型分支 → 数组/对象）；
   ②对每个边界写**字面量期望值**的用例（不照抄生产逻辑计算期望）。
4. 副作用收益：省掉一次「序列化成字符串再解析回来」的往返。

## Requirements

### R1 新增 helper（`web/src/lib/json-clone.ts`）

`cloneJsonValue<T>(value: T): T`，JSDoc 必须写死以下契约（与 `JSON.parse(JSON.stringify(x))` 一致）：

| 输入 | 结果 |
| --- | --- |
| 值为 `undefined` 的**自有键** | 键被丢弃 |
| `undefined` 的**数组元素**（含稀疏数组空位） | `null` |
| `Date` | ISO 字符串（经 `toJSON`） |
| `NaN` / `Infinity` / `-Infinity` | `null` |
| `-0` | `0` |
| 函数 / `Symbol` 作为键值 | 键被丢弃；作为数组元素 → `null` |
| `BigInt` | **抛 `TypeError`**（与 `JSON.stringify` 一致） |
| `Map` / `Set` / 类实例 | 按**可枚举自有字符串键**展开（`Map`/`Set` 展开为 `{}`） |
| `null` / `string` / `boolean` / 有限 `number` | 原样 |
| 循环引用 | **抛 `TypeError`**（旧实现亦抛 TypeError，行为不劣化） |
| 其它对象 | 递归克隆；`toJSON()` 若存在则先调用 |
| 非枚举属性 / `Symbol` 键 | 忽略 |
| 原输入 | **不被修改**（纯函数） |

### R2 两个调用点改用 helper，且**不得保留任何 disable**

- `channel-affinity/constants.ts` 的 `cloneTemplate` → 调 `cloneJsonValue`
- `channels/lib/advanced-custom.ts` 的 `cloneAdvancedCustomConfig` → 调 `cloneJsonValue`

### R3 单测（`web/src/lib/__tests__/json-clone.test.ts`）

覆盖契约表每一行：`undefined` 键 / 数组元素 / 稀疏数组、`Date`、`NaN`、`Infinity`、
`-0`、函数值、`Symbol` 值、`BigInt` 抛错、`Map`/`Set`、类实例、循环引用抛错、
嵌套结构、非枚举与 Symbol 键忽略、原对象不被修改、`null` 透传。
期望值一律写**字面量**（不得调用生产逻辑反推期望）。

## Acceptance Criteria

- [ ] `web/src/lib/json-clone.ts` 存在，JSDoc 含完整契约表。
- [ ] 两个调用点均无 `oxlint-disable`，全部改用 `cloneJsonValue`。
- [ ] `bun run lint` = **0 warnings / 0 errors**，且全仓不再出现 `prefer-structured-clone` 抑制。
- [ ] `bun run typecheck` exit 0；`bun test` ≥ **224 pass / 0 fail**；`format:check` exit 0。
- [ ] 只改：上述两个文件 + 新建 helper 及其 `__tests__/`。

## Constraints

- 不 commit/push、不碰实例、**不碰 `web/src/features/pricing/**`**（他人并行修改）。
