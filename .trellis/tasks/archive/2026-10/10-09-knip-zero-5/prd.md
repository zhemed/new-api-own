# knip 收口：66 条刻意保留逐符号登记

## Goal
把剩余 66 条（53 exports + 13 types）按 knip 的 `ignoreIssues` **按文件 + 符号名逐条**登记，
让 `bun run knip` **exit 0**。它们是跨模块类型契约（删除需同步改调用方 = 改行为，超授权）。

## Requirements
- R1 必须**精确到符号**（不得忽略整文件的所有导出），保证以后新增死导出仍会被报出。
- R2 每个文件一行注释写明保留理由。
- R3 目标：`bun run knip` exit 0；`knip --include files` 仍 0；`tsgo -b --force` 0 错误；
  `lint` 0/0；`bun test` ≥247 pass；`format:check` 干净；`build` exit 0。
- R4 回报：config diff（逐条符号 + 每文件理由）、knip exit code 与输出尾部、五条命令结果、
  累计统计（两轮共删文件/符号数、398→0 全过程）。

## Acceptance Criteria
- [ ] AC1：`bun run knip` exit 0，且 config 中每条都是"文件 + 符号数组"形态。
- [ ] AC2：五条命令全绿。
- [ ] AC3：未 commit/push。
