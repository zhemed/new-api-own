# 设计：LOG_SQL_DSN 支持独立 SQLite 文件/内存库 + 日志清理可调度

## 目标（用户选定路线 B）

让弱盘机器能把**用量日志**整体放进内存（或另一块盘的独立 SQLite），
并且**不会因为日志堆积把内存吃爆**——即日志清理要能自动跑。

底线：**业务数据（账号/token/渠道/任务）永远不进内存库**。

## 1. DSN 形态（仅对日志库生效）

`chooseDB(envName, isLog)` 里新增一个**纯函数解析器**（`isLog=true` 时才走新分支），
其余情况与现状完全一致：

| `LOG_SQL_DSN` | 解析结果 | 说明 |
|---|---|---|
| 空 | 与主库同库（现状）| 不变 |
| `local` | 主库同目录的 `SQLITE_PATH`（现状）| 不变（可用于"主库 MySQL + 日志落 SQLite"）|
| **`memory`** / `:memory:` | `file::memory:?cache=shared`（新增）| 日志只在内存，重启即失 |
| **`sqlite:<path>[?params]`** | 独立 SQLite 文件（新增）| 例：`sqlite:/dev/shm/logs.db?_pragma=journal_mode(WAL)` |
| `postgres://…` / `http(s)://…`（CH）/ 其它 | 现状分支 | 不变 |

**只对日志库开放这两个新形态**：`SQL_DSN=memory` 仍走原分支（避免主库变内存库的灾难性误配）。

## 2. 内存库的连接池约束（关键正确性点）

共享缓存的内存库**只要还有一条连接活着就存在**；连接全被回收时库连同数据消失。
因此内存模式必须强制：

- `SetMaxOpenConns(1)`（单写者，顺带避免共享缓存下的写冲突）
- `SetMaxIdleConns(1)` + `SetConnMaxLifetime(0)`（**不过期**，保证库始终存活）

现有默认值是 `MaxIdle=100 / MaxOpen=1000 / lifetime=60s`（`SQL_MAX_*`），
在内存模式下这些默认值会导致**运行中日志表突然消失**——必须在 `InitLogDB` 里按模式覆盖。

## 3. 自动裁剪（让内存方案安全）

`logCleanupHandler` 目前是 registered **non-scheduled**；改为实现
`ScheduledSystemTaskHandler`（`Enabled/Interval/NewPayload`），复用既有调度器与 DB 租约去重。

| 新增环境变量 | 默认 | 说明 |
|---|---|---|
| `LOG_CLEANUP_INTERVAL` | `0`（关，保持现状）| Go duration，如 `10m`、`1h`；**内存模式默认 `5m`** |
| `LOG_CLEANUP_RETENTION_DAYS` | `7` | `>0` 按时间删；`0` 表示不按时间删 |
| `LOG_MEMORY_MAX_ROWS` | 内存模式 `200000`，否则 `0` | 超过即从最旧开始裁剪；`0` = 不限制 |

- `NewPayload()` 在调度时算出 `TargetTimestamp = now - retentionDays*86400`，并带上 `MaxRows`；
- `Run()` 先按时间删（沿用既有 `DeleteOldLogBatch`），再按行数上限裁（新增
  `model.TrimLogToMaxRows`，用 GORM 的可移植写法，SQLite/MySQL/PG 都能跑）；
- 手动触发的 `StartLogCleanupTask` 行为不变（`MaxRows=0`、沿用调用方给的时间戳）。

## 4. 可观测与告知

- 启动时若日志库是内存模式：`SysLog` 打一条 **WARN**，写明"日志重启即失"、
  裁剪间隔与行数上限的当前取值；
- 若内存模式但 `LOG_MEMORY_MAX_ROWS=0`（用户显式关掉上限）：额外 WARN 提醒有 OOM 风险。

## 5. 改动文件

| 文件 | 改动 |
|---|---|
| `model/main.go` | `chooseDB` 接入新解析器；`InitLogDB` 处理内存模式的连接池覆盖 |
| `model/log.go` | 新增 `TrimLogToMaxRows(ctx, maxRows, batchSize)` |
| `service/system_task.go` | `logCleanupHandler` 实现调度接口；`LogCleanupPayload` 增 `MaxRows`；`Run` 调裁剪；新增 env 读取 |
| `common/env.go` | 新增 `GetEnvOrDefaultDuration`（与既有 `GetEnvOrDefault*` 同族）|
| `MAINTENANCE.md` | 记录 DSN 形态、三个环境变量、内存模式的风险与推荐搭配 |
| 测试 | 见下 |

## 6. 测试

- `model`：DSN 解析表驱动（memory / :memory: / sqlite:path / file:path / local / mysql / postgres / clickhouse）；
- `model`：内存模式端到端——建表、写入、`sqlDB.Close()` 之外的连接回收后仍可读；
  并断言内存模式下连接池被覆盖为 `MaxOpen=1 / ConnMaxLifetime=0`（负向对照：默认 pool 不满足）；
- `model`：`TrimLogToMaxRows` 保留最新 N 条、删除更旧的（SQLite 实测）；
- `service`：`logCleanupHandler` 的 `Enabled/Interval/NewPayload` 随 env 变化（`t.Setenv`）。

## 7. 风险与回滚

| 风险 | 处置 |
|---|---|
| 内存日志重启即失 | 属预期；启动 WARN + 文档显著标注；计费/对账场景明确不建议 |
| 多节点/多副本日志各自为政 | 文档标注：内存模式只适合单实例；`NODE_TYPE=slave` 时不启用日志库迁移 |
| 上限设得过高导致 OOM | 默认 20 万行；显式设 0 时额外 WARN |
| 误把主库配成内存 | 新形态仅对 `LOG_SQL_DSN` 生效；`SQL_DSN=memory` 行为不变（照旧报错）|

**回滚**：全部行为都由 `LOG_SQL_DSN` 与三个环境变量**显式开启**，默认路径零变化；
代码层面回滚 = revert 本次提交；运行层面回滚 = 去掉环境变量。
