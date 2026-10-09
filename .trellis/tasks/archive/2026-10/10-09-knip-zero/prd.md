# knip 全量归零（真修 + 可追溯配置）

## Goal

让 `bun run knip`（全量）**exit 0**，且每一处例外都能在 `knip.config.ts` 里**追溯到一条理由**；
**不得**用"忽略整个 `src/**`"之类的粗暴手段换绿。

## Requirements

### R1 先分类，再动手
把每类条目按位置归组，并按以下规则处置：
- **成套组件库**（`src/components/ui/**`、`src/components/ai-elements/**`）：属**刻意保留的产品面**，**不删**；
  用 `knip.config.ts` 里**按路径**的正式配置表达，并在注释写明理由（可复用底座、外部消费者是本仓库后续开发）。
- **非库的真实死代码**（0 引用、不属上述底座）：**删除**；删前逐个 `grep -rn`（含动态 import、字符串引用、`.mjs`/`.json`），全空才删。
- **不确定者**：保留，并在 config 写明理由。

### R2 未使用依赖
- 逐条查明"谁在用"：若仅被 `ui/**` / `ai-elements/**` 等**已忽略目录**引用 → `ignoreDependencies` **逐条列出**并注明原因，**禁止通配**。
- 若确认**全仓无任何引用** → 记录在案（删依赖会牵动 lockfile 且本机禁止联网安装，需 Lead 决策），本轮用带理由的 ignore 表达。

### R3 重复导出与配置提示：真修
- `Duplicate exports (1)`（`useDialogState|default`）：按 knip 建议**重命名/去掉重复导出**，不 ignore。
- `Configuration hints (2)`（`tailwindcss`、`tw-animate-css` 的 `Remove from ignoreDependencies`）：**移除该配置项**。

### R4 必须一并删除的既有死代码
- `features/pricing/lib/mock-stats.ts` 的 `formatTokenVolume`；
- `features/pricing/components/model-details-api.tsx` 的 `ApiTabIcon`（若确认 0 引用）。

### R5 验证
- `bun run knip` **exit 0**；`bun run knip --include files` 仍为 0。
- 六条命令全绿（`bun test` 基线 **224 pass / 0 fail**，不得下降）+ `format:check`。

## Constraints

写范围：`web/knip.config.ts`、`web/src/**`（可删死代码/导出）。
**排除**（另一位同时改，禁止触碰）：`web/src/features/system-settings/general/channel-affinity/constants.ts`、`web/src/features/channels/lib/advanced-custom.ts`。
不 commit / push；不碰用户实例；不联网（不改依赖版本、不跑安装）。

## Acceptance Criteria

- [ ] AC1：分类结果（各类多少条、分别"删除 / 配置 / 保留"）与依据。
- [ ] AC2：`bun run knip` exit 0，且 config 每条例外都有理由注释。
- [ ] AC3：`formatTokenVolume` 与 `ApiTabIcon` 已删（或给出未删的实证理由）。
- [ ] AC4：重复导出与配置提示已真修（不是 ignore）。
- [ ] AC5：`knip --include files` = 0；六条命令 + `format:check` 输出齐备；`bun test` ≥ 224。
- [ ] AC6：两个排除文件零改动；未 commit/push。

## Notes

- 若"在不削弱检查的前提下无法归零"：**不硬凑**，如实回报剩余条目与原因，交 Lead 定夺。
- 结论与命令输出留 `findings.md`。
