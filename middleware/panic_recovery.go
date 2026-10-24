package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"
)

const (
	// PanicGenericMessage 是返回给客户端的通用文案：**绝不包含 panic 原文**，
	// 避免把 SQL、密钥、内部路径等经 500 响应泄露出去。
	PanicGenericMessage = "Internal server error. Please contact the administrator if it persists."
	// panicErrorType 是既有错误结构里的稳定 type，前端/脚本按它识别 panic 响应。
	panicErrorType = "new_api_panic"
)

// PanicRecovery 捕获 handler panic：进程不退出，当前请求返回 500 + 通用文案，
// 服务端按 error 级记录 panic 值、方法/路径、request id 与完整堆栈。
//
// 实现要点：
//   - 用 gin.CustomRecoveryWithWriter(nil, ...) 而不是 gin.CustomRecovery：
//     后者会在 debug 模式下 dump 请求头（含 Cookie，仅遮蔽 Authorization），
//     且不是 error 级别、不带 request id；传 nil 只关闭它自带的日志，
//     **broken pipe 的识别与"仅 Abort 不写状态码"语义仍由 gin 保留**。
//   - 响应已写（SSE/流式）时只 Abort，不追加 JSON，避免脏响应。
//   - 日志不含任何请求头，因此 Authorization/Cookie 不会落盘。
func PanicRecovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, handlePanic)
}

// handlePanic 是 recovery 的处理分支（非 broken pipe 才会走到这里）。
func handlePanic(c *gin.Context, recovered any) {
	// logger.LogError 会写 error 级别，并自动带上 context 里的 request id（若 RequestId 中间件已执行）。
	logger.LogError(c.Request.Context(), fmt.Sprintf(
		"panic recovered: %v\nmethod=%s path=%s\n%s",
		recovered, c.Request.Method, c.Request.URL.Path, debug.Stack()))

	if c.Writer.Written() {
		// SSE/流式已写出：不能再写 JSON，也不能改写状态码，只终止后续 handler。
		c.Abort()
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{
			"message": PanicGenericMessage,
			"type":    panicErrorType,
		},
	})
}
