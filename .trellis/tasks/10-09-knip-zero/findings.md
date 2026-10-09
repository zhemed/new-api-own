# knip 归零 — 未能完成，已完整回滚（如实报告）

**结论**：本轮**没有交付** knip exit 0。所有改动**已全部回滚**，工作区回到起点
（typecheck 0 错误、`bun test` 247 pass / 0 fail、knip 基线仍是 307/97/6/2/1/2、
`knip.config.ts` 未改、两个排除文件未触碰）。以下是已完成的分析与卡点。

## 一、分类结果（这部分是可用产出）

### 依赖（8 条）——逐条实查后的真相
| 依赖 | 实际被谁用 | 归类 |
|---|---|---|
| `recharts` | `src/components/ui/chart.tsx` | **仅被已忽略树使用** |
| `react-resizable-panels` | `src/components/ui/resizable.tsx` | 同上 |
| `embla-carousel-react` (dev) | `src/components/ui/carousel.tsx` | 同上 |
| `tokenlens` | `src/components/ai-elements/context.tsx` | 同上 |
| `@xyflow/react` (dev) | `src/components/ai-elements/*.tsx` | 同上 |
| **`@tanstack/react-virtual`** | **全仓 0 引用**（含被忽略树） | **真未使用** |
| **`auto-skeleton-react`** | **全仓 0 引用**；只有 `src/styles/index.css` 里一段注释/选择器提到它的类名（`.auto-skeleton-fade`），使用它的组件已不存在 | **真未使用** |
| **`next-themes`** | **全仓 0 引用**；主题由自研 `@/context/theme-provider` 提供 | **真未使用** |

→ 前 5 条用 `ignoreDependencies` + 逐条理由是对的；**后 3 条属真死依赖**，正确处理是从
`package.json` 移除，但那会让 `bun.lock` 失同步且需要联网安装（本轮禁止）→ **需你定夺**。

### 其它三类
- `Configuration hints (2)`：`ignoreDependencies` 里的 `tailwindcss` / `tw-animate-css` 已失效 → 直接删掉该配置项即可（真修，无风险）。
- `Duplicate exports (1)`：`src/hooks/use-dialog.ts` 同时具名 + 默认导出 `useDialogState`；
  实测 knip 自己的 `--fix` **能自动修掉它**。
- `Unused exports/types (307+97)`：集中在 `features/channels/lib`(45)、`features/models/lib`(20)、
  各 `index.ts` barrel、`src/lib/*`；`ui/**` 与 `ai-elements/**` 本就在 `ignore` 里，不在其中。

## 二、卡点（为什么没做到 0）

1. **`knip --fix --fix-type exports,types` 确实能把 404 → 0**（沙箱实测：两个 section 直接消失）。
   它的做法是**去掉 `export` 关键字**。
2. **但随之产生 ~142 个 `noUnusedLocals` 报错**（TS6133 / TS6196）：这些符号去掉 `export` 后
   **在本文件内也没人用**，必须**整段删除声明**才能过 `tsgo`。**knip 不负责这一步**。
3. **我没能用自写脚本安全完成这批删除**。四轮尝试、逐文件验证，结果：
   - 前几轮把 10 个文件改成**语法错误**；
   - 后一轮更糟：我的"回滚"只检查**语法类**错误（`TS1xxx`），**漏掉了语义类**
     （`TS2305: Module has no exported member`）——即删掉了仍被别的文件 import 的导出。
   - 判定该路线不可靠，**立即整体回滚**，未留残留。
   → 正确做法是用 **TypeScript 编译器 API（AST 精确范围）** 或 TS codemod 做这件事；
     本仓库用 `tsgo`（Go 版），`node_modules/typescript` **不存在**，无法在禁止联网的前提下拿到。
4. **另有一条硬阻塞**：`Unused exports` 里剩下 3 条
   （`ADVANCED_CUSTOM_MODEL_REGEX_PREFIX`、`cloneAdvancedCustomConfig`、`getAdvancedCustomIncomingPathOptions`）
   全在你**禁止我触碰**的 `src/features/channels/lib/advanced-custom.ts` 里。
   即使前面都做完，**不碰该文件 + 不加例外配置，knip 不可能 exit 0**。

## 三、回滚验证（当前状态）

```
$ git status --porcelain -- web/
 M web/src/features/channels/lib/advanced-custom.ts          ← 另一位成员
 M web/src/features/system-settings/general/channel-affinity/constants.ts  ← 另一位成员
 ?? web/src/lib/json-clone.ts / __tests__/json-clone.test.ts ← 另一位成员新增
$ bun run typecheck   → 0 错误
$ bun test            → 247 pass / 0 fail
$ bun run knip        → Unused exports (307) / exported types (97) / deps (6) / devDeps (2)
                        / Duplicate exports (1) / Configuration hints (2)   ← 与初始基线一致
$ git diff web/knip.config.ts → 未改动
```
（`web/src/lib/json-clone.ts` 等未跟踪文件是另一位成员的，我未触碰；我的回滚保留了它们。）

## 四、建议（需你定夺）

- **A（推荐）**：等 `advanced-custom.ts` 的改动落地后授权我触碰那两个文件，
  并**在同一轮**引入 `typescript` 依赖（或允许联网安装）来做 AST 级精确删除 —— 一次做完 404 条。
- **B**：先做**无风险部分**——删 `tailwindcss`/`tw-animate-css` 两条失效配置、
  修 `use-dialog.ts` 重复导出、为 5 条"仅被忽略树引用"的依赖写 `ignoreDependencies` + 理由。
  knip 仍不归零，但基线变小、每条例外可追溯。
- **C**：若不允许引入 `typescript`，则接受"exports/types 两类无法在不削弱检查的前提下归零"，
  按你第 3 条的要求**不硬凑**，把剩余条目挂到后续任务。

## 五、学到的一条（建议进 spec）

`knip --fix` 只删 `export` 关键字，**不会删声明**；与 `noUnusedLocals` 组合时会留下
"去掉 export 后连文件内也没人用"的残骸。批量清理必须用 **AST 级**工具，
且**验证必须覆盖语义类错误（TS2xxx）而不只是语法类（TS1xxx）** —— 这一条我踩过。
