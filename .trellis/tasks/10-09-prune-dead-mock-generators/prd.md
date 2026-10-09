# 清理 pricing mock-stats 死代码生成器

## Goal

删除 `web/src/features/pricing/lib/mock-stats.ts` 中 **4 个全仓 0 引用**的 mock 生成器，并顺带清理**因此**变成未使用的辅助函数 / 类型 / 常量；
**只删死代码，不改任何真实行为**。用 knip 前后对比验证收敛效果。

## Requirements

### R1 删前逐个复核 0 引用（硬门槛）
对每个待删符号，先做全仓检索，**全部为空**才允许删除：
```bash
grep -rn "<name>" web/src
```
必须覆盖：普通 import、类型 import、测试文件、动态 `import()`、以及**字符串引用**（如配置表 / 路由表里写字面量）。
**任何一处命中 → 该符号不删**，并在回报中列出命中位置。

待删名单：
`buildLatencyTimeSeries`、`buildUptimeSeries`、`buildGroupPerformance`、`buildAppRankings`

### R2 级联清理必须逐个查证（禁止臆测）
删除上述函数后，逐个判断以下符号**是否仍被真实路径使用**，只删确定不再被引用的：

`PROFILE_BY_NAME`、`PROFILE_SPECS`、`rangeFromSeed`、`applyGroupFactor`、`groupFactor`、
`APP_TEMPLATES`、`GroupPerformance`、`LatencyTimePoint`、`UptimeDayPoint`、`AppRanking`，
以及 `seed.ts` 的 `hashStringToSeed` / `randomInRange` / `randomIntInRange` / `seededRandom`。

**必须保留**（已核实在真实路径上）：
- `buildSupportedParameters`、`buildRateLimits`（保留的 mock，已加"示例数据"标注）；
- `aggregateUptime`（`model-details-uptime-sparkline.tsx` 在用）；
- `formatTokenVolume`、`formatRateLimit`（如仍被引用）；
- 任何被真实组件 import 的类型（如 `UptimeDayPoint` 被 `model-details-performance.tsx` / sparkline 使用）。

### R3 不改真实行为
- 被保留的 mock 与其输出**原样不变**；
- 不得改动调用方（`model-details-*.tsx` 等）。

### R4 验证
- 六条命令全绿：`typecheck / lint / test / build / knip / i18n:check`；
  `bun test` 基线 **224 pass / 0 fail**，**不得下降**。
- 额外 `format:check`。
- 额外跑 `bun run knip --include files` 与 `bun run knip`，给出**删除前后对比**，说明本次是否收敛了 knip 基线。

## Constraints

- 只改：`web/src/features/pricing/lib/mock-stats.ts`、必要时 `web/src/features/pricing/lib/seed.ts` 及其类型定义。
- **不碰** lint 命中文件（另一位成员同时改）；不 commit / push；不碰实例；不新增依赖。

## Acceptance Criteria

- [ ] AC1：4 个待删符号逐个给出 `grep -rn` 结果（全空）与"已删/未删"结论。
- [ ] AC2：级联清理清单逐个给出去留结论 + 依据（哪条真实引用路径保留、为什么某个可删）。
- [ ] AC3：保留的 mock 与调用方零改动（可用 `git diff` 复核）。
- [ ] AC4：六条命令 + `format:check` 输出齐备；`bun test` ≥ 224 pass / 0 fail。
- [ ] AC5：knip 前后对比（`--include files` 与默认各一份），说明是否收敛。
- [ ] AC6：改动仅限允许文件；未 commit/push。

## Notes

- 上一轮已核实这 4 个生成器 0 调用（`.trellis/tasks/10-09-kpi-color-and-mock-labels/findings.md`），本轮需**重新复核**后才动手（工作区在此期间有他人改动）。
- 结论与命令输出留 `findings.md`。
