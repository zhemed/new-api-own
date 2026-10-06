# 后端审查与修复（团队） — 审查结果

## 结论（结论先行）

本轮聚焦 lead 指定的"日志承载功能跨库复核"（`LOG_SQL_DSN=memory|:memory:|sqlite:<path>`、
`TrimLogToMaxRows`、日志清理调度），并扫了计费安全 / JSON 规则 / DTO 指针语义 / 错误处理 / 并发。
**共 6 条发现：1 条已修（含测试），1 条注释更正，4 条记录为待确认/未修**（其中 1 条 ClickHouse 需 Lead 决策）。
四条验收命令全部通过（时间点见 `verify.log`）。

## 详细发现清单

| # | 严重度 | 位置 | 状态 |
|---|--------|------|------|
| F-1 | 中 | `model/log.go:743` 对比 `model/log.go:711-730` | 未修·待确认（需决策） |
| F-2 | 中 | `model/main.go:132-145`（修复前） | **已修** + 测试 |
| F-2b | 中 | `model/log_retention_test.go:125-147` | **已修**（测试自包含化） |
| F-3 | 低 | `model/log.go:739-751` | **已修（注释）** |
| F-4 | 低 | `relay/helper/valid_request.go:195-243` | 未修·待确认 |
| F-5 | 低 | `relay/common/relay_info.go:849`、`dto/video.go:10-12` | 未修·待确认 |
| F-6 | 低 | `service/system_task.go:403-425` | 未修（低优先级） |

### F-2（已修）`sqlite::memory:` 前缀的内存日志库被误判为文件库

- 证据（修复前）：`resolveLogSQLiteTarget` 只识别裸 `memory` / `:memory:` / `:memory:?`；
  带 `sqlite:` 前缀时一律 `{DSN: path, IsSQLite: true}`（`InMemory=false`）。
- 后果：`InitLogDB` 的 pin 连接分支（`model/main.go:296-301`）不生效，默认
  `SQL_MAX_LIFETIME=60s` 会回收连接。对 `sqlite::memory:` 这种"每连接一个私有库"的 DSN，
  多连接下 schema/数据随连接消失（表现为 `no such table: logs`）；对
  `sqlite:file::memory:...` 同样会丢库。
- 判定依据：SQLite 语义（`:memory:` 每连接独立）+ 代码自身注释
  （"InMemory databases exist only while at least one pooled connection is open"）。
- 修法：新增 `isInMemorySQLiteDSN`；`sqlite:` 前缀下的内存路径标记 `InMemory=true`
  **保留用户 DSN 原样**（不重写、不丢 `_pragma` 等参数）。裸 `file::memory:?cache=shared`
  现在也识别为内存库（此前会落到 MySQL 分支）。
- 测试：`model/log_retention_test.go` — 表驱动新增 3 例；`TestInitLogDBWithInMemoryDSN`
  改为覆盖 `memory` 与 `sqlite::memory:` 两种写法（仍断言 pin 单连接 + 数据可读）。

### F-2b（已修）内存库负向对照测试依赖泄漏连接、顺序敏感

- 现象：加完 F-2 的测试清理（子测试结束 `sqlDB.Close()`）后，`make test` 里
  `model` 包失败：`TestSharedCacheMemoryDatabaseDisappearsWithoutPinnedConnection`
  在 `AutoMigrate` 处报 `no such table: main.logs`。
- 根因：该测试用**无名**的 `file::memory:?cache=shared`，与前面测试共享同一个进程级内存库；
  它此前"通过"是因为 `TestInitLogDBWithInMemoryDSN` 的池一直没关、替它保住了库。
  我把池关掉后，`MaxIdleConns(0)` 使每条语句后连接即被回收，库随之消失 → 建索引失败。
  （原断言用 `sqlDB.Close()` 后的同一池查询，实际报的是 "database is closed"，并不真正验证内存库消失。）
- 修法：改用**独立命名库** `file:zz_shared_memory_negative_control?mode=memory&cache=shared`，
  先建表插行 → `keeperSQL.Close()` 释放最后连接 → **另开池**查询并断言报错。
  现在它不依赖任何外部残留状态，真正验证"无连接即丢失 schema"。
- 验证：`go test ./model/ -count=1` 通过；`-shuffle=on` 连跑 3 次均通过（顺序无关）。

### F-1（未修·待确认）ClickHouse 日志库下 trim 无方言分支

- 证据：`DeleteOldLogBatch` 有显式 ClickHouse 分支（`ALTER TABLE logs DELETE ... SETTINGS mutations_sync = 1`），
  而新增的 `TrimLogToMaxRows` 直接 `Limit(limit).Delete(&Log{})`，无分支。
- 触发：`LOG_SQL_DSN=clickhouse...` 且 `LOG_MEMORY_MAX_ROWS>0`（或将来任何给 CH 设 cap 的路径）。
- 离线证据：假 ConnPool + DryRun 下 GORM 对 CH dialector 生成的删除形如 `DELETE WHERE created_at < ?`
  （无 `FROM`），目标 CH 版本是否接受无法离线证明 → 按 lead 要求记"待确认"，不臆断、不改 CH 路径。
- 建议二选一（交 Lead 决策）：① 给 trim 加 CH 分支复用 `DeleteOldLogBatch` 的 ALTER 写法；
  ② `logCleanupMaxRows()` 对 CH 返回 0 并 warn once（CH 不在内存约束内，避免清理任务必然失败）。

### F-3（已修·仅注释）`DELETE ... LIMIT` 只有 MySQL 生效

离线 DryRun（假 ConnPool，不建真实连接）实测四种方言对
`Where(...).Limit(100).Delete(&Log{})` 的渲染：

| 方言 | 生成的 SQL |
|------|-----------|
| MySQL | `DELETE FROM logs WHERE created_at < ? LIMIT 100` |
| PostgreSQL | `DELETE FROM "logs" WHERE created_at < $1`（LIMIT 被丢弃） |
| SQLite | `DELETE FROM logs WHERE created_at < ?`（LIMIT 被丢弃） |
| ClickHouse | `DELETE WHERE created_at < ?` |

影响：`limit` 在 SQLite/PG 上不生效，一次调用可能删除全部超限行；功能仍正确（调用方循环到 0），
但"每次只删一批"的写法不能当作控制单语句开销的依据——已更正 `TrimLogToMaxRows` 的文档注释。

### F-4（未修·待确认）multipart 图片编辑缺 `model` 必填校验

`relay/helper/valid_request.go:195-231` 的 multipart 分支取 `model` 后不校验空值，
而 JSON 分支 `:240-243` 会返回 `model is required`。两条路径行为不一致；补校验会改变
可见行为（原先可能到下游才报错），故只记录。

### F-5（未修·待确认）客户端 DTO 非指针标量 + `omitempty`

`relay/common/relay_info.go:849` `Duration int json:"duration,omitempty"` 为客户端解析后重发的 DTO，
违反 AGENTS.md"可选标量必须指针 + omitempty"；`dto/video.go:10-12`、部分 channel DTO 同类。
改指针会波及全部 task adaptor，超出低风险范围。**计费不变量未受影响**：
`relay/relay_task.go:121-127`（≤0→4，>3600 钳制）与 `relay/common/relay_utils.go:153` 已设上界。

### F-6（未修·低）心跳 goroutine 异常路径泄漏

`service/system_task.go` `runWithLeaseHeartbeat`：`close(done)` 在 `fn(ctx)` 之后而非 `defer`；
若 `fn` panic 且被上层 recover，则心跳 goroutine 与 `time.Ticker` 泄漏。影响极小，属既有代码。

## 正面结论（本轮抽查未发现问题）

- JSON 规则：`scripts/forbid-encoding-json.sh` 通过（无违规）。
- 行锁：全仓 `lockForUpdate(tx)` 用法一致，无 `gorm:query_option` 残留、无调用点重复 `clause.Locking`。
- 计费安全：`types.PriceData.AddOtherRatio` 拒绝非正/NaN/Inf；`MaxTaskDurationSeconds`、`dto.MaxImageN`
  在 task/图片入口设界；`relay/relay_task.go` 对历史任务时长做了钳制。
- 日志库分离：`model/log.go:519-556` 的 channel 名走主库、日志走 `LOG_DB`，未混用；
  `migrateLOGDB` 对 CH 单独分支；`InitLogDB` 空 DSN 时 `LOG_DB = DB` 与主库类型同步。
- 计费裸转换扫描：命中项均为图片尺寸/格式化/充值金额换算，不属 quota 换算路径。

## 红线事件自报（原文保留，供 Lead 汇总）

1. **ClickHouse dialector 自动 Ping**："为验证日志裁剪 SQL 的跨库行为，我在临时探针测试里用了 gorm 的
   ClickHouse dialector，其 Initialize 会自动 Ping，导致一次对 127.0.0.1:9000 的本地连接尝试
   （connection refused，无数据交换、无凭据）。已立即删除探针文件、不再重试、不改 CH 相关代码，
   并按红线要求如实上报。"
   此后所有跨库验证只用**假 ConnPool + DryRun**（不实例化真实 dialector）。
2. **临时日志文件**：`bash-747` 曾把验证输出重定向到 `/tmp/zz-verify.txt`（工作区外），
   发现后立即 `job_kill` 并删除该文件（仅 76 字节表头、无凭据），改写到
   `.trellis/tasks/10-06-team-review-backend/verify.log`；未重试越界写入。

## 实测结果（2026-10-06T12:26:51+00:00，含我的改动）

| 命令 | 退出码 |
|------|--------|
| `GOWORK=off go vet ./...` | 0 |
| `GOWORK=off go build ./...` | 0 |
| `cd relaykit && GOWORK=off go build ./...` | 0 |
| `make test` | 0（无 FAIL 行） |

基线（12:22:50，改动前）：vet / build / relaykit build 均为 0。
中间一次 `make test` 在 `model` 包失败，即 F-2b，修复后复测通过。
原始输出见同目录 `verify.log`。

## 改动文件

- `model/main.go`（`isInMemorySQLiteDSN` + `resolveLogSQLiteTarget`）
- `model/log.go`（`TrimLogToMaxRows` 文档注释）
- `model/log_retention_test.go`（新增/扩展测试；负向对照测试自包含化）

未提交（由 Lead 统一提交）。
