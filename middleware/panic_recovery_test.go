package middleware

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sensitivePanicPayload 模拟可能出现在 panic 原文里的敏感信息（DB 路径/密钥/SQL 片段）。
// 断言：它必须只进服务端日志，绝不能出现在客户端响应里。
const sensitivePanicPayload = "SENSITIVE_TOKEN=sk-live-DEADBEEF db=/var/lib/new-api/secret.db sql=SELECT * FROM users"

// newRecoveryTestEngine 构造与生产同序的最小路由：PanicRecovery → RequestId → 业务路由。
// 同时把 gin 的 writer 换到内存 buffer，便于断言日志内容。
func newRecoveryTestEngine(t *testing.T) (*gin.Engine, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	errBuf, outBuf := &bytes.Buffer{}, &bytes.Buffer{}
	common.LogWriterMu.Lock()
	prevErrWriter, prevOutWriter := gin.DefaultErrorWriter, gin.DefaultWriter
	gin.DefaultErrorWriter, gin.DefaultWriter = errBuf, outBuf
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter, gin.DefaultWriter = prevErrWriter, prevOutWriter
		common.LogWriterMu.Unlock()
	})

	engine := gin.New()
	engine.Use(PanicRecovery())
	engine.Use(RequestId())
	engine.GET("/panic", func(c *gin.Context) {
		panic(sensitivePanicPayload)
	})
	engine.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	engine.GET("/partial", func(c *gin.Context) {
		// 模拟 SSE/流式：响应已写出后再 panic
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString("partial-body")
		panic("panic after response written")
	})
	engine.GET("/brokenpipe", func(c *gin.Context) {
		// gin 的 recovery 会把 broken pipe 当作连接断开：只 Abort，不写状态码/JSON
		panic(&net.OpError{Op: "write", Net: "tcp",
			Err: &os.SyscallError{Syscall: "write", Err: syscall.EPIPE}})
	})
	return engine, errBuf, outBuf
}

func serveTestRequest(engine *gin.Engine, method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	engine.ServeHTTP(recorder, req)
	return recorder
}

// ① panic 原文绝不回显给客户端；响应仍是 500 + 既有错误结构 + 通用文案。
func TestPanicRecoveryDoesNotLeakPanicDetail(t *testing.T) {
	engine, _, _ := newRecoveryTestEngine(t)

	recorder := serveTestRequest(engine, http.MethodGet, "/panic")

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	body := recorder.Body.String()
	assert.NotContains(t, body, "sk-live-DEADBEEF", "响应不得包含密钥")
	assert.NotContains(t, body, "/var/lib/new-api/secret.db", "响应不得包含内部路径")
	assert.NotContains(t, body, "SELECT * FROM users", "响应不得包含 SQL")
	assert.NotContains(t, body, sensitivePanicPayload, "响应不得包含 panic 原文")
	assert.Contains(t, body, `"type":"new_api_panic"`, "必须保持既有错误结构")
	assert.Contains(t, body, PanicGenericMessage, "必须返回通用文案")
}

// ② 服务端 error 级日志：含完整堆栈、方法/路径、request id；不打印 Authorization/Cookie。
func TestPanicRecoveryLogsErrorWithStackAndRequestContext(t *testing.T) {
	engine, errBuf, _ := newRecoveryTestEngine(t)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic?foo=bar", nil)
	req.Header.Set("Authorization", "Bearer super-secret-authorization-token")
	req.Header.Set("Cookie", "session=super-secret-cookie-value")
	engine.ServeHTTP(recorder, req)

	logged := errBuf.String()
	assert.Contains(t, logged, "[ERR]", "panic 必须按 error 级记录")
	assert.Contains(t, logged, "goroutine ", "日志必须包含完整堆栈（debug.Stack 以 goroutine 开头）")
	assert.Contains(t, logged, "runtime/debug.Stack", "堆栈应指向 panic recovery 的采集点")
	assert.Contains(t, logged, "/panic", "日志必须包含请求路径")
	assert.Contains(t, logged, "GET", "日志必须包含请求方法")
	requestID := recorder.Header().Get(common.RequestIdKey)
	require.NotEmpty(t, requestID, "RequestId 中间件应写入响应头")
	assert.Contains(t, logged, requestID, "日志必须包含 request id")
	assert.NotContains(t, logged, "super-secret-authorization-token", "日志不得包含 Authorization")
	assert.NotContains(t, logged, "super-secret-cookie-value", "日志不得包含 Cookie")
}

// ③ panic 之后进程/路由仍可用：后续正常请求照常 200，再来一次 panic 仍能 500。
func TestPanicRecoveryKeepsRouterUsableAfterPanic(t *testing.T) {
	engine, _, _ := newRecoveryTestEngine(t)

	first := serveTestRequest(engine, http.MethodGet, "/panic")
	require.Equal(t, http.StatusInternalServerError, first.Code)

	ok := serveTestRequest(engine, http.MethodGet, "/ok")
	assert.Equal(t, http.StatusOK, ok.Code)
	assert.Equal(t, "ok", ok.Body.String())

	second := serveTestRequest(engine, http.MethodGet, "/panic")
	assert.Equal(t, http.StatusInternalServerError, second.Code, "再次 panic 仍应被兜住")
}

// ④ 响应已写（SSE/流式）时 panic：不得追加 JSON、不得改写已发出的状态码，也不得二次 panic。
func TestPanicRecoverySkipsJSONWhenResponseAlreadyWritten(t *testing.T) {
	engine, _, _ := newRecoveryTestEngine(t)

	var recorder *httptest.ResponseRecorder
	require.NotPanics(t, func() {
		recorder = serveTestRequest(engine, http.MethodGet, "/partial")
	}, "已写响应时 panic 不应导致二次 panic")

	require.NotNil(t, recorder)
	assert.Equal(t, http.StatusOK, recorder.Code, "已发出的状态码不得被改写")
	assert.Equal(t, "partial-body", recorder.Body.String(), "不得在已写响应后追加 JSON")
	assert.NotContains(t, recorder.Body.String(), "new_api_panic")
}

// ⑤ broken pipe 语义不变：仅 Abort，不写状态码也不写 JSON（由 gin 内部识别，不进 panic 处理分支）。
func TestPanicRecoveryBrokenPipeDoesNotWriteBody(t *testing.T) {
	engine, errBuf, _ := newRecoveryTestEngine(t)

	var recorder *httptest.ResponseRecorder
	require.NotPanics(t, func() {
		recorder = serveTestRequest(engine, http.MethodGet, "/brokenpipe")
	})
	require.NotNil(t, recorder)
	assert.Empty(t, recorder.Body.String(), "broken pipe 不得写入 JSON 响应体")
	assert.NotContains(t, errBuf.String(), "panic recovered", "broken pipe 不应走 panic 处理分支")
}
