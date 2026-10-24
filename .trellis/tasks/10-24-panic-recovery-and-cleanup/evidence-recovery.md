# 证据：加固既有 panic recovery（不回显原文 + 堆栈级日志 + 已写响应保护）

- 任务：Trellis `10-24-panic-recovery-and-cleanup`；共享任务板 `task-50`
- 执行者：worker-script；时间：2026-10-24
- 结论：**加固既有实现**（不是新增）。`main.go:174` 的 `gin.New()` 引擎此前已通过
  `gin.CustomRecovery` 兜 panic；本轮把 handler 抽到 `middleware.PanicRecovery()` 并修掉 P1/P2/P3。
- 写作用域：`main.go`、`middleware/`、证据文件。未提交、未推送、**未发版**；未动 `web/`、`data/`、实例。

## 0. 现状复核（含对上轮错误结论的更正）

```console
$ grep -rn "gin.New()\|gin.Default()" --include=*.go . | grep -v _test.go
./main.go:174:	server := gin.New()
$ grep -rn "gin.CustomRecovery\|gin.Recovery" --include=*.go . | grep -v _test.go
（本轮改动前：main.go:179 的 gin.CustomRecovery）
```

- **生产代码只有 main.go 一个 gin 引擎**，因此 panic 覆盖面无遗漏（不存在"某引擎没装 recovery"）。
- 上一轮验证报告"全仓没有 panic recovery"是**漏看 main.go**，已在 PRD 更正；本轮不新增机制，只加固。

## 1. 三个真实问题（改动前）

| # | 问题 | 改动前代码表现 |
| --- | --- | --- |
| P1 | 信息泄露 | `message: fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: ...", err)` —— panic 原文（可能含 SQL / 密钥 / 内部路径）直接回给客户端 |
| P2 | 可观测性不足 | 只 `common.SysLog(...)`（INFO 级、写 DefaultWriter）；**无堆栈**、无 request id；gin 自带 recovery 的日志在 debug 模式还会 dump 请求头（仅遮蔽 Authorization，**Cookie 未遮蔽**） |
| P3 | 已写响应后追加 JSON | 无条件 `c.JSON(500, ...)`；SSE/流式已写出后 panic 会在既有响应体后追加 JSON，产生脏响应 |

## 2. 修复（diff 摘要）

`main.go`（-9/+4）：

```diff
-	server.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
-		common.SysLog(fmt.Sprintf("panic detected: %v", err))
-		c.JSON(http.StatusInternalServerError, gin.H{
-			"error": gin.H{
-				"message": fmt.Sprintf("Panic detected, error: %v. Please submit a issue here: ...", err),
-				"type":    "new_api_panic",
-			},
-		})
-	}))
+	// panic recovery 必须在最外层（先于 RequestId 等中间件）：兜住后续所有 handler 的 panic。
+	server.Use(middleware.PanicRecovery())
```

新增 `middleware/panic_recovery.go`（54 行，可单测）：

```go
func PanicRecovery() gin.HandlerFunc {
	// nil writer：关闭 gin 自带 recovery 的日志（它会 dump 请求头，debug 模式含 Cookie，
	// 且非 error 级、无 request id）；broken pipe 的识别与"仅 Abort"语义仍由 gin 保留。
	return gin.CustomRecoveryWithWriter(nil, handlePanic)
}

func handlePanic(c *gin.Context, recovered any) {
	logger.LogError(c.Request.Context(), fmt.Sprintf(
		"panic recovered: %v\nmethod=%s path=%s\n%s",
		recovered, c.Request.Method, c.Request.URL.Path, debug.Stack()))

	if c.Writer.Written() {
		c.Abort()   // 已写响应：不写 JSON、不改状态码
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{"message": PanicGenericMessage, "type": "new_api_panic"},
	})
}
```

要点：① 客户端只拿 `PanicGenericMessage`（通用文案），结构仍是 `{"error":{"message":...,"type":"new_api_panic"}}`；
② `logger.LogError` 走 **ERR 级**，自动带 context 里的 **request id**；消息含 **完整 `debug.Stack()`** + 方法/路径；
③ **不记录任何请求头** → Authorization/Cookie 永不落日志；④ `c.Writer.Written()` 为真只 Abort。

## 3. 测试：先失败 → 后通过（5 个用例）

新增 `middleware/panic_recovery_test.go`（149 行）。第一版实现**先用与 main.go 等价的旧逻辑**写出，跑出失败：

| 用例 | 断言要点 | 修复前 | 修复后 |
| --- | --- | --- | --- |
| ① `DoesNotLeakPanicDetail` | 500；body 不含敏感假串（`sk-live-DEADBEEF`、`/var/lib/new-api/secret.db`、`SELECT * FROM users`）；含 `"type":"new_api_panic"` + 通用文案 | **FAIL**：body 原样包含 `SENSITIVE_TOKEN=sk-live-DEADBEEF db=... sql=SELECT * ...` | **PASS** |
| ② `LogsErrorWithStackAndRequestContext` | 日志含 `[ERR]`、`goroutine `（debug.Stack 首行）、`runtime/debug.Stack`、方法 `GET`、路径 `/panic`、响应头里的 request id；**不含** Authorization/Cookie 值 | **FAIL**：只有 gin 自带 recovery 的日志（无 `[ERR]`/无 `goroutine `/无 method/无 request id） | **PASS** |
| ③ `KeepsRouterUsableAfterPanic` | panic→500 后 `/ok` 仍 200，再次 panic 仍 500 | PASS（既有能力，锁定不回归） | **PASS** |
| ④ `SkipsJSONWhenResponseAlreadyWritten` | 先写 `200 + "partial-body"` 再 panic：状态码仍 200、body **恰为** `partial-body`、不含 `new_api_panic`、无二次 panic | **FAIL**：body 变成 `partial-body{...new_api_panic...}`（P3 复现） | **PASS** |
| ⑤ `BrokenPipeDoesNotWriteBody` | `*net.OpError{... EPIPE}` panic：不写 body、不走 panic 处理分支、不二次 panic | PASS（既有语义，锁定不回归） | **PASS** |

修复前失败输出（节选，实际含完整 assertion 文本）：

```console
$ go test ./middleware/ -run TestPanicRecovery -v     # 旧逻辑
--- FAIL: TestPanicRecoveryDoesNotLeakPanicDetail
    "{\"error\":{\"message\":\"Panic detected, error: SENSITIVE_TOKEN=sk-live-DEADBEEF db=/var/lib/new-api/secret.db sql=SELECT * FROM users...\"}}" should not contain "sk-live-DEADBEEF"
--- FAIL: TestPanicRecoveryLogsErrorWithStackAndRequestContext
    "...[Recovery] ... panic recovered: SENSITIVE_TOKEN=...\n<stack>..." does not contain "[ERR]"
    ... does not contain "goroutine "
    ... does not contain "GET"
    ... does not contain "<request id>"
--- FAIL: TestPanicRecoverySkipsJSONWhenResponseAlreadyWritten
    expected: "partial-body"
    actual  : "partial-body{\"error\":{\"message\":\"Panic detected, error: panic after response written...\"}}"
FAIL
```

修复后：

```console
$ go test ./middleware/ -count=1 -run TestPanicRecovery -v
--- PASS: TestPanicRecoveryDoesNotLeakPanicDetail (0.00s)
--- PASS: TestPanicRecoveryLogsErrorWithStackAndRequestContext (0.00s)
--- PASS: TestPanicRecoveryKeepsRouterUsableAfterPanic (0.00s)
--- PASS: TestPanicRecoverySkipsJSONWhenResponseAlreadyWritten (0.00s)
--- PASS: TestPanicRecoveryBrokenPipeDoesNotWriteBody (0.00s)
ok  	github.com/QuantumNous/new-api/middleware	0.011s
```

> 说明（P2 的额外发现）：修复前若进程跑在 gin **debug 模式**（默认），gin 自带 recovery 会 dump
> 请求头，只遮蔽 `Authorization`、**Cookie 原样打印**；`CustomRecoveryWithWriter(nil, ...)` 关闭该日志后
> 只保留我们自己的脱敏日志（不含任何头）。

## 4. 日志样例（真实输出，脱敏后）

```console
$ go test ./middleware/ -run <临时用例，验证后已删除> -v
[ERR] 2026/10/24 - 06:21:57 | 202610240621577113978068268d9d6IBxEAGWX | panic recovered: sample failure (desensitized)
method=GET path=/panic
goroutine 50 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:26 +0x5e
github.com/QuantumNous/new-api/middleware.handlePanic(0x83718522400, {0x1651840, 0x19da380})
	/root/new-api-own/middleware/panic_recovery.go:40 +0x6b
github.com/gin-gonic/gin.CustomRecoveryWithWriter.func1.1()
	/root/go/pkg/mod/github.com/gin-gonic/gin@v1.9.1/recovery.go:98 +0x6ff
```

- 结构：`[ERR] 日期 | <request id> | panic recovered: <panic 值>` → `method=GET path=/panic` → 完整堆栈。
- 无任何请求头 → **Authorization/Cookie 不会落日志**。
- request id 由 `logger.LogError` 自动从 `c.Request.Context()` 读取（`RequestId` 中间件在内层已写入）。

## 5. 行为保持（要求 4）

| 契约 | 保障方式 | 测试 |
| --- | --- | --- |
| panic 后该请求 500 | 未写响应时 `AbortWithStatusJSON(500)` | ①③ |
| 进程不退出 | recovery 在独立 goroutine 中 absorb（gin 语义），测试进程正常跑完 | ③ |
| 后续请求正常 | 每个请求独立 `*gin.Context` | ③ |
| broken pipe 语义不变 | 仍用 gin 的 `CustomRecoveryWithWriter`（自带 `*net.OpError`/EPIPE 识别 → `c.Error` + `Abort`，不进 handler） | ⑤ |

## 6. 门禁（全部实际执行）

```console
$ go build ./...                              # OK
$ go vet ./...                                # OK
$ go test ./... -count=1
exit=0  FAIL=0  ok=36
$ gofmt -l .
（空，0 行）
$ cd relaykit && GOWORK=off go build ./...    # OK
```

## 7. 改动清单 / 风险 / 回滚

| 文件 | 改动 |
| --- | --- |
| `main.go` | 内联 `gin.CustomRecovery(handler)` → `server.Use(middleware.PanicRecovery())`（-9/+4） |
| `middleware/panic_recovery.go`（新增 54 行） | `PanicRecovery()` + `handlePanic()` + `PanicGenericMessage` 常量 |
| `middleware/panic_recovery_test.go`（新增 149 行） | 5 个用例（3 个先失败、2 个锁定既有语义） |
| `.trellis/tasks/10-24-panic-recovery-and-cleanup/evidence-recovery.md` | 本证据 |

- **风险**：低。仅 panic 路径的响应体与日志变化；正常请求路径零改动（recovery 只在 panic 时生效）。
  唯一可观察变化：500 响应的 `message` 从"含 panic 原文"变为通用文案（这是本任务的修复目标）；
  客户端仍能按 `type: "new_api_panic"` 与 `error.message` 字段解析（结构不变）。
- **回滚**：单文件 revert `main.go` + 删除两个新增 middleware 文件即可完全回到旧行为。
- **未做**（未在任务范围）：`PanicGenericMessage` 未做 i18n（原实现也是英文硬编码）；未改 `common.SysError` 签名。
