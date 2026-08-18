# MAINTENANCE.md — 维护手册

> 本仓库 `zhemed/new-api-own` 是 **私有** 仓库，基于 QuantumNous/new-api 代码基独立维护（非 fork 关系，历史含上游提交，但自 2026-08 起完全自维护）。

## 铁律（违反即事故）

1. **绝不同步上游**：未经仓库所有者明确许可，禁止对 QuantumNous/new-api（或任何上游）执行 fetch / merge / rebase / cherry-pick。上游改动一律不关注、不引入。
2. **不修改受保护标识**：new-api 与 QuantumNous 的名称、品牌、署名（README、许可头、包路径、Docker 镜像名、文档等）一律不得改动（见 `AGENTS.md` Project Governance）。
3. **行为不变原则**：清理代码（lint/format/重构）时不得改变任何用户可见行为；无法保证等价时，宁可用 `oxlint-disable` 注释，也不改行为。
4. **Docker 环境标准（AGENTS.md 强制）**：Docker Engine 29.7.2 + Docker Compose v5.4.0。任何 docker 操作前必须先运行 `./check-docker-env.sh` 校验；版本不符先安装锁定版本（见 AGENTS.md）。
5. **提交前必须通过质量门禁**（见下）。

## 项目是什么

自用 LLM 网关与 AI 资产管理平台（Go + React）：多模型聚合、Key 管理、计费与分发、API 格式转换（OpenAI ⇄ Claude、OpenAI → Gemini）、授权登录、智能路由、思考模式支持。
- 后端：Go 1.22+、Gin、GORM v2；前端：React 19 + TS + Rsbuild（`web/`，包管理用 Bun）
- 数据库：SQLite / MySQL / PostgreSQL 三库兼容；缓存：Redis + 内存
- 独立模块：`relaykit/`（不得依赖根模块，改动后必须 `GOWORK=off` 单独构建）
- 部署：私有镜像 `ghcr.io/zhemed/new-api-own`（需 `docker login ghcr.io -u zhemed`），docker-compose 用 host 网络

## 本机开发环境

| 组件 | 位置/命令 |
|---|---|
| Go | `/usr/local/go/bin/go`（1.26.4），需 `export HOME=/root PATH=$PATH:/usr/local/go/bin GOPATH=/root/go GOMODCACHE=/root/go/pkg/mod GOCACHE=/root/.cache/go-build` |
| Bun | `~/.bun/bin/bun`（1.3.14），所有前端命令前需 `export HOME=/root` |
| GitHub 私有仓库 | 克隆/推送需带 token：`git clone https://zhemed:<token>@github.com/zhemed/new-api-own.git`（token 在 `/root/.git-credentials`；git 操作需 `export HOME=/root`） |

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

### 已知考量（未改，需决策或条件成熟后处理）
- **A2**：客户端显式 effort 在 deepseek/newapi 透传渠道、OpenRouter 渠道（274/292 行清空）、chat→Claude 转换、Gemini 渠道（全无记录）丢失。建议在 `TextHelper` 公共路径统一同步 `request.ReasoningEffort` 到 info。
- **A3**：`-none` 语义不一致（chat/claude 路径不记录、Responses 路径记 "none"）。需决策统一口径。
- **B1**：`-max/-none` 实际只覆盖 deepseek/newapi 渠道 + Claude 格式；普通 OpenAI 兼容渠道（openai/advancedcustom 等）原样上送、Gemini 渠道 `-max` 被剥掉但不注入 thinking（静默丢语义）。
- **B2（计费）**：`FormatMatchingModelName` 把 `deepseek-v4-*` 带后缀名归一为基名 → 管理员配置的带后缀独立倍率永远匹配不到、保存时被静默改写。
- **B5**：`-max` 上送 `reasoning_effort:"max"` 的枚举兼容性需与上游核对。
- **C1/C2（决策项）**：永不过期默认开启 + 10 年 cookie + 活跃会话上限（默认 50）永久锁。**当前是自用部署的有意配置**（MAINTENANCE.md 已注明）；若对外提供服务需重新评估或加活跃淘汰。
- **E5（部分）**：既有测试（session_expiry、auth_session_policy、adaptor_reasoning）用手写断言未用 testify；后续改动相关文件时顺带迁移。
- **D2**：HTTP 部署下 CC Switch 导入无 HTTPS 告警提示。
- **D3**：`.github/ISSUE_TEMPLATE/*` 仍指向 docs.newapi.ai（上游文档站），追求彻底去上游化可替换。

## 自用部署注意事项

- docker-compose 中 `CRITICAL_RATE_LIMIT_ENABLE=false` 是**有意的自用配置**（内网信任环境、方便频繁操作），不是缺陷；若仓库公开或对外提供服务需重新评估
- 私有镜像部署需先 `docker login ghcr.io -u zhemed`（用 GitHub token 作为密码）
- 部署前运行 `./check-docker-env.sh` 校验 Docker 环境

## 工作流

1. 改代码 → 涉及文件 lint 0 error + typecheck 通过 → 相关 Go 测试/前端测试
2. 行为变更必须补回归测试（后端 `testify`；前端放 `__tests__/` 目录）
3. 提交信息用项目风格（`fix:` / `feat:` / `chore:` 前缀，描述变更与原因）
4. 提交后推送：`export HOME=/root && git push origin main`（私有仓库，凭据在 `/root/.git-credentials`）
