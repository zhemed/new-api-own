# 清理 pricing mock-stats 死代码 — 交付

**结论**：4 个生成器 + 级联的 6 个类型/常量/辅助函数 + `seed.ts` 的 2 个孤立助手已删；
`mock-stats.ts` 854 → 445 行、`seed.ts` 63 → 43 行；knip **收敛 6 条**（-2 类型、-4 导出）；
`bun test` **224 pass / 0 fail**（基线持平）；六条命令 + `format:check` 全绿（knip 基线红未变）。

## 一、删前逐个复核（R1 硬门槛，全部为空才算）

```
$ grep -rn "<name>" web/            # 覆盖 .ts/.tsx/.mjs/.js/.json（含测试、动态 import、字符串引用）
buildLatencyTimeSeries   → 仅 mock-stats.ts:352 定义处
buildUptimeSeries        → 仅 mock-stats.ts:387 定义处
buildGroupPerformance    → 仅 mock-stats.ts:311 定义处 + 两个（同为死代码的）调用点
buildAppRankings         → 仅 mock-stats.ts:430 定义处
```
再做一次**全仓**（排除 `node_modules`/`.git`/`.trellis`）复核：同样只有 `mock-stats.ts` 内部自引用。**4 个全部 0 外部引用 → 删除。**

## 二、级联清理（逐个查证，非臆测）

| 符号 | 引用情况 | 处置 |
|---|---|---|
| `GroupPerformance` | 仅 `buildGroupPerformance` | **删** |
| `AppRanking` | 仅 `APP_TEMPLATES` + `buildAppRankings` | **删** |
| `APP_TEMPLATES` | 仅 `buildAppRankings` | **删** |
| `ProfileSpec` / `PROFILE_SPECS` | 仅 `buildGroupPerformance` | **删** |
| `rangeFromSeed` / `applyGroupFactor` / `groupFactor` | 仅 `buildGroupPerformance` | **删** |
| `randomInRange` / `randomIntInRange`（seed.ts） | 删完后：前者仅被后者调用，后者 0 引用 | **删** |
| **`PROFILE_BY_NAME`** | 被 `apiCategoryOf` 使用（→ `buildSupportedParameters` / `buildRateLimits` 真实路径） | **保留** |
| **`LatencyTimePoint`** | 被 `model-details-charts.tsx:30,97` import | **保留** |
| **`UptimeDayPoint`** | 被 `model-details-charts.tsx` / `model-details-uptime-sparkline.tsx` / `model-details-performance.tsx` import | **保留** |
| **`aggregateUptime`** | `model-details-uptime-sparkline.tsx:151` 在用 | **保留** |
| **`buildSupportedParameters` / `buildRateLimits`** | `model-details-api.tsx:559,671` 在用（已加"示例数据"标注） | **保留，原样未动** |
| **`hashStringToSeed` / `seededRandom`** | `buildRateLimits` 仍在使用 | **保留** |
| `formatTokenVolume` | **0 引用**（但它在本次改动前就已死，不属"因此变成未使用"） | **保留并上报**（见第五节，等指示再删） |

## 三、knip 前后对比（重点）

```
$ bun run knip            # BEFORE → AFTER
Unused exported types (99)  → (97)     # -2: AppRanking, GroupPerformance
Unused exports        (311) → (307)    # -4: buildAppRankings / buildGroupPerformance /
                                       #     buildLatencyTimeSeries / buildUptimeSeries
Unused dependencies (6) / devDependencies (2) / Duplicate exports (1) / Configuration hints (2) → 不变
exit 1（基线红）→ exit 1（剩余 404 条为仓库既有，非本次引入）

$ bun run knip --include files     # BEFORE: 无输出  →  AFTER: 无输出，exit 0
```
**收敛 6 条**（正是被删的 4 个函数 + 2 个类型）；其余基线红条目未受影响。

## 四、六条命令 + format:check

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error / 0 warning |
| 3 | `bun test` | ✅ **224 pass / 0 fail**（42 files）——与基线持平，无下降 |
| 4 | `bun run build` | ✅ exit 0（Total 57331.5 kB / gzip 16541.2 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— 基线红，但**本次收敛 6 条**（见第三节） |
| 6 | `bun run i18n:check` | ✅ 4030 literal + 455 static = 4181 键，七语言齐备 |
| 附 | `bun run format:check` | ✅ exit 0 |

## 五、坦率说明：一次自伤并已修复

我的删除脚本对 **数组字面量** 的块边界判断错了：`APP_TEMPLATES` 以 `]` 结尾，而我的收尾条件是"行首 `}`"，
于是它顺势吞掉了紧跟其后、**属于真实路径**的 `PROFILE_BY_NAME`。**typecheck 立即报错**（`Cannot find name 'PROFILE_BY_NAME'`），测试同时红 3 个。

处置：从 `HEAD` 取回原文**逐字节还原**（用 `diff` 验证函数体与 HEAD 完全一致），随后 typecheck 与 224 个测试恢复全绿。
最终 diff 里 `PROFILE_BY_NAME` 表现为**移动**（旧位置删、新位置加），非删除。
教训已记：按"行首 `}`"配对只适用于块语句，数组/对象字面量必须按各自的收尾符判断。

## 六、边界

- 只改：`web/src/features/pricing/lib/mock-stats.ts`、`web/src/features/pricing/lib/seed.ts`（均在授权范围内）。
- 调用方（`model-details-api.tsx` / `model-details-charts.tsx` / `model-details-performance.tsx` / sparkline）**零改动**。
- 工作区里 `channels/lib/advanced-custom.ts`、`system-settings/general/channel-affinity/constants.ts` 的改动属另一位成员，我未触碰。
- 未 commit/push；未碰实例。

## 七、待决

`formatTokenVolume`（`mock-stats.ts:65`）**当前 0 引用**（knip 仍在报）。它原本服务于已删的 apps 榜单，
但**在本次改动之前就已死**，不属于"因本次删除而变成未使用"，故按指令未删。
如需一并清掉，回一句即可（一行改动 + 六条命令复跑）。
