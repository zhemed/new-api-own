# JSON 克隆语义显式化：移除两处 lint 抑制 — 结果

## 结论

采用**方案 2（显式递归实现，零抑制）**。新建 `web/src/lib/json-clone.ts` 的
`cloneJsonValue<T>()`，两个调用点改用该 helper，**两条 `oxlint-disable` 已彻底删除**，
全仓不再存在 `prefer-structured-clone` 抑制，也不再有 `JSON.parse(JSON.stringify(...))`。
新增 23 个边界单测全绿；我的 4 个文件 lint / format 全净。

## 为什么选方案 2（而非"把抑制集中到 helper"）

1. 目标就是"移除抑制"，方案 1 只是把抑制换个位置；
2. 显式实现把隐含在 `JSON.stringify` 里的语义写成**代码 + JSDoc + 边界用例**，可被测试钉死；
3. 主要风险是"与 `JSON.stringify` 有偏差"，用①严格按规范算法实现 + ②**开发期差分校验**覆盖
   （见下）。副作用收益：省掉一次字符串往返。

## 交付物

### 1) 新建 `web/src/lib/json-clone.ts`

`cloneJsonValue<T>(value: T): T`，JSDoc 写死契约：

```
Contract (identical to `JSON.parse(JSON.stringify(value))`):

| Input                                        | Result                                              |
| -------------------------------------------- | --------------------------------------------------- |
| own property whose value is `undefined`      | the property is dropped                             |
| `undefined` array element / sparse array hole| `null`                                              |
| `Date`                                       | ISO string (via `toJSON`)                           |
| `NaN` / `Infinity` / `-Infinity`             | `null`                                              |
| `-0`                                         | `0`                                                 |
| function / `Symbol` as a property value      | the property is dropped                             |
| function / `Symbol` as an array element      | `null`                                              |
| `BigInt`                                     | throws `TypeError`                                  |
| `Map` / `Set` / class instance               | expanded by own enumerable string keys (`{}` for `Map`/`Set`) |
| `null` / `string` / `boolean` / finite number| copied as-is                                        |
| nested object / array                        | cloned recursively; `toJSON()` is called when present |
| non-enumerable property / `Symbol` key       | ignored                                             |
| circular reference                           | throws `TypeError`                                  |
| the input value                              | never mutated (pure)                                |
```

**唯一的刻意偏差**（已在 JSDoc 写明）：根值本身是 `undefined`/函数/`Symbol` 时抛 `TypeError`，
而旧写法会因 `JSON.parse(undefined)` 抛 `SyntaxError`——两者都抛，只是这里给出了原因。

实现按 `JSON.stringify` 的规范算法：`toJSON` → 解箱（`new Number/String/Boolean`）→
类型分支（string/boolean/finite number/non-finite→null/bigint→throw/undefined·function·symbol→OMIT）
→ 数组按索引（空洞→null）→对象按 `Object.keys`（丢弃 OMIT 值）→ 用祖先栈检测循环引用。

### 2) 两处调用点（**disable 已全部删除**）

```diff
 // channel-affinity/constants.ts
 export function cloneTemplate<T>(template: T): T {
-  // ...旧的长说明...
-  // oxlint-disable-next-line unicorn/prefer-structured-clone -- ...
-  return JSON.parse(JSON.stringify(template))
+  // The JSON view of the template — see `cloneJsonValue` for the exact contract.
+  return cloneJsonValue(template)
 }

 // channels/lib/advanced-custom.ts
 export function cloneAdvancedCustomConfig(
   config: AdvancedCustomConfig
 ): AdvancedCustomConfig {
-  // ...旧的长说明...
-  // oxlint-disable-next-line unicorn/prefer-structured-clone -- ...
-  return JSON.parse(JSON.stringify(config)) as AdvancedCustomConfig
+  // `AdvancedCustomConfig` is a plain JSON shape, so cloning it means taking its
+  // JSON view — see `cloneJsonValue` for the exact contract.
+  return cloneJsonValue(config)
 }
```

两处均补 `import { cloneJsonValue } from '@/lib/json-clone'`。**未保留任何 disable**，
连 `as AdvancedCustomConfig` 断言也不再需要（泛型返回值即为 `T`）。

### 3) 单测 `web/src/lib/__tests__/json-clone.test.ts`（**23 pass / 0 fail**）

覆盖契约表每一行（期望值全部为**字面量**，不用生产逻辑反推）：
`undefined` 键 / `undefined` 数组元素 / 稀疏数组空洞、`NaN`·`Infinity`·`-Infinity`、
`-0`、函数值（对象丢键 / 数组→null）、`Symbol` 值、Symbol 键与非枚举属性忽略、
`BigInt` 抛错、根值不可表示抛错、`Date`→ISO、自定义 `toJSON`、无效 `Date`→null、
`Map`/`Set`→`{}`、类实例按自有可枚举属性展开、`null`/原语透传、深层嵌套、
克隆与原对象解耦、共享引用（非循环）**复制而非报错**、循环引用（含数组内嵌套）抛错、**原对象不被修改**。

### 4) 开发期差分校验（一次性，脚本已删）

写了一个临时探针，把 `cloneJsonValue(x)` 与 `JSON.parse(JSON.stringify(x))` 对 **34 个刁钻输入**
逐一比对（含装箱原语 `new Number/String/Boolean`、getter 属性、数字键顺序、`toJSON`、
无效 `Date`、稀疏数组、共享引用、`BigInt` 抛错）：

```
DIVERGENCES: 0 / 34
```

（该探针为一次性验证，已删除，不进入代码库；提交的测试用例全部使用字面量期望值。）

## ⚠️ 并行写入把本任务的文件**回滚过两次**（Lead 必须知悉）

工作期间另一位成员在跑大规模重构（`web/` 下 151→152 个文件被修改）。该批量操作**把我的文件
回滚到旧快照**，已发现并修复两次：

1. `src/lib/json-clone.ts` 的 `isBoxedPrimitive` 类型谓词（我已内联删除）**被还原**，
   导致 `typescript(no-wrapper-object-types)` 3 个 error 复现 —— 已重新内联修复；
2. `src/lib/__tests__/json-clone.test.ts` 的 `new Array(3)`（我已改成 `sparse[0]=…; sparse[2]=…`）
   **被还原**，导致 `unicorn(no-new-array)` 复现 —— 已重新修复。

**建议**：等对方停止写入后，请再核验一次这 4 个文件的最终内容（关键标记：
`grep -c isBoxedPrimitive src/lib/json-clone.ts` 应为 0、`grep -c 'new Array' src/lib/__tests__/json-clone.test.ts` 应为 0）。

## 四条命令

**我的范围（4 个文件）全部通过**：

| 检查 | 结果 |
| --- | --- |
| `bun run lint` 命中我的文件 | **0 条**（grep 全仓 issue 列表为空） |
| `bun test src/lib/__tests__/json-clone.test.ts` | **23 pass / 0 fail** |
| `oxfmt --check`（我的 4 个文件） | All matched files use the correct format |
| 全仓 `grep prefer-structured-clone` | 仅剩 helper JSDoc 的文字说明，**无 disable 指令** |
| 全仓 `grep 'JSON.parse(JSON.stringify'` | 仅剩 helper JSDoc 的文档文字，**无可执行代码** |

**仓库级四条命令此刻不可测**：另一位成员正在做大规模重构（`git status` 显示 `web/` 下
**151→152 个文件被修改**，含 `assets/brand-icons/*`、`mobile-drawer.tsx`、`lib/format.ts`、
`lib/time.ts`、`stores/system-config-store.ts` 等，均非我的文件）。我持续轮询约 15 分钟，
观测到仓库级结果在**剧烈波动**：

```
lint errors:      4 → 1 → 0 → (随后 106 warnings / 0 errors)
typecheck errors: 64 → 60 → 0 → 159
format issues:    31 → 1 → 31
```

因此仓库级四条命令的最终值应由 Lead 在对方停止写入后再测一次。**我的 4 个文件在上述任何时刻
都未出现在错误/警告列表中**；期间我未触碰任何他人的文件，也未碰 `web/src/features/pricing/**`。
