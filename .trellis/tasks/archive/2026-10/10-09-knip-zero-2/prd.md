# knip 归零 round2：分批删除 + typecheck 语义复核

## Goal

在 `10-09-knip-zero`（已完整回滚）基础上重做。**关键纠正**：`bun run typecheck` 本身就能抓
`TS2305 / TS6133 / TS6196` 等语义错误，它就是安全网——**不需要引入 `typescript` 依赖、不需要联网**。
上次失败的原因是**我的校验脚本只过滤 `TS1xxx`（语法类）**，不是"必须用 AST"。

## Requirements

### R1 零风险真修（并各自量 knip 掉了多少条）
- 删掉失效的 `tailwindcss` / `tw-animate-css` 两条 `ignoreDependencies` 配置。
- `use-dialog.ts` 重复导出：用 `bunx knip --fix --fix-type exports` 在工作区内修（允许）。
- 为 5 条依赖写 `ignoreDependencies`：`recharts` / `react-resizable-panels` /
  `embla-carousel-react` / `tokenlens` / `@xyflow/react` —— **逐条列名 + 注释写明"只被已忽略的组件库树引用"**。
- **试**把 `ui/**` / `ai-elements/**` 改为**文件级忽略**（`ignoreFiles` 或 `ignoreIssues` 形态），
  对比哪种更贴合"不可达产品底座"的语义，并在 config 注释写明为什么；做完量一次总量。
  **这不算削弱检查**。

### R2 剩余 exports/types：分批删除 + 真实 typecheck 复核（核心）
- 按**文件**分批（一次一个文件，或 ≤10 个符号）。
- 每批：删声明 → **`bun run typecheck` 全量跑（不过滤错误类型，任何新错误都算失败）** → 失败则**整批回滚**。
- 每 N 批跑一次 `bun test`（基线 **247 pass / 0 fail**，不得下降）。
- **只删"去掉 export 后文件内也无人使用"的声明**；任何被其它文件引用的一律保留。

### R3 `advanced-custom.ts` 已获授权
只删**真正未使用**的导出（`ADVANCED_CUSTOM_MODEL_REGEX_PREFIX`、`cloneAdvancedCustomConfig`、
`getAdvancedCustomIncomingPathOptions`）。**绝对不要动另一位成员的 `@/lib/json-clone` import 与调用**。
动手前先 `git diff -- <该文件>` 看清现有改动。

### R4 3 条真无引用的依赖（`@tanstack/react-virtual` / `auto-skeleton-react` / `next-themes`）
**先不要移出 package.json**（牵动 `bun.lock`、需联网，等 Lead 问过用户后再定）。

### R5 验证与回报
- `bun run knip` 尽量归零；`bun run knip --include files` 保持 0。
- 六条命令 + `format:check`；`bun test` ≥ 247 pass。
- 回报：knip 前后各节计数、删了什么、config 每条例外的理由、typecheck/test 结果；
  **若最终仍不为 0，如实说明剩下几条及原因**。

## Constraints

写范围：`web/knip.config.ts`、`web/src/**`。不 commit / push；不碰实例；不联网。
排除清单**已解除**（`advanced-custom.ts` 可碰，但只做最小删除）。

## Acceptance Criteria

- [ ] AC1：零风险三项已做，且给出各自让 knip 减少的条数。
- [ ] AC2：分批删除循环完成，每批都过了**全量 typecheck**；失败批已回滚。
- [ ] AC3：`bun run knip` 结果（目标 0；否则逐条说明原因）。
- [ ] AC4：`bun test` ≥ 247 pass / 0 fail；六条命令 + `format:check` 输出齐备。
- [ ] AC5：`advanced-custom.ts` 的 import/调用未被破坏（`git diff` 可证）。
- [ ] AC6：未 commit/push。

## Notes

- 上次失败教训：**校验必须覆盖语义类错误（TS2xxx/TS6xxx），不能只过滤 TS1xxx**。
- 结论与命令输出留 `findings.md`。
