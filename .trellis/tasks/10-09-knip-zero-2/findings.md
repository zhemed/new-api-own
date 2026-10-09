# knip 收口（round2）— 交付

**结论**：按 Lead 指令**停止批量删除**，改为"配置级 + 零风险真修"收口。
**knip 未归零**，但把 6 类问题中的 **4 类清零**；剩 `Unused exports (303)` / `Unused exported types (97)`，
原因见第三节。四条硬门全绿，改动面收敛到 **2 个文件**。

## 一、我保留的改动（逐文件一句话）

| 文件 | 改动 | 为什么零风险 |
|---|---|---|
| `web/knip.config.ts` | ①删掉失效的 `tailwindcss` / `tw-animate-css` 两条 `ignoreDependencies`；②新增 5 条 `ignoreDependencies` 并逐条注释引用点 | 纯配置，无运行时代码 |
| `web/src/hooks/use-dialog.ts` | 具名 `export function useDialogState` → 去掉 `export` 关键字（**仅此一处**） | 该具名形式**全仓无人 import**（5 处消费者都用 `import useDialogState from ...` 默认导入）；默认导出 `export default useDialogState` 保留，行为不变 |

## 二、我恢复掉的（146 个文件）

本轮早先的 `knip --fix` + 分批删除共改动 **146 个 `web/src/**` 文件**。
按 Lead 要求**逐文件** `git status --porcelain` 取列表 + `git checkout -- <单个文件>`
（**未用** `git stash` / `git clean` / 目录级或全仓 checkout）全部恢复。

**明确未触碰**（他人的工作）：`src/lib/json-clone.ts`、`src/lib/__tests__/json-clone.test.ts`、
`src/features/channels/lib/advanced-custom.ts`、`src/features/system-settings/general/channel-affinity/constants.ts`；
**也未动** Lead 的 `package.json` / `bun.lock`（那 3 个真无用依赖的移除）。

最终 `git status --porcelain -- web/`：
```
 M web/bun.lock                 ← Lead
 M web/knip.config.ts           ← 我
 M web/package.json             ← Lead
 M web/src/features/channels/lib/advanced-custom.ts                        ← 他人
 M web/src/features/system-settings/general/channel-affinity/constants.ts  ← 他人
 M web/src/hooks/use-dialog.ts  ← 我
 ?? web/src/lib/json-clone.ts / __tests__/json-clone.test.ts               ← 他人新增
```

## 三、knip 前后各节计数

| 节 | 前 | 后 |
|---|---|---|
| Unused exports | 307 | **303** |
| Unused exported types | 97 | 97 |
| Unused dependencies | 6 | **0** |
| Unused devDependencies | 2 | **0** |
| Unused files | 0 | 0 |
| Duplicate exports | 1 | **0** |
| Configuration hints | 2 | **0** |

**清零的四类**：dependencies / devDependencies（5 条"只被已忽略组件库树引用"逐条登记）、
Duplicate exports（真修）、Configuration hints（真修）。
**303 条的构成**：`features/**` 236、`components/**` 63、`src/lib` 55、`assets` 26、`hooks` 7、其余 8
（`components/**` 里含 `ui/**`、`ai-elements/**` 之外的普通组件；两棵底座树本身已被 `ignore`）。

## 四、为什么没归零（如实）

1. **`Unused exports/types` 共 400 条**：knip 的修法是"去掉 `export` 关键字"，
   而其中约 140 条去掉后**在本文件内也没人用**，必须**整段删除声明**才能过 `noUnusedLocals`。
   这正是上一轮与这一轮都卡住的地方：**跨 152 个文件的机械删除，收益与风险不成比例**
   （语义误删、并发写入者被回滚、review 不可行）→ 按你的判断**已停手**。
   建议：这批留给**专门的清理任务**，或由 knip 上游/TS codemod 处理，而不是在并发开发期做。
2. **`advanced-custom.ts` 已无残留** ✓：knip 输出里 `advanced-custom` **0 条命中**
   （该文件已由他人写成 `cloneJsonValue(...)`，原先那 3 条未用导出随其改动消失）。
   它现在的 `M` 状态是**他人**的改动，我未触碰。

## 五、收工判定（四条硬门）

| 门 | 结果 |
|---|---|
| `bun run typecheck`（`tsgo -b --force`，非增量） | **0 错误** ✓ |
| `bun run lint` | **0 warnings / 0 errors** ✓ |
| `bun test` | **247 pass / 0 fail**（43 files）✓ |
| `bun run format:check` | exit 0（干净）✓ |
| `bun run i18n:check` | 所有键七语言齐备 ✓ |
| `bun run knip` | exit 1（剩 303+97）—— 如实报告，未为绿而 ignore 全仓 |

## 六、踩过的坑（建议进 spec）

1. **`tsgo -b` 是增量构建**：逐批跑 typecheck 时，**只改动的文件**会被重新报告，
   未改动文件的错误**不会重现** —— 我因此得到过"1 个错误"的假象。
   校验必须用 `bunx tsgo -b --force`（或每次清 tsbuildinfo）。
2. **校验要按"错误签名"比较**（`(文件, 代码, 符号)`），不要比总数——删声明会让行号整体位移。
3. **别在多人并发写的工作区做跨文件批量回滚**：这轮我把他人两次修好的文件回滚了两次。
   批量操作前先确认"谁在这片区域写"。
