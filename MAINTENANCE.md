# MAINTENANCE.md — 维护手册

> 本仓库 `zhemed/new-api-own` 是 **公开** 仓库，基于 QuantumNous/new-api 代码基独立维护（非 fork 关系，历史含上游提交，但自 2026-08 起完全自维护）。

## 铁律（违反即事故）

1. **绝不同步上游**：未经仓库所有者明确许可，禁止对 QuantumNous/new-api（或任何上游）执行 fetch / merge / rebase / cherry-pick。上游改动一律不关注、不引入。
2. **不修改受保护标识**：new-api 与 QuantumNous 的名称、品牌、署名（README、许可头、包路径、Docker 镜像名、文档等）一律不得改动（见 `AGENTS.md` Project Governance）。
3. **行为不变原则**：清理代码（lint/format/重构）时不得改变任何用户可见行为；无法保证等价时，宁可用 `oxlint-disable` 注释，也不改行为。
4. **Docker 环境标准（AGENTS.md 强制）**：Docker Engine 29.7.2。一键安装：`curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash`。
5. **唯一部署方式（强制）**：只允许 `docker run` + 公开镜像 `ghcr.io/zhemed/new-api-own`（单容器、host 网络、挂 `./data:/data`）。**禁止** compose / Helm / K8s / 裸机 systemd unit 等第二种形态；`scripts/forbid-extra-deploy-methods.sh` 在提交与 CI 上拦截。规则、do-not-restore 清单与"回滚后必须核验"要求见 `.trellis/spec/guides/deployment-single-method.md`。
6. **提交前必须通过质量门禁**（见下）。注意：PR 质量门禁工作流（`ci.yml` / `pr-check.yml`）已于 2026-10-06 移除，现在**只能靠本地自查**（`go vet` / `go build` / `make test` / 前端 `bun run typecheck`）。

## 项目是什么

自用 LLM 网关与 AI 资产管理平台（Go + React）：多模型聚合、Key 管理、计费与分发、API 格式转换（OpenAI ⇄ Claude、OpenAI → Gemini）、授权登录、智能路由、思考模式支持。
- 后端：Go 1.25.1、Gin、GORM v2；前端：React 19 + TS + Rsbuild（`web/`，包管理用 Bun）
- 数据库：SQLite / MySQL / PostgreSQL 三库兼容；缓存：Redis + 内存
- 独立模块：`relaykit/`（不得依赖根模块，改动后必须 `GOWORK=off` 单独构建）
- 部署：唯一方式 `docker run` + 公开镜像 `ghcr.io/zhemed/new-api-own`（无需登录，直接拉取），单容器 + host 网络；见 `.trellis/spec/guides/deployment-single-method.md`

## 本机开发环境

| 组件 | 位置/命令 |
|---|---|
| Go | `/usr/local/go/bin/go`（实测 1.26.6），需 `export HOME=/root PATH=$PATH:/usr/local/go/bin GOPATH=/root/go GOMODCACHE=/root/go/pkg/mod GOCACHE=/root/.cache/go-build` |
| Bun | **当前这台机器（本机信息已脱敏）未安装**（`~/.bun` 不存在）→ 前端 `bun run typecheck` / `bun test` 要换到有 bun 的机器上执行（PR 质量门禁工作流已移除，**没有 CI 可兜底**）；历史上本机路径为 `~/.bun/bin/bun`（1.3.14），执行前需 `export HOME=/root` |
| GitHub 公开仓库 | 克隆无需凭据（`git clone https://github.com/zhemed/new-api-own.git`）；推送用 gh 的凭据助手且**只对本次命令生效**：`git -c credential.helper='!gh auth git-credential' push origin main`（`gh` 已登录 `zhemed`）。不要设置全局 `credential.helper`、也不要改 `git config`（AGENTS.md 明确禁止）；旧记录提到的本机凭据文件在本机**不存在** |

## 维护者接入（其他人 clone 之后如何开始）

仓库自带全部工作流产物：`.trellis/`（流程、规范、归档任务、journal）、`.dsh/skills/` 与 `.agents/skills/`（Trellis 技能）。**每台机器只需初始化一次开发者身份**（2026-09-18 在全新 clone 上实测通过）：

```bash
npm i -g @mindfoldhq/trellis          # 技能与脚本的运行器（本机 0.6.17）
git clone https://github.com/zhemed/new-api-own.git
cd new-api-own
trellis init --dsh -u <你的开发名> -s -y   # -s 跳过已存在文件；-y 跳过模板选择，不会覆盖仓库内容
```

在已有 `.trellis/` 的仓库上，`trellis init` 会：

- 写入本机身份 `.trellis/.developer`（被 `.trellis/.gitignore` 忽略，不进仓库）
- 创建 `.trellis/workspace/<开发名>/` 个人 journal
- 自动创建 `00-join-<开发名>` 接入任务（in_progress），由 AI 会话带着读完项目上下文
- 更新 `.trellis/.template-hashes.json`（模板追踪文件，属于正常现象；可提交，也可 `git checkout -- .trellis/.template-hashes.json` 还原）

之后每次工作都走同一套任务流程（dsh 里直接让 agent 加载 `trellis-start` 技能，或手工执行）：

```bash
python3 .trellis/scripts/task.py create "<标题>" --slug <slug> -d "<描述>"
python3 .trellis/scripts/task.py start <MM-DD-slug>        # → in_progress，先写 prd.md
# … 实现 → 质量校验（go vet / go build / make test；前端 bun run typecheck / bun test）…
python3 .trellis/scripts/task.py finish
python3 .trellis/scripts/task.py archive <MM-DD-slug> --skip-branch-validation
python3 .trellis/scripts/add_session.py --title "<本次标题>" --commit <sha>
```

随仓库分发 vs 只在本机：

| 随仓库分发（提交进 git） | 只在本机（`.trellis/.gitignore`） |
|---|---|
| `.trellis/spec/` 规范、`.trellis/workflow.md`、`.trellis/config.yaml`、`.trellis/scripts/` | `.trellis/.developer` 开发者身份 |
| `.trellis/tasks/archive/` 历史任务（含 PRD 与验收结论） | `.trellis/.current-task` 当前任务指针 |
| `.trellis/workspace/<开发名>/` 各人 journal | `.trellis/.runtime/`、`__pycache__/` 等运行时文件 |
| `.dsh/skills/`、`.agents/skills/` 技能，`AGENTS.md` 内的 Trellis 托管块 | |

CI 需要的一次性仓库设置（仓库所有者操作）：

- 允许 Actions 运行；现存两个工作流各自声明最小 `permissions:`：`docker-build.yml` 用 `packages: write` 推 GHCR、`id-token: write` 给 cosign 无密钥签名；`trellis-gate.yml` 只读 `contents: read`。二进制 Release 工作流已移除，**不再需要 `contents: write`**
- GHCR 包 `new-api-own` 需与本仓库关联，或配置 `GHCR_TOKEN` secret（带 `write:packages`），否则 tag 构建推送镜像会 403；v0.0.2 已用内置 `GITHUB_TOKEN` 推送成功，说明当前关联有效
- 无需 GitCode 同步相关配置（该工作流已移除）

发版流程与提交闸门见本文后续小节。

## 流程闸门（Trellis 强制规则）

调用 Trellis 不是口头承诺：**任何会话工作，包括只读调查（看代码、查日志、读库定位原因），开工第一步都必须先建 Trellis 任务**。唯一例外是用户明确说"这次跳过 Trellis"。

| 层 | 实现 | 拦什么 | 绕过 |
|---|---|---|---|
| ① 当场拦 | `.githooks/pre-commit`、`.githooks/commit-msg`（`core.hooksPath=.githooks`） | 暂存区含**非 `.trellis/`** 改动时：没有 `status=in_progress` 的任务 → 拒绝；消息里没有 `[task:<slug>]` 或 slug 不存在 → 拒绝 | `git commit --no-verify`（git 内建，封不死） |
| ② 事后审计 | `./scripts/check-trellis-gate.sh` | 逐个提交核对（跳过 merge 与纯 `.trellis/` 提交）：改动非 `.trellis/` 就必须带 `[task:…]` | 无法绕过：跑一次就暴露 |
| ③ 远程兜底 | `.github/workflows/trellis-gate.yml`（push 到 main / 开 PR） | 同一次审计，在 GitHub 上直接标红 | 无法绕过（删工作流是可见动作） |

**每个新克隆要跑一次**（`core.hooksPath` 是本地配置，不随仓库分发）：

```bash
./scripts/install-git-hooks.sh    # 装闸门
./scripts/check-trellis-gate.sh   # 自检：闸门已装 + 提交可追溯
```

提交消息统一带任务锚点（slug = `.trellis/tasks/<MM-DD>-<slug>` 去掉日期前缀）：

```
feat: 一句话说明 [task:commit-gate]
```

刻意放行两类：**纯 `.trellis/` 改动**（journal、任务归档、闸门自身）与 **merge 提交**。

审计起点记录在 `.trellis/gates/enforce-from`（启用闸门那一刻的提交 SHA），早于它的历史提交（上游与自维护基线）不在审计范围内。

> **诚实边界**：机器能强制的只是"提交时必须有任务"。"只读调查也先建任务"没有可审计的产物，靠 `AGENTS.md` 强制段 + journal 留痕，**不是自动的**，别把它说成自动生效。

规则细节与自证清单见 `.trellis/spec/guides/trellis-gate-guide.md`。

## 构建与测试

```bash
export HOME=/root PATH=$PATH:/usr/local/go/bin GOPATH=/root/go GOMODCACHE=/root/go/pkg/mod GOCACHE=/root/.cache/go-build

# 前端（web/）
cd web
bun install --frozen-lockfile
bun run typecheck      # tsgo -b，必须 0 错误
bun run lint           # oxlint，必须 0 error（warning 可按范围评估）
bun run build          # 产物 web/dist（后端 embed 依赖它！）
bun run format:check   # oxfmt 检查
bun run copyright:check

# 后端（仓库根）
go build -o /tmp/new-api-own-bin .    # 需要 web/dist 已存在
go test ./...                          # 全量测试

# relaykit 独立模块
cd relaykit && GOWORK=off go build ./...
```

> 注意：后端 `main.go` 用 `//go:embed web/dist` 内嵌前端，**先构建前端再构建后端**。

### 前端工具链（2026-10-06 起本机可用）

```bash
cd web && bun install --frozen-lockfile   # 需要 bun；本机 1.4.2
bun run typecheck   # tsgo -b；当前 0 错误（退出码 0）
bun run lint        # oxlint；当前 0 error / 21 warning（退出码 0）
bun test            # node:test（DOM 用 happy-dom，footer 用例改用 jsdom）；2026-10-06 复核 167 pass / 0 fail（34 个文件）
bun run build       # rsbuild，漏删引用会在此报错——删代码后必须跑
bun run knip        # 死码分析；入口已在 knip.config.ts 声明。2026-10-06 复核：unused files = 0（`--include files` 退出 0），全量仍以 1 退出：311 unused exports / 99 unused exported types / 6+2 unused deps / 1 duplicate export / 2 configuration hints（基线告警，非本轮引入）
```

## 本地部署验证

```bash
export HOME=/root
SQLITE_PATH=/tmp/own.db SESSION_SECRET=<随机串> PORT=3020 /tmp/new-api-own-bin &
# 注册：POST /api/user/register  {"username":"admin","password":"...","email":"..."}
# 登录：POST /api/user/login → data.access_token（JWT，Authorization: Bearer）
# 日志：GET  /api/log/self?type=2
# 账户用量（CC Switch 契约）：GET /api/usage/account/ 需 sk- relay token（创建接口不返回 key，需从 DB tokens 表读）
```

自定义功能相关环境变量：
- `LOGIN_SESSION_NEVER_EXPIRES=true` — Dashboard 会话永不过期（会话 `expires_at=0` 为哨兵值）

## 弱盘机器：把日志放到内存（或独立盘）

用途：硬盘慢/寿命敏感时，把**用量日志**（面板里那张 logs 表）从数据盘挪走。
**业务数据（账号/token/渠道/任务）永远留在主库，绝不进内存。**

`LOG_SQL_DSN` 除原有取值（空=跟随主库 / `local` / MySQL / PostgreSQL / ClickHouse）外，新增两种**日志专用**形态：

| 取值 | 含义 | 重启后 |
|---|---|---|
| `memory`（或 `:memory:`）| 日志放进**内存库**（`file::memory:?cache=shared`）| 日志清空；主库不受影响 |
| `sqlite:<路径>[?参数]` | 日志写**独立 SQLite 文件**（可指向 tmpfs/另一块盘）| 文件在哪就在哪 |

例：

```bash
# 完全进内存（单实例、日志只用于排障）
LOG_SQL_DSN=memory

# 独立文件 + WAL（放在内存盘上，等同内存但保留文件语义）
LOG_SQL_DSN='sqlite:/dev/shm/newapi-logs.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)'
```

配套自动裁剪（内存模式下**默认开启**，见下），否则内存会随日志无限增长：

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `LOG_MEMORY_MAX_BYTES` | 未设=关 | **载荷字节预算**（写入触发，见下节）；支持 `200MB` / `512KB` / `209715200` |
| `LOG_CLEANUP_INTERVAL` | `0`（关）；内存模式 `5m`；**配了字节预算时 `0`** | 清理周期（Go duration，如 `10m`、`1h`）；显式设置始终生效 |
| `LOG_CLEANUP_RETENTION_DAYS` | `7` | 按时间删；`0` = 不按时间删；定时与写入触发两条路径都会执行 |
| `LOG_MEMORY_MAX_ROWS` | 内存模式 `200000`；否则 `0` | 行数上限；**定时与写入触发两条路径共用同一上限**（`model.LogRowCap()`）|

行为与注意事项：

- 内存模式下启动会打两条 WARN（日志重启即失 + 是否设了行数上限），`LOG_MEMORY_MAX_ROWS=0`
  时额外提醒有 OOM 风险；
- 内存库**必须钉住一条连接**才不会被回收：`InitLogDB` 在内存模式下强制
  `MaxOpenConns=1 / MaxIdleConns=1 / ConnMaxLifetime=0`（默认的 60 秒生命周期会让日志表在运行中消失）；
- 清理任务走既有 system task 调度器，多主节点下由 DB 租约去重，不会重复执行；
- `LOG_SQL_DSN` 只作用于日志库；`SQL_DSN=memory` 不成立（主库不得进内存）；
- 消费日志带 `quota`（计费口径）：**依赖日志对账就别用内存模式**，改用 `sqlite:` 指到另一块盘。

### 体积上限（写入触发）

行数是"条数"，体积才是真正决定内存的数字：同样 20 万行，短日志几十 MB、长 prompt/响应能到 GB。
`LOG_MEMORY_MAX_BYTES` 直接给**载荷字节**设预算（支持 `200MB` / `512KB` / 裸字节数 `209715200`，1024 进制）：

- **写入触发**：每写成功一条日志累加一次（O(1)、不做 IO、不阻塞请求）；超预算就**异步、单飞**裁最老的记录，
  直到回到预算内（`model/log_budget.go:172-182` `noteLogPayloadBytes`、`:198-221` `triggerLogPayloadTrim`）；
- **不再依赖定时器**：配了字节预算，`LOG_CLEANUP_INTERVAL` 默认变 `0`（关闭）；**显式设置仍生效**
  （`service/system_task.go:129-145` `logCleanupInterval`）；
- 写入触发那一遍的**固定顺序**：**按天保留**（`LOG_CLEANUP_RETENTION_DAYS`，默认 7）→ **行数上限**
  （`LOG_MEMORY_MAX_ROWS`）→ **按体积裁最老** → 重算计数（`model/log_budget.go:234-252` `trimLogsToPayloadBudget`，
  保留在 `:257`、行数在 `:277-290`、体积在 `:293-327`）；
- **行数上限与字节预算是同遍生效的次级约束**：上限策略只有**一处事实来源** `model.LogRowCap()`
  （`model/log_budget.go:60-81`：显式值优先 → 内存模式默认 `200000` → 磁盘不设上限），
  **定时清理与写入触发共用**（`service/system_task.go:151-155` 直接委托）——**关掉定时器不影响它**；
- **表不会被清空**：单行就超预算时保留最新一行（`model/log_budget.go:303`，`rows <= 1` 即停）；
- **没配字节预算 = 与以前完全一致**（内存模式 5 分钟 + 20 万行）：未配时写入路径直接返回
  （`model/log_budget.go:172-177`）。

**留余量**：预算是**载荷估算**（文本字段 `len` 之和 + 每行固定开销 128B），**不含** SQLite 页、索引、WAL
与 Go 对象头（`model/log_budget.go:38`、`:101-104`）。所以别把"进程内存上限"直接当预算，
**按目标占用的 70–80% 设**（想控制在 ~250MB 就写 `200MB`）。

**什么时候还要用保留天数**：合规留存、按天对账、只想看最近 N 天——体积没到预算但时间太久的日志照样该消失。

**怎么确认定时清理真的没在跑**：

- 未显式设置 `LOG_CLEANUP_INTERVAL` 时，清理任务 `Enabled()` 直接为 false（`logCleanupInterval() == 0`），
  调度器不会创建该任务 → `GET /api/system-task/list` 里不会周期性出现 log cleanup 记录；
- 想手动跑一次：`POST /api/system-task/log-cleanup`（手动任务只带保留天数，不带行数上限——
  行数上限与体积本来就是写入触发那一遍在管）；
- 把 `LOG_CLEANUP_INTERVAL` 设回去（如 `10m`），定时任务恢复；它与写入触发共用同一套上限，不会互相冲突。

**面板核对**：日志文件接口返回 `memory_log_bytes`（当前估算）/ `memory_log_max_bytes`（预算）/
`memory_log_rows`（行数），与裁剪用的是同一个计数器与同一估算口径（`controller/performance.go:196-202`）。

**不适用**：ClickHouse 日志库——`LOG_MEMORY_MAX_BYTES` 与 `LOG_MEMORY_MAX_ROWS` 都会被忽略并各打一次 WARN，
改用保留天数（`model/log_budget.go:85-99`、`:60-81`）。

## 弱盘机器：减少 fsync（不改内存也能立竿见影）

用量日志是**每行一次事务**，SQLite 默认 `journal_mode=delete` + `synchronous=FULL`，
即每次写日志都 fsync。实测（本机探针）：

| `SQLITE_PATH` 形态 | journal_mode | synchronous |
|---|---|---|
| 默认 `one-api.db?_busy_timeout=30000` | delete | 2 (FULL) |
| 追加 `&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)` | wal | 1 (NORMAL) |

另注：默认 DSN 里的 `_busy_timeout=30000` **不生效**（读回 5000=驱动默认），
要写 `_pragma=busy_timeout(30000)` 才生效。

## 自定义功能线（与上游不同之处）

自 2026-08 起 zhemed 自维护，主要工作：

### 1. Reasoning effort（推理消耗）追踪
- 后端：`relay/common/relay_info.go`（`ReasoningEffort` 字段）、`relay/claude_handler.go`（Claude 持久化）、`relay/channel/openai/relay_responses.go` + `relay/helper/stream_scanner.go`（Responses 流扫描）、`relay/channel/openai/adaptor.go`（请求级 effort 记录）
- 落库：`service/log_info_generate.go` 写日志 `other.reasoning_effort`
- 前端：`web/src/features/usage-logs/components/columns/common-logs-columns.tsx`（列 + 开关）、`details-dialog.tsx`（badge）

### 2. DeepSeek V4 thinking effort 后缀（本仓库独有）
- `7dfdc6bf`：`-max` / `-none` 后缀支持，覆盖所有 relay 渠道
- `b9b39534`：无后缀 deepseek-v4 调用默认记录 effort=high
- 涉及 `setting/reasoning/suffix.go`（`ParseOpenAIReasoningEffortFromModelSuffix`）与各渠道转换

### 3. Dashboard 会话永不过期
- `common/session_expiry.go`（`LoginSessionNeverExpires` + `IsLoginSessionExpired` 哨兵）、`service/auth_session.go`、`model/user_session.go`、`common/init.go`（env 绑定）

### 4. CC Switch 用量导入
- `web/src/lib/cc-switch-import.ts` + `web/src/lib/__tests__/cc-switch-import.test.ts`、`cc-switch-dialog.tsx`
- `edcd6a5e`（8-13）：已移除硬编码的实例域名（已脱敏），改用当前站点 origin（http→https 提升）

### 5. 其他
- 模型倍率全精度（`800c26d6`）、用量日志表格对齐、去上游化（README 双语、链接/检查器/i18n 指向自维护仓库）、Docker 环境标准锁定

## 已知不一致（刻意未改，2026-10-06 团队审查留痕）

审查发现但**决定不改**的项目，改它们会变更用户可见行为或波及面过大；留在此处以免后人重复踩：

| 项 | 位置 | 为什么没改 |
|---|---|---|
| 客户端请求 DTO 用非指针标量（两处行为不同）| `relay/common/relay_info.go:849`：`Duration int` + `json:"duration,omitempty"`；`dto/video.go:7`：`Duration float64` + `json:"duration"`（**无** `omitempty`）| 与 AGENTS.md 的 DTO 指针规则不符，但改指针会波及全部 task adaptor；计费不变量已另有钳制：`relay/relay_task.go:125-126`（`seconds > MaxTaskDurationSeconds` 即钳到上界，注释在 `:124`）、`relay/common/relay_utils.go:148-165` `validateTaskDurationBounds`（同时约束 `Duration` 与 `Seconds`）|
| 日志裁剪的 `DELETE ... LIMIT` 仅 MySQL 生效 | `model/log.go:TrimLogToMaxRows` | GORM 的 SQLite/PG 方言会丢 LIMIT；已在注释写明"不可依赖 limit 限制单次工作量" |
| `count_token_failed` 走 500（客户端可控）| `controller/relay.go:154` | **刻意不改**：该分支混装"客户端上传损坏（应 4xx）"与"token 计数内部失败（应 5xx，`service/token_counter.go:193-209`）"，无法在映射层可靠区分；错判成 400 会掩盖真实服务端故障。留待按错误类型细分后再改 |
| `get_channel_failed`（重试取渠道）走 500 | `controller/relay.go:320,323` | **刻意不改**：容量/配置类故障，5xx 触发重试是期望行为（与 `middleware/distributor.go` 的 503/404 语义确有差异，已留痕待统一）|
| ClickHouse 日志库不支持行数上限 | **事实来源 `model.LogRowCap()`（`model/log_budget.go:60-81`）**；`service/system_task.go:151-155` 只是委托 | 不做 CH 分支（无法离线验证 CH 版本行为）：CH 时返回 0 并打一次性 WARN（`model/log_budget.go:64-71`），改用保留天数；字节预算对 CH 同样关闭（`model/log_budget.go:85-97`）|
| 前端孤儿模块 | `web/src/**`、`web/knip.config.ts` | 2026-10-06 已装 bun 并跑 knip：删掉 33 个确证无人引用的文件（`208483c`），复核阶段追加删 1 个（`web/src/hooks/use-mobile.tsx`）。**仍登记为 `ignore` 的只有产品决策类**：`src/components/ui/**`（shadcn 底座）、`src/components/ai-elements/**`（成套组件库）——**不要据 ignore 删依赖**（会误报 recharts / tokenlens / @xyflow/react 等，见 config 注释）。两条**已移除**的过时 ignore：`src/routeTree.gen.ts`（由 `src/main.tsx` 可达）、`src/i18n/static-keys.ts`（已接进 `scripts/check-i18n-keys.mjs` → `i18n:check` 入口，config 注释写明"不要再加回来"）。实测 `bun run knip --include files` **无未使用文件输出** |

## 团队审查已处理项（2026-10-06 留痕）

> 与上面的「已知不一致」相对：这些是审查中发现并**已修复**的问题，单独记，留痕以免后人重复排查。
> 修改都必须能过本地自查（PR 质量门禁已移除，详见下节）。

| 项 | 位置 | 处理结论 |
|---|---|---|
| multipart 图片编辑缺 `model` 必填校验（曾与 JSON 分支不一致）| `relay/helper/valid_request.go` | **已修复（`7e017cd`）**：multipart 分支现在与 JSON 分支一致——`:196` 取表单 `model`，`:200-202` 为空即 `return nil, errors.New("model is required")`（注释 `:197-199` 说明空 model 会被写进上游 form 或让渠道回落默认模型）；JSON 分支同一校验在 `:246-249`。验证方式：`grep -n 'model is required' relay/helper/valid_request.go` 应看到两条分支各一处；回归测试 `relay/helper/openai_image_request_test.go:80`（`...MultipartModelRequired`，断言在 `:120`）|
| 3 个 api-key group 表格测试失败（既有）| `web/src/features/keys/components/__tests__/api-key-group-cell.test.tsx` | **已修复（2026-10-06 团队）**：判定为"临时禁用、应恢复"而非"设计退役"——同款动效在 `api-key-group-combobox.tsx:111-179` 活跃使用且测试通过、动效 CSS（含 `prefers-reduced-motion`）在 `src/styles/index.css:655-700` 完整保留、`AutoGroupBadge` 唯一引用就是那行注释；已取消注释恢复功能（**未放宽任何断言**），`bun test` 由 151 pass / 3 fail 变为 **154 pass / 0 fail**（2026-10-06 复核为 167 pass / 0 fail）|
| footer XSS 测试曾经永远失败 | `web/src/components/layout/components/__tests__/footer.test.tsx` | **已修复**：原用 happy-dom，`purify.isSupported` 虽为 true 但消毒不完整（实测 `<script>` 原样保留），两个用例长期失败=从未真正验证消毒（复核员在基线 `0941a4f` 上原样复跑：1 pass / 2 fail）。已改用 jsdom（断言未改）并加 `isSupported` 前置断言；同时给 `footer.tsx` 加失效安全（不支持时不注入原始 HTML）|

## 质量门禁现状（2026-08-18 基线）

> **2026-10-06 变更**：PR 质量门禁工作流（`ci.yml` / `pr-check.yml`）已移除，本节基线**没有任何 CI 兜底**。
> 现在是**本地自查为唯一手段**：后端 `go vet` / `go build` / `make test`（含 `relaykit` 独立构建），
> 前端 `bun run typecheck` / `bun run lint` / `bun test`。`docker-build.yml` 只在推 tag 时构建镜像，
> 它**不跑测试**，构建成功 ≠ 校验通过。

- 后端：全量 Go 测试通过；relaykit 独立构建通过
- 前端：typecheck 0 错误；lint 清理进行中（上游遗留 ~386 错误，自动修复已清 250+，其余分批处理中）；format/copyright 干净
- 部署：SQLite 模式启动、注册/登录/日志/账户用量 API、reasoning_effort 端到端链路均已实测

## 自定义功能线审查结论（2026-08-18 深度审查）

### 已修复
1. 🟡 **A1（b9b39534 核心功能缺陷）**：无后缀 deepseek-v4 chat 调用默认 high 不同步 `info.ReasoningEffort` → 用量日志不显示。已在 `applyDeepSeekV4OpenAIDefaultEffort` 同步，并补 5 个测试（`relay/common/deepseek_v4_thinking_test.go`）。
2. 🟡 **B4**：chat→responses 转换丢失 `thinking.type=disabled` → 显式关闭思考被强制 high。已在 `relaykit/relayconvert/internal/oai_chat/to_oai_responses_req.go` 保留 disabled（映射 `effort=none`）。
3. 🟢 **C3**：会话列表对永不过期会话显示 1970-01-01 → 已改显示「Never/永不过期」（i18n 7 语言齐全）。
4. 🟢 **E5（部分）**：`cc-switch-import.test.ts` 已迁至 `web/src/lib/__tests__/`（符合 web/AGENTS.md 目录约定）。

### 已处理（第二批，2026-08-18 全面处理）
- **A2**：`relay/compatible_handler.go` 转换前统一把客户端显式 `reasoning_effort` 同步到 info（覆盖 deepseek/newapi 透传、OpenRouter 清空、chat→Claude、Gemini 等全部 OpenAI 兼容渠道，后缀派生值优先覆盖）。
- **A3**：`-none` 三路径统一记录 "none"（Claude/chat 路径在 `deepseek_v4_thinking.go` 补 disabled 分支；Responses 路径原有）。
- **B1**：`-max/-none` 挂到公共路径——chat 在 `compatible_handler.go`、Responses 在 `responses_handler.go` 对所有非 Gemini/Anthropic 的 OpenAI 兼容渠道生效（deepseek/newapi 原有调用幂等）；Gemini 渠道在 `ApplyThinkingConfig` 新增 DeepSeek V4 分支（`-none` → ThinkingBudget=0、`-max` → ThinkingLevel=high，修复原 TrimEffortSuffix 把 "max" 当 Gemini level 的静默丢语义）。
- **B2**：8 个倍率/限流查询函数（GetModelPrice/GetModelRatio/GetCompletionRatio/GetCompletionRatioInfo/GetAudioRatio/GetAudioCompletionRatio/ContainsAudioRatio/ContainsAudioCompletionRatio）先查原样名（带后缀独立定价条目直接命中），miss 再归一化。
- **B5**：已核对上游——`max` 是 DeepSeek V4 官方支持的 effort 值（vLLM PR #40982、DeepSeek API Thinking Mode 文档、litellm #27439），保持透传，无需映射。
- **C1/C2**：新增 `USER_SESSION_NEVER_EXPIRE_IDLE_DAYS` env（默认 0=不启用，行为不变）；启用后清理任务对 `expires_at=0` 且空闲超阈值的会话按分页批量删除（`model/user_session.go` 的 `deleteIdleNeverExpireUserSessions`），防止永不过期会话永久占满活跃上限。
- **D2**：CC Switch 导入/连接信息复制在 HTTP 地址时提示（warning，不阻止）。
- **D3**：`.github/ISSUE_TEMPLATE/*` 的上游文档链接已替换/移除。
- **E5**：session_expiry / auth_session_policy / adaptor_reasoning 三个测试文件已迁移 testify。

### 其余已知考量
- **C1（10 年 cookie 被浏览器截断为 ~400 天）**：服务端会话仍在，仅需重登，无功能错误。
- **模型后缀派生 effort（gpt-5-high 等）在 Gemini/Claude 渠道的 Responses 转换不解析**：🟢 轻微。

## 模型定价换算速查（防再踩坑）

`defaultModelRatio` 的单位是 quota/token，UI 显示美元/百万 = ratio × 2。两种价格书写模式：

- **美元模型**：`ratio = 美元/M × 0.5`（例：deepseek-chat 旧价 $0.27/M → `0.27 / 2`）
- **人民币模型**：`ratio = 元/千 tokens × RMB`（RMB = USD/7.3 = 68.49；例：ERNIE `0.12 * RMB` = 0.12 元/千 tokens）
- **⚠️ 错误教训（720a96fb）**：人民币价格写成 `元/M × RMB` 会**高估 1000 倍**（把 1.5 元/M 当成 1.5 元/K）——人民币必须先除以 1000 再乘 RMB。

deepseek-v4 已配置（2026-08-17 官方**美元**定价，英文站价格表；取谷价，高峰 2 倍）：
- `ModelRatio`: flash `0.22/2`（0.11）、pro `0.66/2`（0.33）——官方 off-peak 输入 $0.22/$0.66 每 M
- `CompletionRatio`: 3（官方输出 = 输入 × 3：$0.66/$1.98）
- `CacheRatio`: flash `0.007/0.22`、pro `0.022/0.66`（官方缓存命中 $0.007/$0.022）
- UI 校验：flash 输入 $0.22/M、补全 $0.66/M、缓存 $0.007/M ✅（官方英文站价，峰值 UTC 01:00-04:00/06:00-10:00 为 2 倍）
- 注意：官方中文站是人民币价（1.5/4.5 元），英文站是美元价（$0.22/$0.66），两者汇率口径略有差异（≈6.8）；本仓库按美元价配置

## OpenCode Go 会话头（`x-opencode-session`）

OpenCode Go（`https://opencode.ai/zen/go`）自 2026-09-05 起要求每个请求携带会话标识，缺失时上游返回：

```
400 MissingSessionID: Error from provider (Console Go): Request is missing
x-opencode-session and cannot be routed efficiently
```

该头不是鉴权，而是**上游 GPU 提示缓存的路由亲和键**：同一对话的多轮请求带同一个 ID 才能复用显存里的 KV 缓存。

### 两个容易混淆的字段

| 字段 | 结构 | 说明 |
|---|---|---|
| `header_override` | 扁平 map `{"头名": "字符串值"}` | 值为非字符串会报 `ChannelHeaderOverrideInvalid`；键 `*` / `re:<regex>` / `regex:<regex>` 是透传规则（值被忽略） |
| `param_override` | `{"operations": [...]}` | 通用操作流，按数组顺序执行（`pass_headers` / `set_header` / `copy_header` …） |

**不要混用**：把 operations 数组写进 `header_override` 会因值不是字符串而校验失败；`header_override` 的占位符语法在 `param_override` 里也不生效。

### 推荐配置（`header_override`，单条搞定）

```json
{
  "x-opencode-session": "{client_header:x-opencode-session|opencode-go-fallback}"
}
```

语义：客户端带了 `x-opencode-session` 就用客户端的（**每对话独立，缓存亲和完整**）；没带则填 `opencode-go-fallback`（避免 400，代价是该类客户端**共用一个缓存桶**）。

### 等价配置（`param_override` operations）

面板「参数覆盖」内置预设 **OpenCode Go Session Header** 可直接套用，展开为：

```json
{"operations":[
  {"mode":"pass_headers","value":["x-opencode-session","Session-Id","X-Session-Id"],"keep_origin":true},
  {"mode":"set_header","path":"x-opencode-session","value":"opencode-go-fallback","keep_origin":true}
]}
```

> ⚠️ **顺序不可颠倒**：`pass_headers` 必须在前，`set_header`（`keep_origin: true`）在后。
> 写反了兜底值会压过客户端值，退化成「所有对话共用一个会话 ID」——功能不报错，但缓存亲和全丢。
> 该 operations 形式在「测试渠道」时同样生效；`header_override` 的**无兜底**占位符在测试渠道时会被跳过（测试请求没有真实客户端头），带 `|DEFAULT` 的形态不受影响。

### 上游行为实测（2026-09-12，直连上游对照实验）

1. 上游**不挑头名**：`x-opencode-session`、`Session-Id`、`X-Session-Id`、`Session_id` 均返回 200；`Thread-Id` 不认。
2. 只给 `claude-cli` / `codex_cli_rs` 的 `User-Agent` **不能**替代会话头，仍 400。
3. **`x-opencode-session` 参与上游缓存键，`Session-Id` 不参与**：固定长前缀下换 `Session-Id` 仍命中缓存（4608 tokens），换 `x-opencode-session` 则命中为 0。走原生 `Session-Id` 的客户端只是「不报错」，缓存仍是共享的。
4. 两个头同时存在时 `x-opencode-session` 优先。
5. 缓存读取价约为输入价的 1/10，所以命中与否则是实打实的成本差异。

### 诊断手法（可复用）

要确认网关究竟发出了哪些头，**架本地探针比翻二进制快得多**：起一个打印请求头的临时 HTTP 服务，把渠道 `base_url` 临时指向它，发一次请求后读探针日志，最后还原 `base_url`。本次结论 5 即由此得出（探针只看到 `User-Agent: Go-http-client/1.1` + `Authorization` + `Content-Type`）。

> 注意：通过 `ssh 'bash -s' < script` 跑远端脚本时，不需要 stdin 的 `docker run` 必须加 `< /dev/null`，否则容器会吞掉脚本剩余内容，表现为执行中途静默截断。

## 自用部署注意事项

- 本 fork **默认关闭四组限流**（`common/init.go` 默认值即为 false，`.env.example` 已列出）：`GLOBAL_WEB_RATE_LIMIT_ENABLE`、`GLOBAL_API_RATE_LIMIT_ENABLE`、`CRITICAL_RATE_LIMIT_ENABLE`（登录/注册/重置密码/2FA/OAuth 等敏感操作）、`SEARCH_RATE_LIMIT_ENABLE`（搜索接口按用户限流）。这是**有意的自用配置**（内网信任环境、方便频繁操作），不是缺陷
- 关闭后全局爆破式请求没有兜底：若仓库公开或对外提供服务，需把对应 `*_ENABLE` 设回 `true`（`*_RATE_LIMIT` 次数与 `*_DURATION` 秒数原值仍在，设置即可恢复，见 `.env.example` 的「限流配置」段）
- 公开镜像可直接拉取，无需 `docker login`
- 部署前确保 Docker 为标准版本（29.7.2 + v5.4.0）

## 发版流程（版本号第三位递增：0.0.2 → 0.0.3 → 0.0.4 …）

1. 更新 `VERSION`（与即将打的 tag 一致，**不带** `v`），提交到 `main`
2. 打注释 tag 并推送（`<版本>` 为刚提交的版本号，如 `0.0.6`）：

   ```bash
   git tag -a v<版本> -m "v<版本>"
   git push origin main v<版本>
   ```

3. 推 tag **只自动触发一个工作流**（触发面已刻意收敛，避免每次发版扇出）：

   - `Publish Docker image (Multi-arch)` → 构建并推送 `ghcr.io/zhemed/new-api-own:v<版本>`、
     `ghcr.io/zhemed/new-api-own:<版本>`（去掉 `v` 的等值别名）与 `:latest`，多架构清单 + cosign 签名

   **交付只有镜像这一条路径**：二进制 Release（`release.yml`）、Electron 桌面壳（`electron-build.yml`）、
   GitCode 同步（`sync-release-to-gitcode.yml`）与手动分支镜像（`docker-image-branch.yml`）
   均已移除（2026-10-06 用户定调：部署就是 `docker run` + 公开镜像，其余产物一律不要）。

4. 校验（**只产出镜像，不再有 GitHub Release 产物**）：

   ```bash
   docker run --rm ghcr.io/zhemed/new-api-own:<版本> --version          # 应输出 v<版本>
   docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:<版本>    # 应看到 amd64/arm64 清单
   ```

   发布后在自己的部署机上按「版本与升级」再对一次（运行实例 / registry / 仓库三处一致）。

5. 镜像内的版本号来自构建时的 tag：`Dockerfile` 把 `VERSION` 注入 Go ldflags（`common.Version`）与前端 `VITE_REACT_APP_VERSION`，而 CI 会用 tag 覆写 `VERSION` 文件内容，所以**必须走 tag 发版**；只改文件不推 tag 不会触发镜像构建，镜像里会停在旧值（本项目自 2026-10-06 起只产出镜像，不再有 GitHub Release）。
6. 版本注入的 `-ldflags -X` 必须写**完整模块路径** `github.com/QuantumNous/new-api/common.Version`；写成简写（`new-api/common.Version`）会被 Go 静默忽略，镜像内版本会停在内置默认值 `v0.0.0`（`Dockerfile` 用的是完整路径，勿改）。

> 约定：`VERSION` 文件不带 `v`，tag 带 `v`，两者版本号一致；镜像同时提供 `v<版本>` 与 `<版本>` 两种拉取标签，指向同一份多架构清单。

## 版本与升级（每次维护必做）

> **一条命令版**：`make check-version INSTANCE=http://127.0.0.1:3000`（脚本 `scripts/check-version-drift.sh`）。
> 它比对四处——① 仓库 `VERSION`、② 最新 git tag、③ registry `latest` / `<版本>` / `v<版本>` 三个标签的
> 镜像摘要（三者必须同摘要）、④ 运行实例 `/api/status` 的 version；退出码 **0=一致 / 1=不一致 /
> 2=无法判定**，**查不到绝不报"一致"**。手抄命令与判据见下「升级后三处比对」；
> **三处不一致就是事故**，维护/发版/升级后都要跑一遍。
>
> **事故记录（2026-10-06）**：团队三轮全在仓库内部干活，没人核对线上版本，结果运行实例停在 `0.0.5`
> 而 `latest` 早已是 `0.0.6`；同日本机缓存的 `latest` 标签还曾陈旧指向 `0.0.3`（现已刷新）。
> 教训：**任何机器上的本地标签都可能陈旧**，`docker run …:latest` 会复用本地旧标签（不会自动拉新）。

### 升级 / 回滚（唯一部署方式下）

**关键认知：重启容器不会换镜像。** `docker restart new-api`、`--restart always` 触发的自动重启、
甚至重启宿主机，都只是把**同一个镜像**再跑一遍——版本当然不变。升级必须**删掉容器、用新镜像重建**。

升级三步（`<版本>` 换成目标 tag，如 `0.0.6`；**在原来的部署目录执行**，`./data` 是相对路径，换目录＝换了数据目录）：

```bash
docker pull ghcr.io/zhemed/new-api-own:<版本>     # 1. 先拉新镜像
docker rm -f new-api                              # 2. 删旧容器（数据在 ./data，不受影响）
docker run -d --name new-api --restart always \
  --network host \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:<版本>               # 3. 用新镜像重建
```

回滚同理，只换 tag（registry 保留每个历史 tag：`0.0.5`、`0.0.4`…）：

```bash
docker pull ghcr.io/zhemed/new-api-own:0.0.5
docker rm -f new-api
docker run -d --name new-api --restart always \
  --network host \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:0.0.5
```

> **回滚注意**：上面四条 `LOG_*` 需要支持内存日志的版本（`LOG_SQL_DSN=memory` 自 `v0.0.4` 起，
> `LOG_MEMORY_MAX_BYTES` 自 `v0.0.8` 起）。回滚到更早版本时，请把该版本不认识的变量去掉再重建。


**为什么必须显式 `docker pull`、为什么不建议用 `:latest`**：`:latest` 是**本地标签**，
`docker run …:latest` 不会去 registry 比对更新——本地有就直接用。本机 2026-10-06 曾把 `latest` 停在
`0.0.3`（当日已刷新，现与 `0.0.6` 同一摘要）；**这不是一次性事故：任何机器的本地标签都可能陈旧**，
此时跑 `:latest` 得到的就是旧版。所以升级用**固定 tag**；非要用 `:latest` 时，先
`docker pull ghcr.io/zhemed/new-api-own:latest`，再按下节核对摘要。

> 升级 / 回滚**仍然只有 `docker run` 这一种形态**：不要为此引入 compose / Helm / K8s / systemd
> （`.githooks/pre-commit` 与 `scripts/forbid-extra-deploy-methods.sh` 会拦，规则见
> `.trellis/spec/guides/deployment-single-method.md`）。

### 升级后三处比对（可执行）

三处必须是同一个版本，任一处落后都说明"以为升级了，其实没有"（一条命令版见上文「版本与升级」引言）：

| # | 查哪里 | 命令 | 期望 |
|---|---|---|---|
| 1 | **运行实例**（唯一真相）| `curl -s http://127.0.0.1:3000/api/status \| grep -o '"version":"[^"]*"'` | `"version":"v0.0.6"` |
| 2 | **registry 清单摘要** | `docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:0.0.6 --format '{{.Manifest.Digest}}'` | 与 `:latest`、与本地镜像摘要一致 |
| 3 | **仓库源头** | `cat VERSION`；`git tag --sort=-v:refname \| head -1` | `0.0.6`；`v0.0.6`（VERSION 不带 v，tag 带 v）|

> 等价写法：第 1 行也可用 `curl -s <实例>/api/status | jq -r .data.version`，或直接看**面板页脚**的版本号；
> 第 3 行也可用 `git tag | sort -V | tail -1`。

判据：把三处版本号**去掉前缀 `v` 后**逐一比较，全部相等才算对齐。某处落后时的动作：

- **第 1 处落后** → 容器还在跑旧镜像：按上节「升级 / 回滚」的三步重建。**重启不算升级。**
- **第 2 处落后** → 镜像没发出去或没发完：查 `Publish Docker image (Multi-arch)` 的运行结果。
- **第 3 处落后** → 源头没递增：先改 `VERSION` 并提交，再打 tag（见「发版流程」）。

### 坑：本地 `:latest` 标签会陈旧

`docker run …:latest` **不是**"取远端最新"，而是"用本地 `latest` 标签指的那份镜像；本地没有才去拉"。
旧标签不会自动刷新，于是出现"升级了却还是旧版"（本机 2026-10-06 的 `latest` 就停在 `0.0.3`，当日才刷新；
**换台机器同样会发生**）。两条命令自查：

```bash
docker images ghcr.io/zhemed/new-api-own --digests                                                   # 本地：能用到什么
docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:latest --format '{{.Manifest.Digest}}'  # 远端：最新是什么
```

两条摘要不一致 = 本地标签陈旧 → `docker pull ghcr.io/zhemed/new-api-own:latest` 后重建容器，或改用固定 tag。

> ⚠️ **别用 `docker images | grep 0.0.5` 判断"本机有没有某个版本"**：不带仓库名会命中**别的项目**的同号 tag，
> 而且 `grep` 里 `.` 是通配符（`0.0.5` 也会匹配 `…09035…` 这类无关行）。过滤一律带完整仓库名
> `ghcr.io/zhemed/new-api-own`；要 grep 就写 `grep 'ghcr\.io/zhemed/new-api-own'`。

镜像自带版本标签，**不启动容器**也能确认某份镜像的版本：

```bash
docker image inspect ghcr.io/zhemed/new-api-own:0.0.6 \
  --format '{{index .Config.Labels "org.opencontainers.image.version"}}'    # → v0.0.6
```

## 面板内更新（self-update）

> **先说定位**：面板内更新只用于**临时跟上版本**，**不是第二套部署方式**——唯一部署方式仍然是
> `docker run` + 公开镜像（见上节「版本与升级」）。**长期升级/回滚请走镜像三步**。

### 它是什么

1. 面板 **系统设置 → 系统维护 → 检查更新** 显示**当前版本 / 最新版本 / 是否有更新**
   （三态：有更新 / 已最新 / 无法确定，**不会把"查不到"说成"已最新"**）。
   发起检查的是**服务端**（外呼更新源），**浏览器不直连外部 API**，请求不带任何凭据。
2. 有新版本时，管理员点 **「立即更新」**：后端按**本机架构**取对应发布资产
   （`new-api-linux-amd64` / `new-api-linux-arm64`）→ 下载到临时文件 →
   用随发布提供的 `SHA256SUMS` 校验 → **只有校验通过才原子替换**（新文件 rename 到位）→
   **原地重执行**进程（`exec` 不可用时退出进程，由 `--restart always` 拉起），面板随之重连。
3. **安全不变量：校验失败绝不替换。** 摘要不符就丢弃临时文件并报错，正在运行的二进制保持不动。
   其它护栏：只接受**比当前更新**的版本（不回退）、仅支持 `amd64`/`arm64`、同一时刻只允许一个更新任务、
   下载体积有上限、请求不带任何凭据。

> **前置条件**：自更新由**运行中的二进制自身**提供。面板里出现「立即更新」按钮，说明该构建内置更新器；
> 没有这个按钮的旧构建**没有**该能力——请先按上节「升级 / 回滚」的镜像三步升级到含更新器的版本。
> 另外，更新源上对应版本必须带 `new-api-linux-amd64` / `new-api-linux-arm64` 与 `SHA256SUMS` 三个资产
> （由发布流水线产出）；缺资产时更新会明确报错，**不会**替换。

### 代价（必须知道）

- 容器内被替换的二进制**不在镜像里**：下一次 `docker rm` + `docker run`（换机器、重拉镜像、重建容器）
  会**退回镜像内版本**——"升级当次看着成功、重建后又变回旧版"就是这么来的；
- 因此它只适合**临时跟上版本**：要长期停在某个版本，仍然固定镜像 tag（见上节）；
- 它**不碰数据目录**（`./data`），升级前后数据不受影响。

### 开关与网络

| 变量 | 默认 | 作用 |
|---|---|---|
| `UPDATE_CHECK_ENABLED` | `false`（**默认关闭**：未显式开启时服务端不发起任何外呼）| 打开"检查更新"（面板查询/自动检查）|
| `UPDATE_APPLY_ENABLED` | `true`（**默认允许**，仅管理员可触发）| 设为 `false` 可**整体关闭**「立即更新」|
| `UPDATE_CHECK_REPOSITORY` | 本仓库 | 更新源仓库（`owner/repo`），可换自建镜像源 |
| `UPDATE_CHECK_API_BASE_URL` | `https://api.github.com` | 更新源 API 基址，**可指向镜像/自建代理**（受限网络用）|
| `UPDATE_CHECK_PROXY_URL` | 空 | 可选代理出口；不设置时沿用 Go 默认（`HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY`）|

两个开关注册进了全局配置管理器，**在面板里改完即时生效、无需重启**（随时可关掉外呼/自更新）。

大陆网络建议：给容器加 `-e HTTPS_PROXY=http://<代理>:<端口>`（或 `UPDATE_CHECK_PROXY_URL`），
必要时把 `UPDATE_CHECK_API_BASE_URL` 指向可达的镜像/代理。检查与更新都发生在**服务端**，
浏览器不需要能访问 GitHub。

### 排障

```bash
# 1) 运行实例自报的版本（唯一真相）；更新成功后这里应当变化
curl -s http://127.0.0.1:3000/api/status | grep -o '"version":"[^"]*"'

# 2) 容器内二进制的摘要（与发布附带的 SHA256SUMS 对照；镜像内二进制路径为 /new-api）
docker exec new-api sha256sum /new-api

# 3) 回退：删掉容器用镜像重建，立即回到镜像内版本（数据在 ./data 不受影响）
docker rm -f new-api
docker run -d --name new-api --restart always \
  --network host \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v ./data:/data \
  -e LOG_SQL_DSN=memory \
  -e LOG_MEMORY_MAX_BYTES=200MB \
  -e LOG_MEMORY_MAX_ROWS=200000 \
  -e LOG_CLEANUP_RETENTION_DAYS=7 \
  ghcr.io/zhemed/new-api-own:<版本>
```

- 更新后 `/api/status` 的 `version` **没变** → 多半是下载或校验失败（代理不通、更新源不可达、摘要不符）：
  此时**二进制没有被替换**，实例仍按原版本运行；
- 摘要不符**不是 bug**，是预期内的安全行为（绝不替换）；确认资产与 `SHA256SUMS` 同源后再重试；
- 更新源不可达时面板显示"无法确定"，**不会谎报"有更新"**；网络修好后重试即可；
- 三处版本比对（实例 / registry / 仓库）见上节「版本与升级（每次维护必做）」，本节不重复。

### 与「版本与升级」的关系

| 场景 | 走哪条路 |
|---|---|
| 让**长期运行**的实例稳定在新版本 | **镜像三步**（上节「升级 / 回滚」）|
| 临时跟上版本、不想重建容器 | 面板「立即更新」（本节）——注意上面的代价 |
| 更新后版本不对 / 想退回 | 先按本节「排障」确认状态，再按镜像三步回滚 |

## 部署安全基线（必读）

> 来源：2026-09-12 对线上实例（（实例域名已脱敏））的实测排查。**仓库当前配置默认不满足其中数项**，对外部署前请逐条确认。

### 1. 面板端口默认暴露在公网 ⚠️

`docker run` 部署命令里的 `--network host` 让 NewAPI 自身直接监听 `*:3000`（所有网卡，含公网 IP）。

**关键认知：反向代理（lucky / nginx / caddy）只是"额外开一个入口"，不会关闭这个直连端口。** 反代到 `127.0.0.1:3000` 与 `3000` 是否对外可达，是两件互相独立的事。

实测证据（**从另一台机器**访问，不是服务器自测）：

```
http://<公网IP>:3000/   ->  HTTP 200      # 面板直连可达
GET /api/status          ->  HTTP 200      # 无需登录即可读出大量配置
```

若主机又没有入站防火墙（nftables 里只有 Docker 的 FORWARD 链、没有 INPUT 链），入站流量将全部放行。

加固二选一：

```bash
# 方案 A：保留 host 网络，用防火墙限制来源
nft add table ip filter
nft add chain ip filter INPUT '{ type filter hook input priority filter; policy accept; }'
nft add rule ip filter INPUT iif lo accept
nft add rule ip filter INPUT tcp dport 3000 drop      # 或改成白名单 accept

# 方案 B（推荐）：只绑本地，交给反代转发
# 去掉 --network host，改用端口映射：
#   -p 127.0.0.1:3000:3000
# 注意：若另起了 Postgres / Redis，它们要么各自限制监听地址，要么一并改为容器网络
```

### 2. 数据目录含明文密钥，必须收紧权限

`data/one-api.db`（SQLite 模式）中**上游渠道 API Key 是明文存储**的，`users` / `tokens` 表还含可用凭据；`data/` 下的备份文件同样带密钥。默认权限下同机任何用户都可读取。

```bash
chmod 700 data data/logs data/backup 2>/dev/null
chmod 600 data/*.db data/logs/* data/backup/* 2>/dev/null
```

> 真正的防线是**目录 700**：即使应用后续新建的日志文件又是 644，非 root 也无法遍历进入该目录。

### 3. 容器以 root 运行

`Dockerfile` 未设置 `USER`，容器内进程为 `uid=0`，会放大上面「数据权限」与「挂载目录」两项的影响面。如需收紧，可在运行阶段创建非 root 用户并保证 `/data` 属主匹配。

### 4. 默认口令必须替换

唯一部署方式默认走 SQLite（`./data`），不含 Postgres / Redis。若你另外起 MySQL / PostgreSQL / Redis 并接上（`SQL_DSN`、`REDIS_CONN_STRING`），**必须替换它们的默认口令**，不要沿用示例值。

### 5. 部署自检清单

```bash
# 端口暴露面（应只看到预期开放的口）
ss -tulnp | grep -E ':(3000|6379|5432)'

# 是否存在入站防火墙
nft list ruleset | grep -q 'hook input' || echo '⚠️ 无 INPUT 链，入站全放行'

# 数据文件权限（应为 700 / 600）
stat -c '%A %n' data data/*.db 2>/dev/null
```

## 工作流

1. 改代码 → 涉及文件 lint 0 error + typecheck 通过 → 相关 Go 测试/前端测试
   （**PR 质量门禁已移除：本地自查是唯一手段**，推 tag 只构建镜像、不跑测试）
2. 行为变更必须补回归测试（后端 `testify`；前端放 `__tests__/` 目录）
3. 提交信息用项目风格（`fix:` / `feat:` / `chore:` 前缀，描述变更与原因）
4. 提交后推送：`export HOME=/root && git push origin main`（公开仓库，推送需凭据，细节见上文「本机开发环境」）
