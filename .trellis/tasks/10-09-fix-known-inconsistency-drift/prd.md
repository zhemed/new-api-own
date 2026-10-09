# 订正「已知不一致」3 条漂移（DTO / CH 行数上限 / knip）

- 任务：`10-09-fix-known-inconsistency-drift`
- 执行：`review-ops`（写范围：`MAINTENANCE.md`）
- 共享任务：`task-26`（Lead 批准订正）
- 前置：`10-09-fix-stale-known-inconsistency`（复核出这 3 条漂移）

## 目标

按 Lead 批准的口径订正「已知不一致」表第 1、5、6 条；每条改动都必须能指到**代码行号或命令输出**。

## 改动（已落地）

| # | 位置 | 改后内容 | 依据 |
|---|---|---|---|
| 1 | 第 1 行（DTO）| 标题改为「客户端请求 DTO 用非指针标量（**两处行为不同**）」；位置拆成两句：`relay/common/relay_info.go:849` `Duration int` + `json:"duration,omitempty"`；`dto/video.go:7` `Duration float64` + `json:"duration"`（**无** `omitempty`）；钳制位置改为 `relay/relay_task.go:125-126`（注释 `:124`）与 `relay/common/relay_utils.go:148-165` `validateTaskDurationBounds` | 逐行 `sed -n` 核对：`relay_info.go:849`、`dto/video.go:7`、`relay_task.go:125-126`、`relay_utils.go:148`（函数头）/`:165`（函数尾）|
| 2 | 第 5 行（ClickHouse）| 位置改为「**事实来源 `model.LogRowCap()`（`model/log_budget.go:60-81`）**；`service/system_task.go:151-155` 只是委托」；说明补 CH 分支行号 `model/log_budget.go:64-71`，并补「字节预算对 CH 同样关闭：`model/log_budget.go:85-97`」 | `sed -n '60p'` = `func LogRowCap() int64`；`:64` = CH 判断；`:70` = `return 0`；`:85` = `func LogPayloadBudgetBytes`；`:90` = CH 判断 |
| 3 | 第 6 行（knip）| 整句重写：保留 `ui/**`、`ai-elements/**` 的 ignore 与「不要据 ignore 删依赖」警告；明确 **已移除** 的两条过时 ignore（`src/routeTree.gen.ts` 由 `src/main.tsx` 可达；`src/i18n/static-keys.ts` 已接进 `scripts/check-i18n-keys.mjs` → `i18n:check` 入口，注释"不要再加回来"）；实测 `bun run knip --include files` 无未使用文件输出 | `web/knip.config.ts:19`（ui）、`:22`（ai-elements）是仅存的两条 `src/**` ignore；`bun run knip --include files` 实测无输出 |

**与 Lead 口径的微小差异（更精确）**：钳制位置写成 `relay/relay_task.go:125-126`（Lead 写 124-126，124 实为注释）与 `relay/common/relay_utils.go:148-165`（Lead 写 148-160；函数实际到 `:165` 结束）。

## 校验

- `MAINTENANCE.md` 代码围栏 **40 个（配平）**；`README.md` 14（配平）；`.env.example` 0。
- 「已知不一致」表 = 表头 + 分隔 + **6 行**（仍全部是"决定不改"）。
- 新引用行号逐条 `sed -n` 复核（见上表依据列）。
- 只改 `MAINTENANCE.md`（`git diff --stat`：1 file changed, 5 insertions(+), 4 deletions(-)，含上一轮的整表行数变化）；未 commit/push、未碰代码与实例。
