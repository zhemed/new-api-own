# knip 清仓 round3：单文件分批 + 强制 typecheck 复核

## Goal

冻结期单线程（**唯一写入者**）清理剩余 `Unused exports (301)` + `Unused exported types (97)`，
目标 `bun run knip` **exit 0**。每批只改一个文件，用**强制模式 typecheck 的错误签名**做安全网。

## Requirements

### R1 安全规程（违反即停手）
- 只允许本会话写入；不叫醒其它成员、不并行。
- **禁止跨文件批量回滚**：回退只对**本轮刚改过的那一个文件** `git checkout -- <确切文件>`；
  **严禁** `git stash` / `git clean` / 目录级或全仓 checkout / `--no-verify`。
- 校验必须 `bunx tsgo -b --force`（不信增量结果）。
- **按错误签名**（文件, 代码, 符号）比对，不比总数（删声明会让行号位移）。
- 每批只改一个文件；改完立刻强制 typecheck：有错 → **只回退该文件**并记入"保留"清单；无错 → 继续。
- 每 20 批跑一次 `bun test`（基线 **247 pass / 0 fail**）与 `bun run lint`（应保持 **0 warnings / 0 errors**）。

### R2 删除判定
- 只删"**去掉 `export` 后文件内也无人使用、且全仓无引用**"的声明（连同声明体）；
  若该符号在**本文件内仍被使用** → 只去掉 `export` 关键字（保留实现）。
- 连带清理**因此未用**的 import。
- **特殊区域加倍小心**（字符串/动态引用风险）：
  - `src/assets/brand-icons/**`（26 条）：先查是否存在按名字取组件的**动态用法**
    （`Icons[name]`、`iconMap[...]`、注册表、`.map()` 拼名）——有则**保留该文件全部相关导出**并写明；
  - 任何名字出现在 `.json` / `.md` / 脚本文本里的符号 → 保留并注明。
- 不确定 → **保留**（宁留条目，不改错）。
- 即使某文件删完已无引用，**也不要删文件**（文件层面 knip 是干净的）。

### R3 收工判定
`bun run knip` exit 0（若仍有剩余，**逐条**列出并给保留原因，不许为绿而 ignore 全仓）｜
`bunx tsgo -b --force` **0 错误**｜`bun run lint` **0/0**｜`bun test` **≥247 pass / 0 fail**｜
`format:check` 干净｜`bun run build` exit 0｜`bun run knip --include files` 仍 0。

### R4 回报
knip 各节前后计数｜删除文件数 + 删除符号数｜**保留清单**（文件 + 原因）｜最终命令输出｜
过程中回退过的文件｜任何"有风险所以没删"的情况。

## Constraints

写范围：`web/src/**`（可删死代码/导出）、`web/knip.config.ts`（仅在"确实删不掉"时逐条写理由）。
不 commit / push；不碰用户实例；不联网。
冻结期：**Lead 不会改动任何东西**（单写入者）。

## Acceptance Criteria

- [ ] AC1：每批单文件 + 强制 typecheck 复核，回退只针对该文件（有据可查）。
- [ ] AC2：`bun run knip` exit 0；若否，剩余逐条列出 + 原因。
- [ ] AC3：`tsgo -b --force` 0 错误；lint 0/0；`bun test` ≥247；`format:check` 干净；`build` exit 0。
- [ ] AC4：`knip --include files` 仍 0。
- [ ] AC5：brand-icons 等风险区已查证（动态用法/字符串引用）并给出结论。
- [ ] AC6：未 commit/push；回退未越出"单文件"边界。

## Notes

- 上一轮教训：`tsgo -b` 增量会漏报；跨文件批量回滚会误伤并发写入者（本轮已冻结，且仍禁止）。
- 结论与命令输出留 `findings.md`。
