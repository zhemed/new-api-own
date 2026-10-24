# 证据：修复 model 包日志预算用例的测试隔离 flaky

- 任务：Trellis `10-24-latest-and-fix-flaky-log-tests`；共享任务板 `task-36`
- 执行者：worker-script；时间：2026-10-24
- 写作用域：`model/log_budget_test.go`（**唯一改动**）、证据文件。
- **生产代码零改动**（`model/log_budget.go` 未动）：结论是**测试隔离缺陷**，不是生产 goroutine 泄漏（判定见 §3）。
- 未提交、未推送、未发版；未动 `web/`。

## 1. 症状与机制（已核实）

翻红的两个用例与断言：

| 用例 | 失败断言 | 观测值 |
| --- | --- | --- |
| `TestLogPayloadTrimIsSingleFlight` | `require.True(t, beginLogPayloadTrim())` | `Should be true`（单飞标志被上个用例的裁剪占着） |
| `TestLogPayloadBudgetDisabledByDefault` | `assert.Zero(t, logPayloadBytes.Load())` | `Should be zero, but was 14196`（字节计数器被后台任务回写） |

机制（`model/log_budget.go`，均为 fire-and-forget `gopool.Go`）：

1. 写日志 `createLog` → `noteLogPayloadBytes`（:172）累加 `logPayloadBytes`；超预算 → `triggerLogPayloadTrim`（:198）：
   **同步** `beginLogPayloadTrim()` 置位 `logByteTrimRunning`，再异步起裁剪 goroutine。
2. 裁剪 goroutine 末尾在 `calibrateLogPayloadCounter`（:329-335）里 `logPayloadBytes.Store(total)`，
   然后才 `endLogPayloadTrim()`（:228）释放标志。
3. 另有一路一次性校准 `ensureLogPayloadCalibration`（:186-194，`sync.Once` + gopool）也会 `Store(total)`，
   **且完全不碰单飞标志**。

这些 goroutine 可能活过用例边界 → 下个用例里：
- 标志仍为 true → 单飞用例第一条断言失败；
- 用例刚 `Store(0)`，后台随后 `Store(total)` → 「预算关闭时不计字节」断言失败。

## 2. 稳定复现（不是"偶发见过"）

纯概率复现很难：`go test ./model/ -count=40` 连跑 3 轮、以及"背景其它包并发 + model -count=30" 3 轮，
**均未自然翻红**（窗口在微秒级；上一轮 verifier 是在全仓 scale 下 6 次中 2 次）。

因此改用**确定性手法**：用生产函数制造"上个用例遗留的裁剪"，并用 `TestMain` 里
`sqlDB.SetMaxOpenConns(1)` 的唯一连接把它卡在第一次查询上，使跨用例时序变得确定。
然后**直接调用两个真实目标用例**（不是仿写断言）：

```go
// 临时验证文件（验证后已删除，内容见本报告附录）
tx := LOG_DB.Begin()              // 占住唯一连接
triggerLogPayloadTrim()    // 生产函数：同步占用单飞 + 异步起 goroutine
go func(){ time.Sleep(150ms); tx.Rollback() }()   // 150ms 后放行 = "上个用例的裁剪稍晚结束"

t.Run("singleflight", TestLogPayloadTrimIsSingleFlight)
t.Run("budget_disabled", TestLogPayloadBudgetDisabledByDefault)
```

### 修复前输出（原测试文件 + 上述确定性场景；3/3 次一致）

```console
$ go test ./model/ -count=1 -run 'TestValidateIsolationAgainstLeftoverTrim' -v
=== RUN   TestValidateIsolationAgainstLeftoverTrim/singleflight
        	Error:      	Should be true
=== RUN   TestValidateIsolationAgainstLeftoverTrim/budget_disabled
        	Error:      	Should be zero, but was 14196
        	Error:      	Not equal:
--- FAIL: TestValidateIsolationAgainstLeftoverTrim (0.15s)
    --- FAIL: .../singleflight (0.00s)
    --- FAIL: .../budget_disabled (0.15s)
FAIL
```

> 两条失败与生产事故报告**逐字一致**：`Should be true` / `Should be zero, but was <N>`。

独立复现脚本（无需改测试，直接证明机制）：临时用例 `TestReproLeftoverTrimBreaksNextTest` 连续 3/3 输出

```console
REPRODUCED #1: beginLogPayloadTrim() == false —— 单飞标志被上个用例占用
REPRODUCED #2: 复位为 0 之后残留 logPayloadBytes=5670 —— 字节计数器残留
```

## 3. 根因判定：测试隔离，不是生产 goroutine 泄漏

- 每个后台 goroutine 都会**终止**（无循环、无永久阻塞），不构成泄漏；生产中 fire-and-forget 只影响
  预算计算的实时性（自校正设计，见 `calibrateLogPayloadCounter` 注释），不影响正确性。
- 问题在于**测试**：用例结束时不等待后台任务收敛，就复位共享包级状态
  （`logPayloadBytes`、`logByteTrimRunning`、`logs` 表），于是被在途 goroutine 二次改写。
- 结论：**只改测试即可**，符合"优先只改测试"的要求；未改生产代码。

## 4. 修复（仅测试文件）

新增两个 helper，并把本文件 8 个用例的"删表 + 复位"统一收敛：

```go
// 有界等待后台收敛：单飞标志空闲 + 计数器连续 5 次观测不变（5ms 轮询，5s 上限）。
// 失败信息带上实测的标志与计数器值；不使用固定 sleep。
func waitLogBudgetIdle(t *testing.T)

// 进入前收敛 → 复位计数器 → 清空日志表；离开前再收敛一次。
// cleanup LIFO：收敛等待晚于 truncateTables 的删表注册，因此先收敛、后删表。
func beginLogBudgetTest(t *testing.T)
```

各用例的 `truncateTables(t); logPayloadBytes.Store(0)` → `beginLogBudgetTest(t)`；
`TestLogPayloadTrimIsSingleFlight` 与磁盘库用例也加入进入前收敛。

### 断言强度：未降低

- `git diff` 中**没有任何原有 `assert.*` / `require.*` 断言被修改或删除**（唯一新增的断言语句是
  helper 内部的 `require.Eventually` 等待）。
- 未使用 `t.Skip`、未加固定 sleep 掩盖；两个契约断言原样保留：
  「并发触发只允许一路赢得单飞」「未配置预算时不记账、不触发裁剪」。
- 用例可单独运行、可任意顺序运行（`-shuffle=on` 已验证）。

### 修复后：同一确定性场景

```console
$ go test ./model/ -count=1 -run 'TestValidateIsolationAgainstLeftoverTrim' -v
--- PASS: TestValidateIsolationAgainstLeftoverTrim (0.24s)
    --- PASS: .../singleflight (0.20s)     # 等待遗留裁剪收敛后才断言，逻辑仍是"先拿单飞"
    --- PASS: .../budget_disabled (0.04s)
ok  	github.com/QuantumNous/new-api/model	0.265s

# 连跑 5 次：5/5 PASS
```

## 5. 验证统计（全部实际执行）

| 命令 | 结果 |
| --- | --- |
| `go test ./model/ -count=20 -shuffle=on` | **ok 135.593s** |
| 并发负载：后台 `go test ./service/... ./controller/... ./setting/... -count=2` + `go test ./model/ -count=30` ×3 轮 | **3/3 绿**（195.7s / 204.6s / 200.3s） |
| `go test ./... -count=1` | **exit=0，0 FAIL，36 ok** |
| `go test ./... -count=3` | 除下面「已知无关失败」外全绿（`model` 17.962s ok） |
| `go build ./...` / `go vet ./...` | 均 exit 0 |
| `go test ./model/... ./controller/... -count=1` | ok（model 4.525s / controller 0.446s） |

## 6. 必须升级给 Lead 的**无关**既有问题（→ 已在 §9 修复）

`middleware/TestRedisFailurePolicies` 在 `-count>1` 时**必然**失败（本次实测 3/3），
但**与本次改动无关**：把我的改动 `git stash` 后在干净 HEAD 上复跑，失败完全相同。
**该问题已由 Lead 追加的 task-38 修复，见 §9（修复后 `go test ./... -count=3` 已全绿）。**

```console
# 含我的改动
$ go test ./middleware/ -count=3
--- FAIL: TestRedisFailurePolicies
    middleware/rate_limit_test.go:224: expected: 204, actual: 429
# git stash 掉我的改动后（干净 HEAD）
$ go test ./middleware/ -count=3
--- FAIL: TestRedisFailurePolicies
    middleware/rate_limit_test.go:224: expected: 204, actual: 429
```

根因（已定位）：该用例先 `redisClient.Close()` 制造 Redis 故障，让 `EmailVerificationRateLimit()`
回落到**包级全局** `inMemoryRateLimiter`；同一进程内 `-count=3` 重复执行时，全局限流计数未复位
（`common.InMemoryRateLimiter.Init` 只做一次性初始化、**不会清空已有 store**），
固定 IP `192.0.2.62:12345` 第 2 次就被判 429。

建议（不在本次写作用域，未擅自改）：
1. 在该用例/`useRateLimitMiniRedis` 的 cleanup 里复位内存限流器（需给 `common.InMemoryRateLimiter`
   加一个仅供测试的 reset，或在 `middleware` 测试内重建该全局变量）；
2. 或让用例使用每次唯一的 key/IP（避免跨 repetition 命中同一计数）；
3. 若只想让本任务验收通过，可把该用例单独 `-count=1` 跑、其余包 `-count=3`——但这只是绕过。

## 7. 风险与回滚

- 改动只在测试代码，无生产影响；回滚为单文件 revert `model/log_budget_test.go`。
- 收敛等待的稳定性判据是"连续 5 次观测不变（≈25ms）+ 5s 上限"。理论上若后台任务恰好在这 25ms 之后
  才写计数器仍可能漏，但该路径需先通过单飞标志/一次性校准触发，而本文件用例在收敛期间不会再触发新任务；
  helper 超时会**直接 fail 并打印实测值**，不会静默通过（不掩盖问题）。
- 未触碰上一轮提到的生产改善点（如给后台任务加可观测计数器）；如需，建议单开任务。

## 8. 建议下一步

1. Lead/verifier 复核 `git diff -- model/log_budget_test.go`（纯测试）与 `git diff -- middleware/ common/`（§9）。
2. §6 的 middleware 既有失败已按 Lead 裁决在 §9 修复，`go test ./... -count=3` 已全绿（R8 达成）。
3. 两份改动（model 测试隔离 + middleware/common 限流复位）可一起按闸门提交（本任务未提交、未发版）。

## 9. middleware 限流用例 `-count>1` 必现失败（R7/R8 追加）

### 9.1 复现原文（必现，不是概率）

```console
$ go test ./middleware/ -run TestRedisFailurePolicies -count=3
[ERR] ... | rate limit check failed (mark=FAIL-IP): redis: client is closed
[ERR] ... | rate limit check failed (key=rateLimit:v2:user:FAIL-USER:7): redis: client is closed
--- FAIL: TestRedisFailurePolicies (0.00s)
    rate_limit_test.go:224:
        	Error Trace:	/root/new-api-own/middleware/rate_limit_test.go:224
        	Error:      	Not equal:
        	            	expected: 204
        	            	actual  : 429
FAIL	github.com/QuantumNous/new-api/middleware	0.013s
```

### 9.2 根因复核（与预判一致）

- `TestRedisFailurePolicies` 先 `redisClient.Close()` 制造 Redis 故障；`/email` 路由的
  `EmailVerificationRateLimit()` → `redisEmailVerificationRateLimiter` 出错后回落到
  `memoryEmailVerificationRateLimiter`，用的是**包级全局** `inMemoryRateLimiter`
  （`middleware/rate-limit.go:38`）。
- 回落 key = `"EV:" + ClientIP`，本用例固定 IP `192.0.2.62`；
  `EmailVerificationMaxRequests = 2`、窗口 30 秒（`email-verification-rate-limit.go:13-15`）。
- `common.InMemoryRateLimiter.Init` 只在 `store == nil` 时初始化，**不会清空已有计数**；
  `-count=N` 在同一进程内重复执行，第 1、2 轮各记 1 次，第 3 轮累计到 2 次 → 判 429。
  实测 `-count=3` 必现、`-count=5` 必现。
- 结论：**测试隔离**问题（进程级全局状态跨 repetition 残留），不是限流逻辑错误；
  生产里 `-count` 场景不存在，但共享限流器需要可复位的测试入口。

### 9.3 修复

采用任务给出的路线 (a)：给 `common.InMemoryRateLimiter` 增加**最小、race-safe** 的 `Reset()`，
再用测试 helper 在用例首尾各复位一次。**没有**采用"换唯一 key/IP"（会削弱"同一 IP 重复触发
被限流"的契约，属绕开而非修复）。

```diff
--- a/common/rate-limit.go
+func (l *InMemoryRateLimiter) Reset() {
+	l.mutex.Lock()
+	defer l.mutex.Unlock()
+	for key := range l.store {
+		delete(l.store, key)
+	}
+}
```
- 原地清空 map：保留 `expirationDuration` 与 `Init` 启动的清理 goroutine，不会让限流器失效；
  未初始化过的实例（store 为 nil）保持未初始化，后续 `Init` 仍能正常初始化。
- 与 `Request`/`clearExpiredItems` 用同一把 mutex，**无数据竞争**（`go test -race` 已验证）。
- 生产代码零调用、零行为变更（纯新增方法）。

```diff
--- a/middleware/rate_limit_test.go
+func resetMemoryRateLimiter(t *testing.T) {
+	t.Helper()
+	inMemoryRateLimiter.Reset()
+	t.Cleanup(func() { inMemoryRateLimiter.Reset() })
+}
 func TestRedisFailurePolicies(t *testing.T) {
 	gin.SetMode(gin.TestMode)
+	resetMemoryRateLimiter(t)
 	_, redisClient := useRateLimitMiniRedis(t)
```

### 9.4 断言强度：未降低

- `git diff -- middleware/rate_limit_test.go` 中**没有新增/修改/删除任何 `assert` / `require` 断言**。
- 失败策略契约原样保留：Redis 故障时 `/ip`、`/user` 必须返回 500 且响应体为空；
  `/email` 回落内存限流器必须放行（204）。
- 无 `Skip`、无 `time.Sleep` 掩盖、未换 key 绕开。
- 用例可单独运行、`-count` 重复运行、`-shuffle=on` 任意顺序运行。

### 9.5 验证统计（全部实际执行）

| 命令 | 结果 |
| --- | --- |
| `go test ./middleware/ -run TestRedisFailurePolicies -count=5` | **ok 0.013s** |
| `go test ./middleware/ -count=3` | **ok 0.085s** |
| `go test ./middleware/ -count=1 -shuffle=on` ×3 | **3/3 ok** |
| `go test ./middleware/ ./common/ -race -count=3`（相关用例） | **ok**（middleware 1.590s / common 3.183s） |
| **`go test ./... -count=3`（R8 最终验收）** | **exit=0，0 FAIL，36 ok（全绿）** |
| `go build ./...` / `go vet ./...` | 均 exit 0 |

### 9.6 影响面与回滚

- 生产影响面：`common.InMemoryRateLimiter` 新增一个导出方法，现有调用方与行为均不变；
  唯一风险是"未来有人误在业务里调用 Reset 清空限流计数"——已在 doc 里写明仅测试用途。
- 回滚：`git revert` 这两个文件的改动即可（纯新增方法 + 测试 helper/调用）。
- 未提交、未推送、未发版。

## 附录：临时复现/验证文件（已删除，可复原）

- `model/zz_repro_flaky_test.go`：用 `triggerLogPayloadTrim` + 占住唯一连接，确定性打印
  `REPRODUCED #1`（单飞被占）与 `REPRODUCED #2`（残留 5670）。
- `model/zz_validate_isolation_test.go`：同样的遗留裁剪场景下，直接 `t.Run` 两个真实目标用例，
  修复前 FAIL / 修复后 PASS（即 §2、§4 的 before/after 输出）。
- 两者均为临时验证用途，验证完删除；如需复核可按本报告描述重建。
