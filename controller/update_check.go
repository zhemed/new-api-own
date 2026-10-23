package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// updateActivationDelay 给响应留出写回客户端的时间：原地重执行会替换进程镜像，
// 之后本进程无法再写任何响应。
const updateActivationDelay = 500 * time.Millisecond

// GetUpdateStatus 返回服务端更新检查结果（功能未开启时返回 disabled，绝不外呼）。
// 自更新属于 root 级能力：这里额外拒绝用系统访问令牌（PAT）发起的请求。
func GetUpdateStatus(c *gin.Context) {
	if !middleware.RequireSessionCredential(c) {
		return
	}
	common.ApiSuccess(c, service.GetUpdateStatus(c.Request.Context()))
}

// ApplyUpdate 由管理员触发：下载并校验目标版本二进制，通过后原子替换本机文件，
// 再把响应写回，最后原地重执行让新版本生效。
//
// 校验顺序：SHA256SUMS 签名（ed25519，缺失/不过关即拒绝）→ 二进制 SHA256 → 生成回滚点 → 原子替换。
// 任何一步失败都不会改动现有二进制。
//
// 注意：替换的是运行中容器内的二进制，下一次 `docker rm` / `docker run` 会回到镜像内的
// 版本；长期升级仍以重建容器（新镜像）为准。
func ApplyUpdate(c *gin.Context) {
	if !middleware.RequireSessionCredential(c) {
		return
	}
	result, err := service.ApplyLatestUpdate(c.Request.Context())
	if err != nil {
		writeUpdateApplyError(c, err)
		return
	}

	common.ApiSuccess(c, result)
	c.Writer.Flush()

	// 响应写完后再生效：优先原地重执行（无需外部 supervisor，也不依赖 restart 策略），
	// exec 不可用时 Activate 会退出进程交给容器 restart 策略。
	go func() {
		time.Sleep(updateActivationDelay)
		service.ActivateLatestUpdate(result)
	}()
}

// RollbackUpdate 由管理员触发：仅用本机回滚点 <exe>.prev 原子换回上一个版本（不下载旧版本）。
// 回滚失败时当前二进制保持原样；回滚点会被消费，需要再次更新才会重新生成。
func RollbackUpdate(c *gin.Context) {
	if !middleware.RequireSessionCredential(c) {
		return
	}
	result, err := service.RollbackLatestUpdate()
	if err != nil {
		writeUpdateApplyError(c, err)
		return
	}

	common.ApiSuccess(c, result)
	c.Writer.Flush()

	go func() {
		time.Sleep(updateActivationDelay)
		service.ActivateLatestRollback(result)
	}()
}

// writeUpdateApplyError 统一把 UpdateApplyError 映射成稳定 code + 状态码；其它错误回落 500。
func writeUpdateApplyError(c *gin.Context, err error) {
	var applyErr *service.UpdateApplyError
	if errors.As(err, &applyErr) {
		c.JSON(applyErr.HTTPStatus, gin.H{
			"success": false,
			"message": applyErr.Message,
			"code":    applyErr.Code,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to apply update"})
}
