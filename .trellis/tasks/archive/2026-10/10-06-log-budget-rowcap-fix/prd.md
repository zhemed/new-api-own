# 修正文档：行数上限已并入写入触发（与字节预算同遍生效）

- 任务：`10-06-log-budget-rowcap-fix`
- 执行：`review-ops`（写范围：`.env.example`、`MAINTENANCE.md`、`README.md`）
- 共享任务：`task-24`（Lead 指派）
- 前置：`10-06-log-byte-budget-docs`（同一批文档，本轮只改语义错误那部分）

## 问题

上一轮我按当时的代码写下「配了字节预算后，`LOG_MEMORY_MAX_ROWS` 默认不再自动执行，要它生效必须显式开定时器」。
后端**最终实现**把行数上限并进了写入触发那一遍，这句话已不成立 → 必须撤回。

## 目标（按最终代码修正）

1. 撤掉"行数上限默认不生效 / 需显式开定时器"的表述（`.env.example`、MAINTENANCE 表格 + 正文）；
2. 写清新的裁剪顺序与"单一行数上限事实来源"；
3. 其余内容（写入触发、异步单飞、载荷估算口径、留 20–30% 余量、如何确认没有定时任务）保留；
4. 报出复核过的**最终**代码行号。

## 最终代码事实（已逐行核对，文件 md5 稳定）

| 事实 | 代码 |
|---|---|
| 裁剪顺序：保留天数 → **行数上限** → 按体积裁最老 → 校准 | `model/log_budget.go:234-252` `trimLogsToPayloadBudget`（保留 `:257`、行数 `:277-290`、体积 `:293-327`、校准 `:251`）|
| 行数上限复用既有 `TrimLogToMaxRows` | `model/log_budget.go:277-290` 里的调用 |
| 行数上限策略**单一事实来源** `model.LogRowCap()`：显式 `LOG_MEMORY_MAX_ROWS` 优先 → 内存模式默认 `200000` → 磁盘 0（不设上限） | `model/log_budget.go:60-81`、常量 `:27`、`:32` |
| 定时清理与写入触发**共用**同一上限 | `service/system_task.go:151-155`（`logCleanupMaxRows()` 直接 `return model.LogRowCap()`）|
| 字节预算与行数上限都是"次级约束"，同遍生效；关定时器不影响 | 同上 |
| 未配字节预算 = 行为与以前一致（写路径直接返回） | `model/log_budget.go:172-177` |
| 手动触发不带行数上限（payload 只有 TargetTimestamp/BatchSize） | `service/system_task.go:284-289` + handler `:541-542`（`payload.MaxRows <= 0` 直接返回）|
| 其它引用（写入触发、单飞、载荷口径、面板字段、ClickHouse） | `:172-182`、`:198-221`、`:38`、`:101-104`、`controller/performance.go:196-202`、`model/log_budget.go:85-99` |

## 验收

- 三个文件中不再出现被撤回的表述（`grep` 验证为空）；
- 新表述与代码一致，行号逐条复核；
- 不 commit/push、不改代码、无网络动作。

## 交付记录

| 文件 | 改动 |
|---|---|
| `.env.example:41` | 行数上限注释改为「与字节预算**同遍生效**：写入触发时先按保留天数、再按行数、最后按体积裁」|
| `MAINTENANCE.md` 表格 | `LOG_MEMORY_MAX_ROWS` 行改为「定时与写入触发两条路径共用同一上限（`model.LogRowCap()`）」|
| `MAINTENANCE.md` 正文（197-234）| 新增裁剪顺序条目；把"只在定时清理里执行…要它继续生效就显式设 `LOG_CLEANUP_INTERVAL`"替换为「同遍生效的次级约束 + 单一事实来源 + 关定时器不影响」；手动触发那条改为"不带行数上限（行数/体积本来就在写入触发那一遍管）"；ClickHouse 那条补上行数上限同样被忽略 |
| `README.md` | **未改**（原文只说"写入触发、超预算裁最老"，无误导，无需动） |

校验：`grep "只在定时清理里执行\|要它继续生效就显式设\|默认不再自动生效"` → 空；围栏 README 14 / MAINTENANCE 40 配平；
引用行号逐条 `sed -n` 复核通过；代码文件 md5 在核对期间稳定。
