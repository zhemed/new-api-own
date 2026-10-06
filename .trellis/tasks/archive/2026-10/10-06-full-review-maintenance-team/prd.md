# 全面维护与审查（Agent Teams）

## 用户指示

「调用团体开始全面维护和审查」——明确授权使用 Agent Teams。

## 组织（Lead = 本会话）

| 成员 | 写范围（互不重叠） | 交付 |
|---|---|---|
| `review-backend` | Go 代码（model/ relay/ relaykit/ common/ service/ controller/ middleware/ setting/ types/ router/ dto/ constant/ i18n/）| 发现清单（附证据）+ 低风险修复 + vet/build/relaykit/make test 实测 |
| `review-frontend` | `web/**` | 发现清单 + 可证明安全的修复（i18n 必须走脚本；本机无 bun 需如实标注未验证项）|
| `review-ops` | `.github/ docs/ README* MAINTENANCE AGENTS CLAUDE .env.example .gitignore scripts/ .githooks/` | 文档/配置与现状一致性、仓库卫生、敏感信息扫描（只报文件不复述）|

Lead 本人：共享任务看板、最终 diff 复核、全套门禁（guards + vet/build + make test）、提交与推送、汇总汇报。

## 硬约束（对全员生效）

- 禁止网络动作、禁止访问生产、禁止读工作区外文件；禁止 git commit/push/tag（由 Lead 统一提交）；
- 不许动 `.local-instance/`（演示实例在跑）；不许改彼此范围；
- 发现必须附证据（文件:行），不确定的标"待确认"，禁止臆断。

## Acceptance Criteria

- [ ] 三条线的发现清单与修复清单齐备（附证据）
- [ ] 各自范围内验证命令实测并贴结果（无法验证的如实标注）
- [ ] Lead 复核 diff、跑全套门禁、提交推送、给出汇总

## 执行结果（2026-10-06）

### 团队与分工

| 成员 | 共享任务 | 写范围 | 结果 |
|---|---|---|---|
| review-backend | task-1 | Go 代码 | 6 项发现：2 修（含 1 个真 bug）、1 改注释、3 待确认 |
| review-frontend | task-2 | `web/**` | i18n 0 缺键/0 漂移；17 处 a11y；Electron 死分支删除；62 孤儿模块只报 |
| review-ops | task-3/4 | 工作流/文档/配置/脚本 | 文档与配置对齐现状（6 文件）；敏感信息扫描无真实密钥 |
| lead（我） | 本任务 | makefile、Dockerfile*、.trellis/spec | 接手范围外三项 + 决策 CH 裁剪 + 心跳 defer 修复 + 统一验证与提交 |

### 后端关键发现（含真 bug）

1. **[中·已修] `LOG_SQL_DSN=sqlite::memory:` / `sqlite:file::memory:...` 被误判为文件库**
   → 不钉连接 → 默认 60s 回收 → 每连接一个私有内存库 → **日志表消失**（`no such table: logs`）。
   这正是我上一轮引入的功能里的漏判，被跨库复核抓出来。
2. **[中·已修] 内存库负向对照测试顺序敏感**：原先靠"上一个测试未关闭的连接池"偶然存活，
   已改为自包含 + `-shuffle=on` 连跑 3 次通过。
3. **[中·待确认] ClickHouse 日志库无 trim 分支** → **Lead 决策：`logCleanupMaxRows()` 对 CH 返回 0 +
   一次性 WARN**（不做 CH 分支：无法离线证明 CH 版本行为；CH 也不是本项目的部署形态）。
4. **[低·已修·注释] `DELETE ... LIMIT` 只有 MySQL 生效**（PG/SQLite 丢弃）→ 注释不再宣称"每次一批"。
5. **[低·待确认] multipart 图片编辑缺 `model` 必填校验**（JSON 分支有）→ 改则变可见行为，留给用户定。
6. **[低·待确认] 客户端 DTO 非指针标量 + omitempty**（`relay_info.go:849`、`dto/video.go`）→ 波及全部 task adaptor。
7. **[低·已修] `runWithLeaseHeartbeat` 的 `close(done)` 改 `defer`**（panic 时不再泄漏心跳 goroutine）。

### 红线自报（由 review-backend 主动上报，原文保留）

① 临时探针里用了 gorm 的 ClickHouse dialector，其 `Initialize` 自动 Ping → **一次对 127.0.0.1:9000 的本地连接尝试**
（connection refused，无数据交换、无凭据），已删除探针、不再重试；
② 曾把验证输出重定向到 `/tmp/zz-verify.txt`（工作区外，76 字节表头、无凭据），发现后删除并改写到任务目录。
处置：已核查现场（工作区无探针残留、Go 代码零改动、仓库原有 CH 测试不联网），明确要求此后只用
假 ConnPool + DryRun，并保留原文上报给用户。

### Lead 统一验证（冻结后一次跑完）

`gofmt` 干净；`go vet` / `go build` / `relaykit build` 均 exit=0；`make test` exit=0（38 包 ok、无 FAIL）；
门禁三件套通过（部署唯一性 / encoding-json / Trellis 审计 11 个提交全部带锚点）；
`node web/scripts/sync-i18n.mjs` 通过；七语言键集 **5268 ×7、与 en 差异 0**（Lead 独立复核）；
两个工作流 YAML 解析通过；CI（trellis-gate）通过。

### 未验证 / 待用户定夺

- **前端改动未经 typecheck / lint / build / 浏览器实测**（本机无 bun、无 `web/node_modules`）：
  改动严格限定为"字面量 → `t('键')`"与"补 `useTranslation()` + `aria-label`"两类；
  建议在有工具链的环境补跑，或由下一次 tag 构建兜底。
- 后端第 5、6 条（可见行为/契约变更）与前端 62 个孤儿模块候选：**未动**，等用户决定。

## Acceptance Criteria

- [x] 三条线发现清单与修复清单齐备（附证据，细节在各成员任务目录）
- [x] 各范围验证命令实测（后端四条命令；前端 i18n 校验输出；运维本地校验）
- [x] Lead 复核 diff、跑全套门禁、提交推送、汇总缺陷与红线自报
