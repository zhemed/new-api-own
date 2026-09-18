# 关闭全局 web / API / 敏感操作限流（含搜索限流）

## Goal

本 fork 默认不启用全局 web 限流、全局 API 限流、敏感操作（critical）限流与按用户搜索限流，
并在部署配置中显式声明同样的关闭状态。

## Background

- 开关读取：`common/init.go:121/125/129/133`，四者默认 `true`。
- 关闭语义：`middleware/rate-limit.go:160-179`（web/api/critical）与 `:238-242`（search）在开关为
  `false` 时返回 `defNext`，中间件退化为空操作。
- 生效位置：`router/web-router.go:26`、`router/api-router.go:19`、`router/dashboard.go:14`，
  以及 `router/api-router.go` 内 36 处 `middleware.CriticalRateLimit()`（登录/注册/重置密码/2FA/OAuth 等）。
- `*_DURATION` 只在对应 `ENABLE=true` 时参与限流计算，关闭后不会生效。
- 现状：`docker-compose.yml:44` 已经对 `CRITICAL_RATE_LIMIT_ENABLE` 关闭，`MAINTENANCE.md:196`
  把它记录为有意的自用配置。

## Requirements

1. 代码默认值改为关闭：`common/init.go` 四处 `GetEnvOrDefaultBool(..., true)` → `false`
   （`GLOBAL_API_RATE_LIMIT_ENABLE`、`GLOBAL_WEB_RATE_LIMIT_ENABLE`、`CRITICAL_RATE_LIMIT_ENABLE`、
   `SEARCH_RATE_LIMIT_ENABLE`）。
2. `common/constants.go` 中 `SearchRateLimitEnable` 的包级初值同步改为 `false`，避免声明即为 `true`。
3. `docker-compose.yml` 显式声明四个开关为 `false`（`CRITICAL_RATE_LIMIT_ENABLE` 已存在，保留）。
4. `.env.example` 增加这四个开关的说明块：注明本 fork 默认关闭，以及重新开启的方式。
5. `MAINTENANCE.md` 的部署安全基线/自用配置说明更新为"四个开关均为有意的自用关闭"，
   并写明对外提供服务时需要重新评估。
6. `*_DURATION` 与 `*_RATE_LIMIT`（次数）数值不改，保留原值以便随时恢复；不删除开关与中间件代码。

## Out of Scope

- 不动上传/下载限流（`UploadRateLimit` / `DownloadRateLimit`）与 relay 侧渠道/用户配额逻辑。
- 不新增 UI 开关，不改变环境变量名称与读取方式。

## Constraints / Risks

- 关闭后失去针对爆破式请求的全局兜底；只适用于内网/自用部署，需在 `MAINTENANCE.md` 明确标注。
- 仓库内没有任何 `*_test.go` 引用这四个开关，改动不影响测试断言。
- 环境变量仍可覆盖：任一部署设 `GLOBAL_API_RATE_LIMIT_ENABLE=true` 即可恢复对应限流。

## Acceptance Criteria

- [ ] `common/init.go` 四处默认值为 `false`，`common/constants.go` 的 `SearchRateLimitEnable` 初值为 `false`
- [ ] `docker-compose.yml` 含四个 `*=false` 环境变量
- [ ] `.env.example` 与 `MAINTENANCE.md` 的说明与实际默认值一致，并写明重新开启方式
- [ ] `GOWORK=off go vet ./...`、`GOWORK=off go build ./...`（root 与 `relaykit` 各自独立）无诊断输出
- [ ] 提交信息带 `[task:09-18-disable-global-rate-limits]`
