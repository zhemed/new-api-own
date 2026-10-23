# 实例数据目录迁移证据：/root/data → /root/new-api-own/data

- 任务：Trellis `10-23-move-instance-data-into-workspace`；共享任务板 `task-8`（执行者 worker-containers）
- 执行时间：2026-10-23T16:58:50Z ～ 2026-10-23T17:00:28Z（UTC），全程只访问本机
- 事前依据：PRD `prd.md:5-9`（用户指令「迁移到我们的工作区」）、`prd.md:27-41`（R1-R7）
- 写作用域：本文件 + `/root/new-api-own/data`（迁移落点）
- 红线遵守：未删 `/root/data`；未升级镜像；未动其它容器/镜像/卷；未提交、未推送；未改仓库其它文件
- 环境：Docker 29.7.2 + Compose v5.4.0（达标）

## 1. 前后对照表（容器配置逐项）

| 参数项 | 迁移前 | 迁移后 | 判定 |
|---|---|---|---|
| 容器名 | new-api | new-api | 一致 |
| 镜像 | ghcr.io/zhemed/new-api-own:v0.0.45 | ghcr.io/zhemed/new-api-own:v0.0.45 | 一致（未升级） |
| Entrypoint | ["/new-api"] | ["/new-api"] | 一致 |
| Cmd | null | null | 一致 |
| WorkingDir | /data | /data | 一致 |
| NetworkMode | host | host | 一致 |
| RestartPolicy | always / MaximumRetryCount=0 | always / MaximumRetryCount=0 | 一致 |
| LogConfig Type | json-file | json-file | 一致 |
| LogConfig max-size | 10m | 10m | 一致 |
| LogConfig max-file | 3 | 3 | 一致 |
| Env LOG_SQL_DSN | memory | memory | 一致 |
| Env LOG_MEMORY_MAX_BYTES | 200MB | 200MB | 一致 |
| Env LOG_MEMORY_MAX_ROWS | 200000 | 200000 | 一致 |
| Env LOG_CLEANUP_RETENTION_DAYS | 7 | 7 | 一致 |
| Mount Source | /root/data | /root/new-api-own/data | **预期变化（本次目标）** |
| Mount Destination | /data | /data | 一致 |
| Mount 类型/RW/传播 | bind / RW=true / rprivate | bind / RW=true / rprivate | 一致 |
| RestartCount | 0 | 0 | 一致 |
| State | Up（迁移前）→ 已停 → 旧容器删除 | running | 达标 |

只有 bind 的宿主侧路径按计划改变，其余逐项一致。

## 2. 业务数据前后对照

| 指标（只读 sqlite，uri mode=ro） | 迁移前（/root/data/one-api.db） | 迁移后（/root/new-api-own/data/one-api.db） | 判定 |
|---|---|---|---|
| users 行数 | 1 | 1 | 一致 |
| tokens 行数 | 1 | 1 | 一致 |
| options 总行数 | 14 | 14 | 一致 |
| options 定价相关行数（key like ratio 或 pricing） | 5 | 5 | 一致 |
| 定价相关 key 列表 | AudioCompletionRatio, AudioRatio, CacheRatio, CompletionRatio, ModelRatio | 同左（逐项一致） | 一致 |

口径说明：`options` 表存在 `key` 列（先 `select key from options limit 5` 确认，样本为
AudioCompletionRatio / AudioRatio / CacheRatio / CompletionRatio / DemoSiteEnabled），
故按 task-8 指定口径 `key like '%ratio%' or key like '%pricing%'` 取数，另附 `options` 总行数作交叉核对。

## 3. 数据完整性校验（复制后、删旧容器前）

| 校验项 | 结果 |
|---|---|
| `diff -r /root/data /root/new-api-own/data` | 无差异，exit=0 |
| 文件数 | 源 53 / 目标 53 |
| 目录数 | 源 2 / 目标 2 |
| 文件总字节（复制时） | 源 7097601 / 目标 7097601 |
| `one-api.db` sha256 | 两侧同为 55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283 |
| 全部 53 个文件逐个 sha256 清单比对 | 完全一致（manifest 逐行相同） |
| 属主/权限/路径逐个比对 | 完全一致（`%M %u:%g %p` 清单无差异） |

说明：第 1 步基线读到的文件总字节是 7097395，复制时为 7097601（+206 字节）——
差额来自停机前容器仍在写 `logs/`；`one-api.db` 的 sha256 在停机前后未变。
判定以「停机后 源 与 目标 的对照」为准，两者一致。

## 4. 完整重建命令（实际执行）

```console
$ docker stop new-api
new-api
$ docker rm new-api
new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.45
f3084bb008e27cae4f26df612a6a12893be0b443afad2ca650dc0c0debed529f
```

新容器 ID：`f3084bb008e27cae4f26df612a6a12893be0b443afad2ca650dc0c0debed529f`
（与 README 的 `-v ./data:/data` 等价，此处用绝对路径，避免 cwd 依赖）。

## 5. 回滚命令（未触发，R3 要求写进证据）

触发条件：第 3 节校验失败、或启动失败、或版本/计数对不上。本次**未触发**（全部通过）。

```console
$ docker rm -f new-api
$ docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 -v /root/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.45
$ curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:3000/api/status
200
```

回滚前提与代价（诚实标注）：`/root/data` 自 16:58:36 起已被冻结、内容完整（53 文件，
`one-api.db` sha256 仍为 551941...），可直接回滚；但回滚后**迁移窗口内写入工作区那份库的增量会丢失**
（本例仅启动期的会话/日志类写入，业务计数未变）。

## 6. 原始输出（逐步骤）

### 6.1 STEP1 迁移前基线（环境与容器）

```console
### 2026-10-23T16:58:50Z STEP1 baseline
--- docker ps -a (new-api) ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up 2 hours | 
--- inspect: image/entrypoint/cmd/workdir ---
Image=ghcr.io/zhemed/new-api-own:v0.0.45
Entrypoint=["/new-api"]
Cmd=null
WorkingDir=/data
--- inspect: network/restart/log/mounts/env ---
NetworkMode=host
RestartPolicy={"Name":"always","MaximumRetryCount":0}
LogConfig={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
Binds=["/root/data:/data"]
--- inspect: Env ---
LOG_SQL_DSN=memory
LOG_MEMORY_MAX_BYTES=200MB
LOG_MEMORY_MAX_ROWS=200000
LOG_CLEANUP_RETENTION_DAYS=7
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
--- inspect: Mounts ---
[{"Type":"bind","Source":"/root/data","Destination":"/data","Mode":"","RW":true,"Propagation":"rprivate"}]
--- inspect: RestartCount ---
0
```

### 6.2 STEP1 迁移前业务计数与源目录

```console
### 2026-10-23T16:58:58Z STEP1 business counts (read-only sqlite uri mode=ro)
--- counts ---
users                    = 1
tokens                   = 1
options total            = 14
options ratio/pricing    = 5
--- options ratio/pricing keys ---
   AudioCompletionRatio
   AudioRatio
   CacheRatio
   CompletionRatio
   ModelRatio
--- source dir ---
7.0M	/root/data
53 files / 2 dirs
55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283  /root/data/one-api.db
```

### 6.3 STEP2 停机

```console
### 2026-10-23T16:59:06Z STEP2 stop container
--- target dir pre-state (before mkdir) ---
ls: cannot access '/root/new-api-own/data': No such file or directory
--- docker stop new-api ---
new-api
stop_exit=0
--- confirm Exited ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Exited (0) Less than a second ago
--- inspect State ---
Status=exited ExitCode=0 FinishedAt=2026-10-23T16:59:06.210832634Z
```

### 6.4 STEP3+4 复制与校验

```console
### 2026-10-23T16:59:15Z STEP3 copy
mkdir -p /root/new-api-own/data && cp -a /root/data/. /root/new-api-own/data/
cp_exit=0

### 2026-10-23T16:59:15Z STEP4 verify copy
--- diff -r /root/data /root/new-api-own/data (empty = identical) ---
diff_exit=0
src_files=53  dst_files=53
src_dirs=2  dst_dirs=2
src_bytes=7097601  dst_bytes=7097601
55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283  /root/data/one-api.db
55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283  /root/new-api-own/data/one-api.db
MANIFEST_MATCH=yes (all files identical)
manifest lines: src=53 dst=53
META_MATCH=yes (mode/owner identical)
```

### 6.5 STEP5+6 删旧容器与重建

```console
### 2026-10-23T16:59:24Z STEP5 rm old container (no -v)
docker rm new-api
new-api
rm_exit=0
--- confirm gone ---
(empty above = removed)

### 2026-10-23T16:59:25Z STEP6 recreate with workspace data dir
docker run -d --name new-api --restart always --network host --log-opt max-size=10m --log-opt max-file=3 -v /root/new-api-own/data:/data -e LOG_SQL_DSN=memory -e LOG_MEMORY_MAX_BYTES=200MB -e LOG_MEMORY_MAX_ROWS=200000 -e LOG_CLEANUP_RETENTION_DAYS=7 ghcr.io/zhemed/new-api-own:v0.0.45
f3084bb008e27cae4f26df612a6a12893be0b443afad2ca650dc0c0debed529f
run_exit=0
```

### 6.6 STEP7 就绪与版本头

```console
### 2026-10-23T16:59:37Z STEP7 readiness poll (max 60s)
  t+1s http_code=200
final_http_code=200
--- curl -sD - /api/status (headers) ---
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-New-Api-Version: v0.0.45
X-Oneapi-Request-Id: 202610231659372388899928268d9d6z2GQi57x
Date: Fri, 23 Oct 2026 16:59:37 GMT
Transfer-Encoding: chunked

### 2026-10-23T17:00:28Z STEP7b filtered liveness evidence
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
--- body fields extracted ---
success = True
version = v0.0.45
setup   = True
```

说明：`/api/status` 响应体会带实例专属配置值，按红线第 3 条**不落盘**，只摘取
`success` / `version` / `setup` 三个判定字段与响应头。

### 6.7 STEP8 迁移后核验

```console
### 2026-10-23T16:59:53Z STEP8 post-migration verification
Image=ghcr.io/zhemed/new-api-own:v0.0.45
Entrypoint=["/new-api"]
Cmd=null
WorkingDir=/data
NetworkMode=host
RestartPolicy={"Name":"always","MaximumRetryCount":0}
LogConfig={"Type":"json-file","Config":{"max-file":"3","max-size":"10m"}}
Binds=["/root/new-api-own/data:/data"]
Env: LOG_SQL_DSN=memory / LOG_MEMORY_MAX_BYTES=200MB / LOG_MEMORY_MAX_ROWS=200000 / LOG_CLEANUP_RETENTION_DAYS=7
Mounts=[{"Type":"bind","Source":"/root/new-api-own/data","Destination":"/data","Mode":"","RW":true,"Propagation":"rprivate"}]
RestartCount=0 Status=running StartedAt=2026-10-23T16:59:25.107694354Z

--- business counts (post) ---
users                 = 1
tokens                = 1
options total         = 14
options ratio/pricing = 5
ratio/pricing keys    = ['AudioCompletionRatio', 'AudioRatio', 'CacheRatio', 'CompletionRatio', 'ModelRatio']

--- ls data / ls data/logs ---
-rw-r--r--  1 root root 1159168 one-api.db
-rw-r--r--  1 root root 1126400 one-api.db.bak-before-model-trim-20261022-153933
-rw-r--r--  1 root root 1044480 one-api.db.bak-sessions-20261018-143837
-rw-r--r--  1 root root 1044480 one-api.db.bak-sessions-20261019-104308
drwxr-xr-x  2 root root    4096 logs
dst_logs=50

--- old /root/data still intact (NOT deleted) ---
src_files=53  src_db_sha256=55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283
volume_count=40
image_present_v0045=1
```

### 6.8 STEP9 写入落点证明与 git 状态

```console
### 2026-10-23T17:00:10Z STEP9a write-location proof
container StartedAt = 2026-10-23T16:59:25.107694354Z
--- files under /root/data modified after container start (expect none) ---
(empty above = old path is frozen)
--- files under /root/new-api-own/data modified after container start ---
2026-10-23T16:59:25.1878729110       4096 /root/new-api-own/data/logs
2026-10-23T16:59:37.2416088970       2921 /root/new-api-own/data/logs/oneapi-20261023165925.log
2026-10-23T16:59:55.5106972850    1159168 /root/new-api-own/data/one-api.db
2026-10-23T16:59:55.5116973440       4096 /root/new-api-own/data
--- db size compare (src frozen vs dst live) ---
1155072 2026-10-23 16:58:36 +0000 /root/data/one-api.db
1159168 2026-10-23 16:59:55 +0000 /root/new-api-own/data/one-api.db
--- logs file count: src vs dst ---
src_logs=49  dst_logs=50

### 2026-10-23T17:00:10Z STEP9b git status
?? .trellis/tasks/10-23-move-instance-data-into-workspace/
--- lines mentioning data/ (expect none) ---
(none)
--- check-ignore ---
.gitignore:27:/data/	data/
```

这说明运行实例的写入**只落在工作区那份**：旧路径自停机后无任何文件被改动，
新路径出现新日志文件并持续更新 `one-api.db`；`data/` 被 `.gitignore:27` 忽略、未被跟踪。

## 7. 验收对照（PRD AC）

| AC | 内容 | 状态 | 证据位置 |
|---|---|---|---|
| AC1 | 迁移前后业务计数一致 | 达标 | 第 2 节对照表 + 6.2 / 6.7 |
| AC2 | 新挂载生效且其余参数逐项一致，restarts=0 | 达标 | 第 1 节对照表 + 6.7（RestartCount=0） |
| AC3 | `/api/status` 200 且 `X-New-Api-Version: v0.0.45` | 达标 | 6.6 |
| AC4 | `/root/data` 仍完整存在（回滚点保留） | 达标（未删） | 6.7 末段 |
| AC5 | `data/` 未被跟踪 | 达标 | 6.8（check-ignore 命中 `.gitignore:27`） |
| AC6 | 证据含前后对照、重建命令、回滚命令、验证输出 | 达标 | 第 1-6 节 |

## 8. 风险与未完成

- `/root/data` 与工作区 `data/` 现为两份：运行实例**只以工作区那份为准**（已由 6.8 证明），
  旧目录仅是冻结回滚点；删除需用户决定，本次**未删**。
- 迁移窗口内的写入只落在新库（启动期会话/日志类，业务计数未变）；若日后回滚，这部分增量会丢。
- 旧目录的 3 个历史备份与 `logs/` 已整份复制，属主/权限一致，无需额外处置。
- 未做（约束内）：未清悬空卷（卷总数仍 40）、未删镜像、未改任何业务配置。
- 提交留痕未做（红线）：本文件留在工作区，待 Lead 按 Trellis 闸门提交。

## 9. 建议下一步

1. Lead 复核本证据后决定 `/root/data` 的处置（建议再观察一轮，确认实例稳定运行后再删）。
2. 若要落实「唯一部署形态」文档一致性，可核对 README 是否以 `-v ./data:/data` 表述（本次未改 README）。
3. 提交时锚点用 `[task:move-instance-data-into-workspace]`，并把 `data/` 排除在提交范围外
   （已由 `.gitignore:27` 保证）。

> 说明：上面第 1-9 节记录的是**迁移当时**的状态（当时 `/root/data` 保留作回滚点）。
> 下面的第 10-11 节是随后 task-9 的两阶段动作，**已删除** `/root/data`；
> 第 7 节 AC4 与第 8 节相关描述以第 11 节为最新状态。

---

# 阶段 A/B（任务 task-9）：重启存活验证 + 删除冻结旧目录

- 共享任务板 `task-9`；执行者 worker-containers；时间 2026-10-23T17:03:44Z ～ 17:04:56Z（UTC）
- 事前依据：用户裁定「验证通过后立即删除 `/root/data`」（task-9 描述 + Lead 转述）
- 结果：**阶段 A 全绿（PHASE_A=GREEN）→ 阶段 B 全绿（PHASE_B=GREEN）**，旧目录已删，实例健康

## 10. 阶段 A：重启存活验证

目的：证明新挂载在**新容器的第二次生命周期**里仍稳定生效（迁移当时只验证过首次启动）。

### 10.1 重启前基线

```console
### 2026-10-23T17:03:44Z PHASE-A1 pre-restart baseline
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up 4 minutes
/root/new-api-own/data:/data restarts=0
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
StartedAt_before=2026-10-23T16:59:25.107694354Z
```

### 10.2 重启与就绪

```console
### 2026-10-23T17:03:44Z PHASE-A2 docker restart
docker restart new-api
new-api
restart_exit=0
StartedAt_after=2026-10-23T17:03:45.012487501Z
--- readiness poll (max 60s) ---
  t+1s http_code=000
  t+2s http_code=200
final_http_code=200
```

StartedAt 从 16:59:25 变为 17:03:45，证明这是一次真实重启（非空转）。

### 10.3 重启后核验

```console
### 2026-10-23T17:03:55Z PHASE-A3 post-restart verification
state=running
restarts=0
mount=/root/new-api-own/data:/data
image=ghcr.io/zhemed/new-api-own:v0.0.45
binds=["/root/new-api-own/data:/data"]
HTTP/1.1 200 OK
X-New-Api-Version: v0.0.45
success = True | version = v0.0.45
users                 = 1
tokens                = 1
options total         = 14
options ratio/pricing = 5
ratio/pricing keys    = ['AudioCompletionRatio', 'AudioRatio', 'CacheRatio', 'CompletionRatio', 'ModelRatio']
dst_logs=51   (重启后新建 oneapi-20261023170345.log)
```

核验对照（迁移后基线取自本文件第 2 节）：

| 核验项 | 期望 | 实测 | 判定 |
|---|---|---|---|
| State | running | running | 达标 |
| RestartCount | 0 | 0 | 达标 |
| Mounts | /root/new-api-own/data:/data | /root/new-api-own/data:/data | 达标 |
| HTTP | 200 | 200 | 达标 |
| X-New-Api-Version | v0.0.45 | v0.0.45 | 达标 |
| users / tokens / 定价 options | 1 / 1 / 5 | 1 / 1 / 5 | 达标 |

```console
=== PHASE A GATE RESULT ===
state=running (expect running) | restarts=0 (expect 0) | mount=/root/new-api-own/data:/data (expect /root/new-api-own/data:/data)
http=200 (expect 200) | version=v0.0.45 (expect v0.0.45) | users=1 tokens=1 pricing_options=5 (expect 1/1/5)
PHASE_A=GREEN
```

阶段 A 全绿 → 进入阶段 B（若任一不符则停手且不删旧目录，本次未触发）。

## 11. 阶段 B：删除冻结旧目录

### 11.1 删前安全校验（四条，逐条打印）

```console
### 2026-10-23T17:04:13Z PHASE-B5 pre-delete safety checks (each gated)
--- [B5.1] new instance health ---
  state=running restarts=0 mount=/root/new-api-own/data:/data
  http=200 version=v0.0.45  users=1 tokens=1 pricing_options=5
  B5.1=PASS
--- [B5.2] new dir exists and non-empty ---
  total 4284
  drwxr-xr-x  3 root root    4096 Oct 23 17:03 .
  drwxr-xr-x  2 root root    4096 Oct 23 17:03 logs
  -rw-r--r--  1 root root 1159168 Oct 23 17:03 one-api.db
  -rw-r--r--  1 root root 1126400 Oct 22 15:39 one-api.db.bak-before-model-trim-20261022-153933
  -rw-r--r--  1 root root 1044480 Oct 18 14:38 one-api.db.bak-sessions-20261018-143837
  -rw-r--r--  1 root root 1044480 Oct 19 10:43 one-api.db.bak-sessions-20261019-104308
  files=55 bytes=7109379
  b8093c6669922dd2c102ec1de8650271b7d615653f41c93a1aaf7eec91297832  /root/new-api-own/data/one-api.db
  B5.2=PASS
--- [B5.3] target path confirmation (literal, no wildcard) ---
  TARGET(raw)=/root/data
  realpath=/root/data
  is_dir=yes
  is_symlink=no
  workspace_dir_realpath=/root/new-api-own/data
  B5.3=PASS
--- [B5.4] frozen-source evidence (sha256 must match migration record) ---
  expect=55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283
  got   =55194152028e6b392adb76148423b59db06550487afc36f0254351e1121b5283
  files=53  bytes=7097601
  B5.4=PASS

PHASE_B_GATE=GREEN (all four checks passed; safe to delete /root/data)
```

校验要点：

- B5.3 的 `realpath /root/data` 严格等于 `/root/data`、且不是符号链接、父目录必须是 `/root`；
  工作区目录 `realpath` 为 `/root/new-api-own/data`，两者是不同 inode（见 11.2），排除同目录误判。
- B5.4 用**冻结副本**的 sha256 与迁移记录比对，逐位相同；这同时说明旧目录自停机后未被写入过。
- 新库 sha256（`b8093c66...`）与旧库不同属**预期**：新库是活库，实例持续写入；
  判定依据是「新库可读且计数正确」，不是「两侧 sha 相同」。

### 11.2 删除执行（字面绝对路径 + 即时复核守护）

```console
### 2026-10-23T17:04:46Z PHASE-B6 delete frozen /root/data
--- inode distinctness proof (target vs workspace data) ---
dev:inode=2050:13771618  /root/data
dev:inode=2050:13764445  /root/new-api-own/data
--- immediate pre-delete re-guard (literal path, no variable expansion in rm) ---
  guard: ALL PASS
  contents about to be deleted: files=53 dirs=2 bytes=7097601
--- executing: rm -rf -- /root/data ---
rm_exit=0
```

守护条件（任一不满足即 ABORT，什么都不删）：目标为目录、`realpath` 等于 `/root/data`、非符号链接、
工作区目录存在且库非空、旧库 sha256 等于冻结值、容器 running、容器挂载为工作区路径、实例返回 200。
删除命令是**字面量** `rm -rf -- /root/data`，未使用变量与通配符。

### 11.3 删后核验

```console
### 2026-10-23T17:04:56Z PHASE-B7 post-delete verification
$ ls -ld /root/data
ls: cannot access '/root/data': No such file or directory
  B7.1=PASS (target removed)
--- workspace data still present ---
  files=55  db_sha=5947ab10c191b33a2a419a07eb2753730b07cba3ae39ed67b5175d5df9263594
--- container health ---
new-api | ghcr.io/zhemed/new-api-own:v0.0.45 | Up About a minute
  state=running restarts=0 mount=/root/new-api-own/data:/data
  HTTP/1.1 200 OK
  X-New-Api-Version: v0.0.45
--- counts after delete ---
  users                 = 1
  tokens                = 1
  options ratio/pricing = 5
--- git status ---
?? .trellis/tasks/10-23-move-instance-data-into-workspace/
  data-mentioning lines: 0
--- other containers / volumes / images untouched ---
  new-api | Up About a minute ; komari-agent-test | Up 7 hours ; komari-test | Up 5 hours ;
  picsur | Up 3 days (healthy) ; support-devdb-1 | Up 3 days ; litepan | Up 3 days ;
  halo | Up 3 days (healthy) ; halodb | Up 3 days (healthy)
  volume_count=40  v0045_image=1
--- /root still exists (parent not damaged) ---
  drwx------ 35 root root 4096 Oct 23 17:04 /root

PHASE_B=GREEN (old dir deleted, instance healthy on workspace data)
```

### 11.4 阶段 B 结论

`/root/data` 已删除；工作区 `data/` 仍完整（55 文件，活库持续更新），容器 running、restarts=0、
挂载为 `/root/new-api-own/data:/data`、`/api/status` 200 且 `X-New-Api-Version: v0.0.45`、
业务计数 1 / 1 / 5 未变；`git status` 无 data 相关行；其它容器（8 个全 Up）、卷（40）、
镜像（v0.0.45 仍在）均未受影响。

新增风险提示：删除后**回滚点消失**——本文件第 5 节的回滚命令（`-v /root/data:/data`）已不再可用；
现行唯一数据源为工作区 `/root/new-api-own/data`，其备份依赖目录内的 3 个历史 `.bak` 文件。
