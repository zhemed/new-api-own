# knip 收口 round5 — 达成 exit 0

## 最终状态（全部硬门）

| 门 | 结果 |
|---|---|
| `bun run knip` | **exit 0** ✓ |
| `bun run knip --include files` | **0 项** ✓ |
| `bunx tsgo -b --force` | **0 错误** ✓ |
| `bun run lint` | **exit 0（0 warnings / 0 errors）** ✓ |
| `bun test` | **247 pass / 0 fail** ✓ |
| `bun run format:check` | exit 0 ✓ |
| `bun run build` | exit 0 ✓ |

## 关键决策：没有用 `ignoreIssues`，改用 knip 官方的**符号级**机制

Lead 指定用 `ignoreIssues` 按"文件 + 符号"登记。**实测 knip 6.27.0 不支持**：
其 schema 为 `ignoreIssues: { <filePattern>: IssueType[] }`（只能填 `exports` / `types` 等**问题类型**），
`node_modules/knip/schema.json:57` 可查。我按"文件→符号数组"写入后 knip 直接报
`Invalid input (location: ignoreIssues.src/...ts.0)`（66 条全报）——即：
**用它就只能忽略整文件的全部导出**，恰恰违反你"必须精确到符号、以后新增死导出仍要报出来"的要求 ✗。

改用 knip **官方符号级机制**，两类处置：

1. **删除真正的死 re-export 说明符**：11 个（如 `model-details-api` 的 `ApiTabIcon`、`pricing/components/index.ts` 的 `ModelCard`）——
   纯 re-export，删掉即消失，**无需任何配置** ✓
2. **JSDoc `@public` 标记 53 处**（knip 官方支持的"刻意公开"标记）：
   标记写在**具体声明/语句正上方**，注释里写明理由，如
   `/** @public auth rotation contract (paired with auth-session.ts); deleting it requires updating callers. */`
   ——符号级 ✓、以后**新增**死导出仍会被报出 ✓、理由就在代码现场 ✓

**结果：为这 66 条新增的配置例外 = 0 条。**

## 累计统计（398 → 0）

| 阶段 | exports | types | files |
|---|---|---|---|
| round3 起点 | 301 | 97 | 0 |
| round3 结束 | 157 | 19 | 17 |
| round4 结束 | 53 | 13 | 0 |
| **round5 结束** | **0** | **0** | **0** |

- **删除文件 17 个**：13 个 `brand-icons/icon-*.tsx` + `mobile-drawer.tsx` / `nav-link-item.tsx` /
  `public-navigation.tsx` + `components/layout/constants.ts`（均逐个核对绝对路径、逐个 `rm`）。
- **删除符号 ≈ 332 + 重导出说明符 11 个**；**`@public` 标记 53 处**；（另 2 条为 i18n default 导出等）。

## 过程中的自身瑕疵（已全部修净）

`X as X` 无用重命名（import 重写器产生）、`consistent-type-imports`、空 `export {}` /
`export type {}`、`newline-after-import` → 已用 `bun run format` + `oxlint --fix` + 定点脚本清理，
最终 lint 0/0。
