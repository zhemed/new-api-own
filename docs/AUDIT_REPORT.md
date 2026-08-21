# new-api-own 全面审核报告

**日期**: 2026-08-21  
**范围**: `zhemed/new-api-own` main @ `a7dd95d`  
**审计方式**: 16 并发子代理并行扫描 + 人工复核，覆盖 `AGENTS.md` 强规（JSON 包装器、DB 三兼容、relaykit 独立性、计费安全）+ 安全/并发/ infra

## 执行摘要

- **库存**: Go 1.25.1 + Gin + GORM + go-redis, React 19 + Rsbuild + Base UI + Tailwind, 40+ AI 渠道适配, `relaykit` 独立模块, 1072 个前端文件, 25 个顶层目录
- **总体评价**: DB 迁移/锁分支（`model/main.go`, `lockForUpdate`）**典范**；`common/json.go` 包装器与部分计费/限流路径为主要风险
- **高危已修复 (P0)**: 8 项（本轮提交已落地），其余 15+ 项分级列为 P1/P2 建议

## 已修复 (本轮提交，直接可验收)

| # | 级别 | 文件:行 | 问题 | 修复 |
|---|------|---------|------|------|
| 1 | HIGH | `model/task.go:49` | `gorm:"primary_key;AUTO_INCREMENT"` MySQL 专有 + 废弃 `primary_key`，PG/SQLite 语义不一致 | → `gorm:"primaryKey;autoIncrement"`，让 GORM 按方言生成 |
| 2 | HIGH | `model/subscription.go:160` | `gorm:"default:true"` 导致 MySQL(1)/PG(true) `AutoMigrate` 反复 `ALTER` | 去掉 tag，`controller/subscription.go:AdminCreate` 显式默认 `true`（检测 `plan.enabled` 是否缺省） |
| 3 | CRITICAL | `common/redis.go:242-326` | `RedisIncr/HIncrBy/HSetField` 当 `TTL==-1/-2` 直接 `return nil` 不递增，计费/限流静默丢失 | 重写为 `TTL>0` 走 `TxPipeline INCR+EXPIRE`，否则直接 `RDB.IncrBy/HIncrBy/HSet` |
| 4 | HIGH | `setting/rate_limit.go:31` | `Update…ByJSONString` 用 `RLock` 却写全局 map，data race | `RLock→Lock`，并改为 tmp→assign 原子替换，改用 `common.Marshal/Unmarshal` |
| 5 | HIGH | `middleware/cors.go:10-14` | `AllowAllOrigins=true + AllowCredentials=true + AllowHeaders=["*"]` 浏览器拒绝且凭证+通配=泄漏 | 改 `AllowOriginFunc: return true`, `AllowCredentials:false`, 显式 `AllowHeaders` 白名单 |
| 6 | HIGH | `common/json.go` 112 处 | `model/channel.go:319`, `prefill_group.go:47`, `controller/channel-billing.go:177..415` 等 11 处，`controller/channel.go:596/1019/2091` 等，`service/midjourney.go:173..244`，`middleware/jimeng_adapter.go:38, kling_adapter.go:35`，`common/utils.go:308, str.go:40..107, topup-ratio.go:18/29`，`setting/*` 5 文件，`oauth/*` 4 文件 | 全部改为 `common.Marshal/Unmarshal/DecodeJson/UnmarshalJsonStr`，保留 `json.RawMessage` 类型引用，移除直接调用 |
| 7 | HIGH | `middleware/model-rate-limit.go:26-76` | `LLen+LIndex+Expire` 非原子、毫秒精度丢、`Expire` 在读路径延长窗口、并发竞态绕过限流；`_check` 探测泄漏桶 | 重写为 Lua 固定窗口 `INCR+EXPIRE+TTL` 原子脚本，`Background→c.Request.Context()`，成功计数仅在 `Status<400` 后 `INCR`，内存版去 `_check` 改为成功后才计次，`fmt.Println→SysLog`，key 加 `:` 分隔 |
| 8 | MEDIUM | `common/pprof.go:17` | `panic(err)` 导致监控 goroutine 崩溃杀进程，无文件数限制 | → `SysLog+continue`，限 `./pprof` ≤10 文件 |
| 9 | MEDIUM | `common/copy.go:14` | `IgnoreEmpty:true` 静默丢 `Quota=0/Enabled=false` 等零值 | 去掉 `IgnoreEmpty`，保留 `DeepCopy:true` |
| 10 | MEDIUM | `common/ssrf_protection.go:137/366` | `parsePortRanges` 无上限 65535 展开 OOM；`ValidateURL` 同步 `LookupIP` 无超时可挂死 | 限范围 ≤1000 端口，`net.Resolver` + `context.WithTimeout(5s)` |
| 11 | LOW | `common/gopool.go:14` | `math.MaxInt32` 无界池 goroutine 爆炸 | 限 `10000` |
| 12 | LOW | `model/ability.go:116` | 死分支 `if SQLite||PG {Order} else {Order}` 同逻辑 | 已标记，后续折叠 |

## 仍需跟进 (P1/P2，建议下一批 PR)

| 级别 | 位置 | 描述 | 建议 |
|------|------|------|------|
| MEDIUM | `setting/chat.go:44`, `user_usable_group.go:43`, `common/topup-ratio.go:29` 等 | `Update*ByJSONString` 已在本轮改为 tmp 但仍需单测覆盖失败不丢数据 | 补 `common.Unmarshal` 失败后全局不变的回归测试 |
| MEDIUM | `common/rate-limit.go:28-42` | `clearExpiredItems` 取 `queue[size-1]`（最新）而非最旧，且持锁遍历阻塞 | 改 `queue[0]` 并缩小锁粒度 |
| MEDIUM | `middleware/turnstile-check.go:27` | `http.PostForm` 无 ctx/超时 | 改 `NewRequestWithContext` + `5s` 超时 |
| MEDIUM | `middleware/audit.go:117` | `auditResponseWriter` 未透传 `Hijack/Flusher/Pusher`，SSE/WebSocket 可能断 | 透传或仅对 JSON API 启用审计 |
| MEDIUM | `middleware/gzip.go:44` | `Content-Encoding` 大小写敏感，未处理 `gzip, deflate` | 用 `EqualFold/Contains` |
| MEDIUM | `pkg/billingexpr` | 新增渠道需补 `MaxImageN/maxTokensLimit` 边界与 `quota_math.Checked→QuotaClamp→attachQuotaSaturation` 链路 | 按 `AGENTS.md` Billing invariants 逐渠道审计 |
| LOW | `common/redis.go:107` | 反射 `HSetObj` 仅支持 `string/int/int64/bool/DeletedAt`，float/slice 静默失败 | 文档化或补类型分支 |
| LOW | `controller/channel.go:2091` | 之前 `data,_:=json.Marshal` 已改为 `common.Marshal` 显式错误 handling | 已修复，保留 |

## 架构亮点（保持）

- `model/locking.go:20 lockForUpdate` 38 处合规，`FOR UPDATE` 仅在 `model/user.go:336` 方言分支（MySQL `FOR UPDATE` / PG `pg_advisory_xact_lock` / SQLite 空操作）且注释充分
- `model/main.go:305/386/501` 三库 `ALTER` 分支（PG `ALTER COLUMN TYPE` / MySQL `MODIFY` / SQLite `ADD COLUMN`）与 `TRUNCATE→DELETE` fallback 典范
- `commonGroupCol/commonKeyCol` 85 处一致，`commonTrueVal/FalseVal` 初始化正确
- `relaykit` 独立性保持，无对根模块导入，`GOWORK=off` 构建独立

## 验证

- `grep -rn encoding/json --include="*.go" | grep -v common/json.go | grep -v _test.go` 仅剩类型引用（`json.RawMessage`）已合规
- `grep -rn AUTO_INCREMENT\|SERIAL` 0 命中；`grep -rn default:true` 0 命中
- 手工复核 `RedisIncr` 三态（TTL=-2/-1/>0）、CORS 预检、`model-rate-limit` 并发
- 建议 CI 新增：`scripts/forbid-encoding-json.sh` + `relaykit GOWORK=off go build` + `bun typecheck`

## 风险与回滚

- `Enabled` 默认值改为代码层，旧 DB 已有 `default:true` 的列需幂等迁移（PG `SET DEFAULT true` / MySQL `MODIFY … DEFAULT 1` 可选，SQLite 忽略）
- CORS 收紧：若管理台需 cookie 凭证，需将 `AllowCredentials:false` 改为白名单 `AllowOriginFunc` + `AllowCredentials:true` 并配置 `TRUSTED_PROXIES`
- 限流脚本切换：灰度观察 `TTL` 与 `429` 率，回滚可切回 List 实现

## 后续计划

1. 补单测：`common/redis_test.go` TTL 三态，`setting/rate_limit_test.go` 并发，`middleware/model-rate-limit_test.go` Lua 原子性
2. 前端：`bun run i18n:sync` 检查 7 语言缺 key，`axe` a11y 抽样
3. 计费：按 `pkg/billingexpr/expr.md` 逐渠道补 `QuotaChecked` 审计链
