# 弱盘日志路线选型与落地

## 背景

评估已归档：`.trellis/tasks/archive/2026-10/10-06-memory-log-eval/prd.md`
（痛点是用量日志"每行一事务 + fsync"；日志进内存需先解决自动裁剪上限）。

**用户选定路线 B**：让 `LOG_SQL_DSN` 支持独立 SQLite 文件/内存库，并把日志清理改为可调度。
设计见同目录 `design.md`。

## 已实现

| 文件 | 改动 |
|---|---|
| `model/main.go` | `resolveLogSQLiteTarget()` 解析日志专用 DSN（`memory` / `:memory:` / `sqlite:<path>`）；`chooseDB` 仅对日志库走新分支；`UsingInMemoryLogDatabase()`；内存模式下 `InitLogDB` 强制 `MaxOpenConns=1 / MaxIdleConns=1 / ConnMaxLifetime=0` 并打 WARN |
| `model/log.go` | 新增 `TrimLogToMaxRows(ctx, maxRows, limit)`：按主键裁掉最旧的行，分批返回删除数（GORM 可移植写法）|
| `service/system_task.go` | `logCleanupHandler` 实现 `Enabled/Interval/NewPayload`（接入既有调度器）；`LogCleanupPayload` 增 `MaxRows`；`LogCleanupResult` 增 `TrimmedCount`；新增 `trimLogsToMaxRows()`；内存模式下默认 5m 周期 / 20 万行上限，并打一次性 WARN |
| `common/env.go` | 新增 `GetEnvOrDefaultDuration` |
| `MAINTENANCE.md` | 新增两节：内存/独立盘日志的配置与风险；WAL+NORMAL 降 fsync（含 `_busy_timeout` 未生效的实测发现）|

边界：**新形态仅对 `LOG_SQL_DSN` 生效**，`SQL_DSN=memory` 行为不变（主库绝不进内存）。

## 验证

**单元（新增 8 个用例，全部通过）**

- `model/log_retention_test.go`：DSN 解析表驱动（9 例）；内存模式端到端（钉连接 + 行可读）；
  **负向对照**——`file::memory:?cache=shared` 在没有存活连接时确实连表一起消失（证明"钉连接"不是多余的）；
  `TrimLogToMaxRows` 保留最新 N 条、无上限时不动数据。
- `service/system_task_log_cleanup_test.go`：磁盘模式默认不调度（opt-in）；显式 `LOG_CLEANUP_INTERVAL`
  生效；内存模式默认 `5m` + 20 万行上限；`LOG_CLEANUP_RETENTION_DAYS=0` 时按时间不删（cutoff=1）。

**启动冒烟（本机真实进程，两种形态各跑一次）**

1. `LOG_SQL_DSN=memory` + `LOG_CLEANUP_INTERVAL=3s` + `LOG_MEMORY_MAX_ROWS=5`：
   启动日志出现两条内存提示与 RAM 告警；登录 9 次后面板日志接口读到 7 行（说明写进了内存库、能被读出）；
   再登录 8 次后 `total` 稳定回到 **5**（上限确实在自动裁），全程未手动触发清理。
2. `LOG_SQL_DSN=sqlite:/dev/shm/...?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)`：
   启动提示"独立 SQLite 文件"；日志接口可读；`/dev/shm/dsh-logs.db` + `-wal` + `-shm` 三个文件存在
   → **WAL 确实生效**。冒烟环境已删除。

**门禁**：`gofmt` 干净；根模块与 relaykit 的 `go vet` / `go build` 均 exit=0；
`make test` exit=0（38 包 ok，无 FAIL）。

## 未做（明确边界）

- 未把日志清理做成"面板可配"（仅环境变量；面板设置项需另一轮）；
- 未做批量攒批写（评估里的可选优化，收益与内存方案重叠）；
- 未动前端。

## Acceptance Criteria

- [x] `LOG_SQL_DSN` 支持独立 SQLite 文件与内存库，且仅对日志库生效
- [x] 内存模式钉连接，避免日志表运行中消失（含负向对照测试）
- [x] 日志清理可调度、内存模式自动裁剪并有行数上限（默认 5m / 20 万行）
- [x] 单元测试 + 两种形态的启动冒烟
- [x] `make test` / vet / build 全绿；文档更新
- [ ] 提交与发布：**待用户授权**（当前改动仍在工作区，未提交）
