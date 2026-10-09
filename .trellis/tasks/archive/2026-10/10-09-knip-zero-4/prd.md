# knip 清仓 round4：AST 精确删除 + 授权删文件

## Goal
用已装好的 `@babel/parser`（plugins typescript+jsx）按 AST 精确整段删除剩余
`Unused exports (157)` + `Unused exported types (19)`；并删除 Lead 授权的 16 个不可达文件。
目标：`bun run knip` exit 0 且 `knip --include files` 归 0。

## Requirements
- R1 规程不变：每批**一个文件** → `bunx tsgo -b --force` → **按错误签名**比对 →
  有新签名就**只** `git checkout -- <单个确切文件>` 并记入保留清单；
  **严禁** stash / clean / 目录级 / 全仓 checkout；每 20 批 `bun test`(≥247) + `lint`(0/0)。
- R2 用 `@babel/parser` 解析并整段删除声明节点（含 JSDoc/装饰器/导出修饰符）；
  被其它文件引用的符号一律不删（只去 `export`）。
- R3 删除授权文件：13 个 `src/assets/brand-icons/icon-*.tsx` + `mobile-drawer.tsx` +
  `nav-link-item.tsx` + `public-navigation.tsx`。**逐个核对绝对路径、逐个删**（禁止通配/递归）。
- R4 `src/components/layout/constants.ts`：先用**精确 import 路径**核查
  （`from './constants'` / `from '../constants'` 等），**确认无任何 import 才删**，否则保留并说明。
- R5 保留 3 个 `.oxlintrc.json` 中出现的名字（getDashboardSectionNavItems 等）。
- R6 删完立刻 `bunx tsgo -b --force` + `bun run build`；`bunx knip --include files` 必须 0 项。

## Acceptance Criteria
- [ ] AC1：`bun run knip` exit 0（或逐条保留原因）。
- [ ] AC2：`knip --include files` = 0。
- [ ] AC3：`tsgo -b --force` 0 错误；`lint` 0/0；`bun test` ≥247；`format:check` 干净；`build` exit 0。
- [ ] AC4：删除文件/符号总数 + 保留清单（逐条原因）。
- [ ] AC5：未 commit/push；回退未越出单文件边界。
