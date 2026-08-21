# MAINTENANCE.md — 维护手册

> 本仓库 `zhemed/new-api-own` 是 **公开** 仓库，基于 QuantumNous/new-api 代码基独立维护（非 fork 关系，历史含上游提交，但自 2026-08 起完全自维护）。

## 铁律（违反即事故）

1. **绝不同步上游**：未经仓库所有者明确许可，禁止对 QuantumNous/new-api（或任何上游）执行 fetch / merge / rebase / cherry-pick。上游改动一律不关注、不引入。
2. **不修改受保护标识**：new-api 与 QuantumNous 的名称、品牌、署名（README、许可头、包路径、Docker 镜像名、文档等）一律不得改动（见 `AGENTS.md` Project Governance）。
3. **行为不变原则**：清理代码（lint/format/重构）时不得改变任何用户可见行为；无法保证等价时，宁可用 `oxlint-disable` 注释，也不改行为。
4. **Docker 环境标准（AGENTS.md 强制）**：Docker Engine 29.7.2 + Docker Compose v5.4.0。需 Docker Engine 29.7.2 + Compose v5.4.0；一键安装：`curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash`。
5. **提交前必须通过质量门禁**（见下）。

## 项目是什么

自用 LLM 网关与 AI 资产管理平台（Go + React）：多模型聚合、Key 管理、计费与分发、API 格式转换（OpenAI ⇄ Claude、OpenAI → Gemini）、授权登录、智能路由、思考模式支持。
- 后端：Go 1.22+、Gin、GORM v2；前端：React 19 + TS + Rsbuild（`web/`，包管理用 Bun）
- 数据库：SQLite / MySQL / PostgreSQL 三库兼容；缓存：Redis + 内存
- 独立模块：`relaykit/`（不得依赖根模块，改动后必须 `GOWORK=off` 单独构建）
- 部署：公开镜像 `ghcr.io/zhemed/new-api-own`（无需登录，直接拉取），docker-compose 用 host 网络

## 本机开发环境

| 组件 | 位置/命令 |
|---|---|
| Go | `/usr/local/go/bin/go`（1.26.4），需 `export HOME=/root PATH=$PATH:/usr/local/go/bin GOPATH=/root/go GOMODCACHE=/root/go/pkg/mod GOCACHE=/root/.cache/go-build` |
| Bun | `~/.bun/bin/bun`（1.3.14），所有前端命令前需 `export HOME=/root` |
| GitHub 公开仓库 | 克隆无需凭据（`git clone https://github.com/zhemed/new-api-own.git`），推送需带 token（`https://zhemed:<token>@...`，token 在 `/root/.git-credentials`） |

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
- `edcd6a5e`（8-13）：已移除硬编码 `cs.shemedhb.eu.org`，改用当前站点 origin（http→https 提升）

### 5. 其他
- 模型倍率全精度（`800c26d6`）、用量日志表格对齐、去上游化（README 双语、链接/检查器/i18n 指向自维护仓库）、Docker 环境标准锁定

## 质量门禁现状（2026-08-18 基线）

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

## 自用部署注意事项

- docker-compose 中 `CRITICAL_RATE_LIMIT_ENABLE=false` 是**有意的自用配置**（内网信任环境、方便频繁操作），不是缺陷；若仓库公开或对外提供服务需重新评估
- 公开镜像可直接拉取，无需 `docker login`
- 部署前确保 Docker 为标准版本（29.7.2 + v5.4.0）

## 工作流

1. 改代码 → 涉及文件 lint 0 error + typecheck 通过 → 相关 Go 测试/前端测试
2. 行为变更必须补回归测试（后端 `testify`；前端放 `__tests__/` 目录）
3. 提交信息用项目风格（`fix:` / `feat:` / `chore:` 前缀，描述变更与原因）
4. 提交后推送：`export HOME=/root && git push origin main`（公开仓库，推送需凭据（`/root/.git-credentials`））
