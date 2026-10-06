# 评估：弱盘机器能否把日志写入内存

## 结论（先行）

1. **这里有两类日志，写盘压力差一个量级**：应用日志是"追加写文件"（便宜）；用量日志是
   **每行一个事务 + 一次 fsync**（贵）。弱盘真正的痛点在后者的 fsync，而不是日志总量。
2. **零代码就能拿掉大部分压力**：① 应用日志不落盘（`-log-dir=` 只走 stdout，或指向 tmpfs）；
   ② 日志库开 WAL + `synchronous=NORMAL`（实测 `journal_mode` delete→wal、`synchronous` 2→1）。
3. **"日志整体进内存"可行，但要分部署形态**：主库不是 SQLite 时**纯配置**可达成
   （`LOG_SQL_DSN=local` + `SQLITE_PATH=/dev/shm/...`）；主库就是 SQLite 的单机部署需要一个小改动
   （`LOG_SQL_DSN` 目前没有"另一个 SQLite 文件 / 内存库"的表达方式）。
4. **内存方案必须先解决"上限"**：清理任务现在是**手动触发**、不在调度器里，
   日志堆在内存里不裁剪会 OOM。调度器与接口都已存在，让清理处理器实现
   `ScheduledSystemTaskHandler` 即可自动裁剪（小改动）。
5. **不建议**：整个 SQLite 主库放 tmpfs（重启丢账号/token/渠道）、MySQL datadir 放 tmpfs、
   或在没有自动裁剪的前提下把日志放内存。

## 事实与证据

### 两类日志的写入路径

| 类别 | 代码 | 行为 |
|---|---|---|
| 应用日志 | `logger/logger.go` | 写 `-log-dir`（CLI 标志，默认 `./logs`）下 `oneapi-<启动时间>.log`；`io.MultiWriter(os.Stdout, fd)` **双写**；每 100 万行换新文件（`maxLogCount`）；**无硬上限**；`-log-dir=`（空）即完全不写文件 |
| 用量日志 | `model/log.go:103` `LOG_DB.Create(log)` | 逐条 INSERT，**每行一个隐含事务**；`LOG_SQL_DSN` 未设时 `LOG_DB = DB`（与主库同库同文件） |

### 为什么弱盘怕的是 fsync（本机实测）

用项目自带驱动（`github.com/glebarez/sqlite v1.9.0`）打开 SQLite，读回 PRAGMA：

| DSN | journal_mode | synchronous |
|---|---|---|
| 项目默认形态 `x.db?_busy_timeout=30000` | **delete**（回滚日志） | **2 = FULL** |
| 追加 `&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)` | **wal** | **1 = NORMAL** |
| `file::memory:?cache=shared` | memory | 2 |

- GORM 默认 `SkipDefaultTransaction=false`（`model/gorm_logger.go:newGormConfig` 只设了
  `PrepareStmt` 与 Logger）→ 每写入一条日志就是 BEGIN/COMMIT，在 delete+FULL 下**每次提交都 fsync**。
- 顺带发现：默认 DSN 里的 `_busy_timeout=30000` **并未生效**（探针读回 5000 = 驱动默认）；
  要生效应写 `_pragma=busy_timeout(30000)`。

### 内存承载的可达路径

| 形态 | 前置条件 | 改动量 | 备注 |
|---|---|---|---|
| 应用日志进 tmpfs | `-log-dir=/dev/shm/newapi-logs` | 0（改启动参数）| 需给 tmpfs 设 `size=`，否则吃内存 |
| 应用日志完全不落盘 | `-log-dir=` | 0 | 只剩 stdout；容器层再加 `--log-driver=none`/`max-size` |
| 用量日志进内存（主库非 SQLite）| `LOG_SQL_DSN=local` + `SQLITE_PATH=/dev/shm/newapi-logs.db?...` | 0（改环境变量）| `chooseDB` 对 `local` 复用 `SQLitePath`；主库在 MySQL/PG 时该文件只放日志 |
| 用量日志进内存（主库即 SQLite）| 需要 `LOG_SQL_DSN` 支持 SQLite 文件路径或 `memory` 关键字 | 小（~10–20 行 + AutoMigrate）| 实测 `file::memory:?cache=shared` 可用；重启即失、单进程可见 |
| 日志写别的盘/别的机器 | `LOG_SQL_DSN` 指向 MySQL/PG/**ClickHouse** | 0 | 代码已支持 ClickHouse 日志库；弱盘单机上引入 CH 反而更重 |
| 批量攒批写（减少事务数）| 代码改动 | 中 | 内存队列 + 定时/定量刷盘，落盘量不变但事务数降到 1/N |

### 内存方案的风险清单

- **重启即失**：内存里的用量日志重启后为空（面板日志页随之清空）。
- **无自动上限**：`logCleanupHandler` 明确是 registered **(non-scheduled)**，清理靠手动/接口触发
  （`service/system_task.go:77` 注释与 `StartLogCleanupTask`）；调度器
  （`runSystemTaskScheduler` + `ScheduledSystemTaskHandler`）与周期机制**已存在**，
  让清理处理器实现该接口即可自动裁剪。
- **多实例/多节点**：内存库是进程内的，`NODE_TYPE=slave` 或多副本部署时日志各自为政，面板看不全。
- **账务计费数据也在这张 `logs` 表里**（消费日志 `type=2` 含 quota 字段）：如果下游有对账、
  用量统计依赖它，重启丢日志等于丢账 → 这类部署应选"落盘但减轻写放大"，而不是纯内存。

## 推荐分档

| 场景 | 建议 |
|---|---|
| 磁盘弱、日志只用于排障（不需要历史） | 应用日志 `-log-dir=` 或 tmpfs；日志库开 **WAL+NORMAL**；若仍嫌写盘，再把日志库放 `/dev/shm` |
| 磁盘弱、但日志要能查历史 | 只做 **WAL+NORMAL**（+ 可选：攒批写）；不要放内存 |
| 有第二块盘/第二台机器 | `LOG_SQL_DSN` 指过去（含 ClickHouse 选项），本机只留业务库 |
| 依赖日志对账/计费 | 不放内存；优先 WAL+NORMAL + 定时清理策略 |

## Acceptance Criteria

- [x] 说清两类日志与各自写入特征（含事务/fsync 机制）
- [x] 用本机实测证明 PRAGMA 可由 DSN 配置（delete/FULL → wal/NORMAL）
- [x] 给出 4 类内存承载形态，各自前置条件、改动量、丢失风险
- [x] 指出不建议的做法与原因
- [x] 给出按场景分档的推荐
