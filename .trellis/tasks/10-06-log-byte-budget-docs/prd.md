# 日志体积上限（写入触发）文档

- 任务：`10-06-log-byte-budget-docs`
- 执行：`review-ops`（写范围：`.env.example`、`MAINTENANCE.md`、`README.md`）
- 设计源：`.trellis/tasks/10-06-log-byte-budget-write-trigger/prd.md`

## 目标

把"日志按**体积**控上限、且**写入触发**（不再依赖 5 分钟定时器）"这套新机制写进：
`.env.example`（新变量 + 注释、旧变量注释同步真实语义）、
`MAINTENANCE.md`「弱盘机器：把日志放到内存（或独立盘）」（新增"体积上限（写入触发）"小节）、
`README.md`（部署段一句话入口）。

## 硬约束

- **等后端实现落地再定稿**：变量名、默认值、语义一律以**代码**为准（不按 Lead 信里的描述猜）。
- 不 commit/push、不改代码、不写主机名/IP/凭据。

## 设计要点（来自设计 PRD，待与代码核对）

| 环境变量 | 语义 | 默认 |
|---|---|---|
| `LOG_MEMORY_MAX_BYTES`（新增）| 日志**载荷字节**预算，接受 `200MB` / `209715200` | 未设=关闭 |
| `LOG_MEMORY_MAX_ROWS` | 行数上限（次级约束，与字节预算并存）| 内存模式 20 万 |
| `LOG_CLEANUP_INTERVAL` | 定时清理间隔 | **配置了字节预算时默认 0（关闭）**；显式设置仍生效 |
| `LOG_CLEANUP_RETENTION_DAYS` | 按天保留 | 7（在写入触发那一遍里执行）|

机制：进程内 atomic 字节计数（载荷估算 + 首次 `SUM(length(...))` 校准）→ 每次写日志累加（O(1)、不阻塞请求）
→ 超预算触发**异步单飞**裁剪（删最老，直到回到预算内）→ 同一遍顺带执行保留天数删除；
未配置字节预算时行为与现在**完全一致**（内存模式 5 分钟 + 20 万行）。

## 要写的内容（Lead 指定的四点）

1. 为什么**体积**比行数直观（行数 × 行长差异大，行数上限不代表内存上限）；
2. 字节是**载荷估算**、要留 20–30% 余量（SQLite 页/索引/Go 运行时对象头都另占空间）；
3. 何时仍要用**保留天数**（合规/对账/只想看最近 N 天，而不是"撑到预算"）；
4. 如何确认**真的没在跑定时清理**（`LOG_CLEANUP_INTERVAL` 默认 0 + 如何查看）。

## 验收

- 三个文件改动清单；`.env.example` 新变量与注释和代码默认值一致；
- 报告中给出**核对过的代码行号**（变量绑定、默认值、写入触发、保留天数同遍执行）；
- 未配置字节预算时的"行为不变"表述与代码一致。

## Notes

- 实现未落地前只做草案，不改仓库文件；落地后逐条对齐再写盘。

---

# 草案（待代码落地后核对定稿）

## A. `.env.example`（替换现 34–39 行那段）

```
# 日志清理周期（不设=不改行为：未配字节预算时内存模式默认 5m；**配置了 LOG_MEMORY_MAX_BYTES 时默认 0＝关闭定时器**）
# LOG_CLEANUP_INTERVAL=5m
# 按时间保留天数（0=不按时间删；默认 7）；配置字节预算后，这一遍随写入触发的裁剪一起执行
# LOG_CLEANUP_RETENTION_DAYS=7
# 日志载荷字节预算（**写入触发**：每写一条累加，超预算就异步裁最老的记录；支持 200MB / 209715200；0/不设=关闭）
# LOG_MEMORY_MAX_BYTES=200MB
# 日志总行数上限（**次级约束**，与字节预算同时生效；内存模式默认 200000）
# LOG_MEMORY_MAX_ROWS=200000
```

## B. `MAINTENANCE.md`「弱盘机器：把日志放到内存」新增小节

```
### 体积上限（写入触发）

行数是"条数"，体积才是真正决定内存的数字：同样是 20 万行，短日志几十 MB、长 prompt/响应可以上 GB。
`LOG_MEMORY_MAX_BYTES` 直接给**载荷字节**设预算：

- 写入触发：每写一条日志累加（O(1)，不阻塞请求）；超预算就**异步、单飞**裁最老的记录，直到回到预算内；
- **不再依赖 5 分钟定时器**：配了字节预算，`LOG_CLEANUP_INTERVAL` 默认变 `0`（关闭）；显式设置仍生效；
- `LOG_MEMORY_MAX_ROWS` 保留为**次级约束**（两个上限谁先到谁生效）；
- `LOG_CLEANUP_RETENTION_DAYS`（默认 7）仍在，由**写入触发那一遍顺带执行**；
- **没配字节预算 = 行为与以前完全一致**（内存模式 5 分钟 + 20 万行）。

写预算时要留余量：计数是**载荷估算**（各文本字段长度 + 固定开销），不含 SQLite 页/索引开销与
Go 运行时对象头，**建议按"想占的内存 × 70–80%"来设**（例如想控制在 250MB 左右就写 `200MB`）。

仍然要用保留天数的场景：合规留存、按天对账、只想看最近 N 天（体积没到预算但时间太久的日志也该消失）。

确认没有在跑定时清理：

```bash
# 配置项默认值（配了字节预算时应为 0）
#   LOG_CLEANUP_INTERVAL 未显式设置时默认 0（关闭定时器）
# 运行日志里不应再出现定时的清理任务；看 system task 列表/日志即可
```
```

## C. `README.md`（部署段一句话）

```
> 想按**体积**控内存：加 `-e LOG_MEMORY_MAX_BYTES=200MB`（写入触发裁剪，不再需要定时器）——详见
> [MAINTENANCE.md](./MAINTENANCE.md)「弱盘机器：把日志放到内存」。
```

---

# 定稿记录（2026-10-06，实现落地后按代码核对）

## 实现已落地（`model/log_budget.go`、`model/log.go`、`common/env.go`、`service/system_task.go`、`controller/performance.go`）

写盘前逐条核对（**行号已复核**，文件 md5 稳定）：

| 文档写的 | 代码依据（已核对） |
|---|---|
| 变量名 `LOG_MEMORY_MAX_BYTES`，支持 `200MB`/`512KB`/裸字节，1024 进制 | `model/log_budget.go:20`（常量）、`common/env.go` 的 `GetEnvOrDefaultSize` + `sizeUnitMultipliers`（`M`/`MB` 等价、大小写不敏感） |
| 未设/0 = 关闭；未配时写入路径直接返回、行为与以前一致 | `model/log_budget.go:54-68`、`:141-146` |
| 写入触发：写成功后累加（O(1) 不阻塞）| `model/log.go:103-107`（`createLog` → `noteLogPayloadBytes`）、`model/log_budget.go:141-153` |
| 异步 + 单飞 | `model/log_budget.go:167-189`（`triggerLogPayloadTrim`/`beginLogPayloadTrim` CAS） |
| 同一遍：先按天保留 → 再按体积裁最老 → 重算计数 | `model/log_budget.go:203-217`、`:221-240`、`:278-285` |
| 表不会被清空（单行超预算保留最新一行）| `model/log_budget.go:252`（`rows <= 1 → return`） |
| 配了字节预算 → `LOG_CLEANUP_INTERVAL` 默认 0；显式设置仍生效 | `service/system_task.go:129-145`（`logCleanupInterval`） |
| 保留天数默认 7，两条路径共用 | `model/log_budget.go:23-25`、`service/system_task.go:32-38` |
| 载荷口径 + 每行固定开销 128B，不含页/索引/WAL | `model/log_budget.go:29-31`、`:70-73` |
| 面板可看用量/预算/行数 | `controller/performance.go:196-202`（`memory_log_bytes` / `memory_log_max_bytes` / `memory_log_rows`） |
| ClickHouse 不支持（忽略 + WARN，改用保留天数）| `model/log_budget.go:49-68` |
| 定时任务可见性/手动触发 | `router/api-router.go:288-294`（`GET /api/system-task/list`、`POST /api/system-task/log-cleanup`，RootAuth）|

## 与 Lead 信里的一处**事实差异**（已按代码写，需你知悉）

Lead 说「`LOG_MEMORY_MAX_ROWS` 保留为**次级约束**」——代码里它的**执行路径只有定时清理**
（`service/system_task.go:117` 的 payload `MaxRows` → `:564` 的 `trimLogsToMaxRows`）；
**手动触发**（`StartLogCleanupTask`，`service/system_task.go:287-302`）构造 payload 时**不设 `MaxRows`**。
所以：**配了字节预算后，定时器默认关 → 行数上限默认不再自动执行**（要它生效必须显式设 `LOG_CLEANUP_INTERVAL`）。
文档已按此写明（`.env.example`、MAINTENANCE 表格与正文），没有写成"两个上限都在自动跑"。

## 交付

| 文件 | 改动 |
|---|---|
| `.env.example` | 新增 `LOG_MEMORY_MAX_BYTES`（含 `200MB` 写法、"设了它默认关掉定时器"）；同步 `LOG_CLEANUP_INTERVAL` / `LOG_CLEANUP_RETENTION_DAYS` / `LOG_MEMORY_MAX_ROWS` 三条注释到真实语义 |
| `MAINTENANCE.md` | 「弱盘机器：把日志放到内存」新增 `### 体积上限（写入触发）`；配套裁剪表格更新（新变量 + 新默认） |
| `README.md` | 部署段一句话入口（`-e LOG_MEMORY_MAX_BYTES=200MB`） |

校验：围栏 README 14 / MAINTENANCE 40（配平）；`.env.example` 无围栏；引用行号逐条复核（上表）；
未 commit/push、未改代码、无网络动作。


