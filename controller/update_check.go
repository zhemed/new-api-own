package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// updateActivationDelay 给响应留出写回客户端的时间：原地重执行会替换进程镜像，
// 之后本进程无法再写任何响应。
const updateActivationDelay = 500 * time.Millisecond

// GetUpdateStatus 返回服务端更新检查结果（功能未开启时返回 disabled，绝不外呼）。
func GetUpdateStatus(c *gin.Context) {
	common.ApiSuccess(c, service.GetUpdateStatus(c.Request.Context()))
}

// ApplyUpdate 由管理员触发：下载并校验目标版本二进制，通过后原子替换本机文件，
// 再把响应写回，最后原地重执行让新版本生效。
//
// 注意：替换的是运行中容器内的二进制，下一次 `docker rm` / `docker run` 会回到镜像内的
// 版本；长期升级仍以重建容器（新镜像）为准。
func ApplyUpdate(c *gin.Context) {
	result, err := service.ApplyLatestUpdate(c.Request.Context())
	if err != nil {
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
