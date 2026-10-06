# 日志上限：写入触发 + 字节预算（Go 侧实现留痕）

设计见同目录 `prd.md`（Lead 定稿）。本文件记录实现落点、口径与证据。

## 改动文件

| 文件 | 改动 |
|---|---|
| `common/env.go` | 新增 `GetEnvOrDefaultSize(env, default)` + `ParseSize`：`200MB`/`200mb`/`200M`/`209715200`/`1.5GB`/`4096B`，1024 进制；解析失败记日志并回退默认值（与既有 `GetEnvOrDefault*` 风格一致） |
| `model/log_budget.go`（新） | 记账 + 校准 + 单飞触发 + 写入触发清理一遍；`LogPayloadBudgetBytes()`、`LogRowCap()`、`LogPayloadUsage()` |
| `model/log.go` | `createLog` 写入成功后调用 `noteLogPayloadBytes(log)`（唯一写入口，全部 8 个记录函数共用） |
| `service/system_task.go` | `logCleanupInterval()`：配置了字节预算时默认 0；`logCleanupMaxRows()` 改为委托 `model.LogRowCap()`（单一事实来源）；内存模式告警文案区分"字节预算/行数上限"两种模式 |
| `controller/performance.go` | `/api/performance/logs` 响应增加 `memory_log_bytes` / `memory_log_max_bytes` / `memory_log_rows`（文件日志关闭时同样回填） |
| 测试 | `common/env_size_test.go`、`model/log_budget_test.go`、`service/system_task_log_cleanup_test.go`（新增 3 例 + 2 处环境变量隔离）、`controller/performance_logs_test.go` |

## 配置语义

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `LOG_MEMORY_MAX_BYTES`（新） | 未设=0=关闭 | 载荷字节预算；`200MB` 或 `209715200` |
| `LOG_MEMORY_MAX_ROWS` | 内存模式 20 万 | 行数上限（次级约束，与字节预算**在同一遍**同时生效） |
| `LOG_CLEANUP_INTERVAL` | 配置字节预算时 0（关闭定时器）；显式设置仍生效 | 定时清理间隔 |
| `LOG_CLEANUP_RETENTION_DAYS` | 7 | 改由写入触发那一遍顺带执行（`trimLogsByRetention`） |

## 口径（面板与裁剪必须一致）

- 载荷估算 = 文本列 `len` 之和（content/username/token_name/model_name/group/ip/request_id/upstream_request_id/other）
  + 每行固定开销 128B；SQL 侧用 `COALESCE(LENGTH(col),0)` 同一列集合（`group` 经 `commonGroupCol` 引号化）。
- **这是载荷估算，不含 SQLite 页、索引、WAL 等存储开销**；SQLite/PostgreSQL 的 `LENGTH()` 对多字节文本按字符计，
  Go 侧按字节计，所以面板数字是相对预算而非精确内存占用（注释已写在 `model/log_budget.go` 与 `controller/performance.go`）。
- 面板读的就是裁剪判定用的同一个计数器（`logPayloadBytes`）与同一个 `LogPayloadUsage()`，未另算口径。

## 机制落点

1. 写路径：`createLog` → `noteLogPayloadBytes`：读到预算才 `atomic.Add`，O(1)、无锁、无 IO；未配置预算立即返回（零开销）。
2. 首次使用：`ensureLogPayloadCalibration()` 起一个后台 goroutine 做一次 `SUM(LENGTH(...))` 校准（不阻塞写路径）；
   每遍裁剪结束再做一次校准，计数漂移自愈。
3. 触发：累计超过预算 → `triggerLogPayloadTrim()`；`beginLogPayloadTrim()`（CAS）保证单飞；
   释放单飞后若仍超预算再自检触发一次（避免"最后一次触发被单飞挡掉"后长期超预算）。
4. 一遍清理顺序：保留天数删除 → 行数上限（`TrimLogToMaxRows`）→ 字节预算分批删最老记录 → 校准。
   - 单行就超预算时保留最新一行（`rows <= 1` 即停，绝不删空）；
   - `deleteOldestLogPayloadBatch` 先选 `id + 每行载荷` 再按 id 删除：不依赖 `DELETE ... LIMIT`（MySQL 专有）或 `RETURNING`；
   - 失败只 `LogWarn`，绝不影响写日志主流程。
5. ClickHouse 日志库：与 `LOG_MEMORY_MAX_ROWS` 既有取舍一致，显式忽略并只告警一次（本包没有 CH 方言分支）。

## 实测（四条命令 + gofmt）

见 `.trellis/tasks/10-06-log-byte-budget-write-trigger/` 的最终汇报；命令：`GOWORK=off go vet ./...`、
`GOWORK=off go build ./...`、`cd relaykit && GOWORK=off go build ./...`、`make test`、`gofmt -l`。

## 明确结论：用户设 200MB 后还会不会每 5 分钟跑一次清理？

**不会。** `LOG_MEMORY_MAX_BYTES=200MB` 且未显式设置 `LOG_CLEANUP_INTERVAL` 时，
`logCleanupInterval()` 返回 0（`service/system_task.go`），`logCleanupHandler.Enabled()` 为 false，
调度器不会再创建 log cleanup 任务；清理改由写入触发（含保留天数与行数上限）。
若显式设置 `LOG_CLEANUP_INTERVAL`（如 `10m`），定时任务照旧运行（测试 `TestLogCleanupSchedulingExplicitIntervalWinsOverByteBudget`）。
面板的手动清理入口（`StartLogCleanupTask`）不受影响。
