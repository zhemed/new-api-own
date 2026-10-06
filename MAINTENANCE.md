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
| `LOG_CLEANUP_INTERVAL` | `0`（关）；内存模式 `5m` | 清理周期（Go duration，如 `10m`、`1h`）|
| `LOG_CLEANUP_RETENTION_DAYS` | `7` | 按时间删；`0` = 不按时间删，只按行数上限 |
| `LOG_MEMORY_MAX_ROWS` | 内存模式 `200000`；否则 `0` | 总行数上限，超出即从最旧开始裁剪 |

行为与注意事项：

- 内存模式下启动会打两条 WARN（日志重启即失 + 是否设了行数上限），`LOG_MEMORY_MAX_ROWS=0`
  时额外提醒有 OOM 风险；
- 内存库**必须钉住一条连接**才不会被回收：`InitLogDB` 在内存模式下强制
  `MaxOpenConns=1 / MaxIdleConns=1 / ConnMaxLifetime=0`（默认的 60 秒生命周期会让日志表在运行中消失）；
- 清理任务走既有 system task 调度器，多主节点下由 DB 租约去重，不会重复执行；
- `LOG_SQL_DSN` 只作用于日志库；`SQL_DSN=memory` 不成立（主库不得进内存）；
- 消费日志带 `quota`（计费口径）：**依赖日志对账就别用内存模式**，改用 `sqlite:` 指到另一块盘。

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
| multipart 图片编辑缺 `model` 必填校验（JSON 分支有）| `relay/helper/valid_request.go:195-231` vs `:240-243` | 补校验会让原本被接受的请求变 400，属上游遗留不一致，改=可见行为变更 |
| 客户端请求 DTO 用非指针标量 + `omitempty` | `relay/common/relay_info.go:849`（`Duration`）、`dto/video.go:10-12` | 与 AGENTS.md 的 DTO 指针规则不符，但改指针会波及全部 task adaptor；计费不变量已另有钳制（`relay/relay_task.go:121-127`、`relay/common/relay_utils.go:153`） |
| 日志裁剪的 `DELETE ... LIMIT` 仅 MySQL 生效 | `model/log.go:TrimLogToMaxRows` | GORM 的 SQLite/PG 方言会丢 LIMIT；已在注释写明"不可依赖 limit 限制单次工作量" |
| ClickHouse 日志库不支持行数上限 | `service/system_task.go:logCleanupMaxRows` | 不做 CH 分支（无法离线验证 CH 版本行为）；显式返回 0 并打一次性 WARN，改用保留天数 |
| 前端孤儿模块 | `web/src/**` | 2026-10-06 已装 bun 并跑 knip：删掉 33 个确证无人引用的文件；`ui/`、`ai-elements/`（成套组件库）与 `i18n/static-keys.ts`（i18n 工具链的键登记表）、`routeTree.gen.ts`（生成物）登记为 knip `ignore`，不删 |
| 3 个 api-key group 表格测试失败（既有）| `web/src/features/keys/components/__tests__/api-key-group-cell.test.tsx` | 组件里 `AutoGroupBadge` 与动效环**被注释掉**，测试仍断言旧设计；恢复功能属产品决策，改测试又会掩盖差异——保持失败并在此留痕 |
| footer XSS 测试曾经永远失败 | `web/src/components/layout/components/__tests__/footer.test.tsx` | 原用 happy-dom：`DOMPurify.isSupported=false`，`sanitize()` 原样返回，两个用例从未真正验证消毒。已改用 jsdom（断言未改）并加 `isSupported` 前置断言；同时给 `footer.tsx` 加失效安全（不支持时不注入原始 HTML）|

### 前端工具链（2026-10-06 起本机可用）

```bash
cd web && bun install --frozen-lockfile   # 需要 bun；本机已装 1.4.2
bun run typecheck   # tsgo -b
bun run lint        # oxlint（0 error / 21 warning 为当前基线）
bun test            # node:test + happy-dom；当前 151 pass / 3 fail（3 个为上述既有失败）
bun run build       # rsbuild，漏删引用会在此报错——删代码后必须跑
bun run knip        # 死码分析；测试入口与应用入口已在 knip.config.ts 声明
```

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
2. 打注释 tag 并推送：

   ```bash
   git tag -a v0.0.3 -m "v0.0.3"
   git push origin main v0.0.3
   ```

3. 推 tag **只自动触发一个工作流**（触发面已刻意收敛，避免每次发版扇出）：

   - `Publish Docker image (Multi-arch)` → 构建并推送 `ghcr.io/zhemed/new-api-own:v0.0.3`、
     `ghcr.io/zhemed/new-api-own:0.0.3`（去掉 `v` 的等值别名）与 `:latest`，多架构清单 + cosign 签名

   **交付只有镜像这一条路径**：二进制 Release（`release.yml`）、Electron 桌面壳（`electron-build.yml`）、
   GitCode 同步（`sync-release-to-gitcode.yml`）与手动分支镜像（`docker-image-branch.yml`）
   均已移除（2026-10-06 用户定调：部署就是 `docker run` + 公开镜像，其余产物一律不要）。

4. 校验（**只产出镜像，不再有 GitHub Release 产物**）：

   ```bash
   docker run --rm ghcr.io/zhemed/new-api-own:0.0.3 --version          # 应输出 v0.0.3
   docker buildx imagetools inspect ghcr.io/zhemed/new-api-own:0.0.3    # 应看到 amd64/arm64 清单
   ```

5. 镜像内的版本号来自构建时的 tag：`Dockerfile` 把 `VERSION` 注入 Go ldflags（`common.Version`）与前端 `VITE_REACT_APP_VERSION`，而 CI 会用 tag 覆写 `VERSION` 文件内容，所以**必须走 tag 发版**；只改文件不推 tag 不会触发镜像构建，镜像里会停在旧值（本项目自 2026-10-06 起只产出镜像，不再有 GitHub Release）。
6. 版本注入的 `-ldflags -X` 必须写**完整模块路径** `github.com/QuantumNous/new-api/common.Version`；写成简写（`new-api/common.Version`）会被 Go 静默忽略，镜像内版本会停在内置默认值 `v0.0.0`（`Dockerfile` 用的是完整路径，勿改）。

> 约定：`VERSION` 文件不带 `v`，tag 带 `v`，两者版本号一致；镜像同时提供 `v0.0.3` 与 `0.0.3` 两种拉取标签，指向同一份多架构清单。

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
