# 设计：knip 全量归零

## 1. 问题构成（实测基线）

| 类别 | 数量 | 主要分布 |
|---|---|---|
| Unused exports | 307 | `features/channels/lib` 45、`features/models/lib` 20、`components/layout/index.ts` 19、`features/pricing/lib` 15、`assets/brand-icons/*` 26、`src/lib/*` 若干… |
| Unused exported types | 97 | `components/layout/index.ts` 10、`components/data-table/index.ts` 9、各 `features/*/types.ts` 若干 |
| Unused dependencies | 6 | `@tanstack/react-virtual`、`auto-skeleton-react`、`next-themes`、`react-resizable-panels`、`recharts`、`tokenlens` |
| Unused devDependencies | 2 | `@xyflow/react`、`embla-carousel-react` |
| Duplicate exports | 1 | `src/hooks/use-dialog.ts` → `useDialogState|default` |
| Configuration hints | 2 | `ignoreDependencies` 里的 `tailwindcss`、`tw-animate-css` 已不再需要 |

## 2. 分类与处置策略

### 2.1 成套组件库（保留 + 配置表达）
`src/components/ui/**`、`src/components/ai-elements/**` 已经是 `ignore`，本轮**不新增**路径级 ignore。
它们的连带影响只体现在依赖报告上（被忽略的树对 knip 不可见）：→ 用 `ignoreDependencies` **逐条**表达，
每条注释写明"由哪个被忽略的树引用、为何保留"。

`assets/brand-icons/**` 与 `components/layout/index.ts` 等 barrel 属**半库半死码**：
- barrel（`index.ts`）里**指向真实实现**的 re-export：若该实现另有直接 import 路径，barrel 条目可删；
- **无任何消费者**的独立图标组件（如未被使用的品牌图标）：属"按需取用素材"，与 `ui/**` 同类，按**路径** ignore 并注明理由。

### 2.2 非库死代码（删除）
判定标准：`grep -rn` 全仓（含测试、动态 import、字符串引用、`.mjs`/`.json`）为空 **且** 不属上述底座。
删除方式分级：
- 该符号**在本文件内仍被使用** → 只去掉 `export` 关键字（保留实现，最小改动、零行为变化）；
- 该符号**在本文件内也不再被使用** → 删除整个声明。

### 2.3 依赖
逐条实查使用点：命中在 `ui/**` 或 `ai-elements/**` → 归入 2.1 的 `ignoreDependencies`；
命中在其它地方（knip 误报）→ 不 ignore，查明原因；确实全仓无引用 → 记录并上报（删依赖会牵动 lockfile 且禁止联网安装，需 Lead 决策）。

### 2.4 真修项
- 重复导出：`use-dialog.ts` 同时 `export function useDialogState` 与 `export default`，去掉重复的那个（保留具名导出，调用方按具名使用）。
- 配置提示：从 `ignoreDependencies` 移除 `tailwindcss` / `tw-animate-css`。

## 3. 风险与对策

| 风险 | 对策 |
|---|---|
| 批量删除导出误伤运行时 | 三重验证：`typecheck`（引用消失会立即报错）、`bun test`（224 项）、`bun run build`（打包期发现动态引用缺失） |
| knip 对**仅类型**导出误判 | 类型导出被删只会影响类型引用，`typecheck` 可捕获 |
| 触碰他人正在改的文件 | 排除清单两个文件在配置层做"仅忽略其在 knip 中的条目、不动其代码" |
| 大范围机械改动难以人工复核 | 改动前在 `/tmp` 沙箱跑一遍同款操作、逐项比对；正式改动后给出 diff 统计与分类清单 |

## 4. 边界

- 不改 `package.json` 依赖版本、不跑安装（禁止联网）。
- 不新增路径级 `ignore`（除既有两条）；所有 `ignoreDependencies` 均带理由。
