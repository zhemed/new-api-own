package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// GetAccountUsage returns the authenticated relay token owner's account quota.
// It deliberately ignores token-level unlimited quota and upstream balances.
func GetAccountUsage(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "token user is not authenticated",
		})
		return
	}

	remainingQuota, err := model.GetUserQuota(userID, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	usedQuota, err := model.GetUserUsedQuota(userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ok",
		"data": gin.H{
			"object":         "account_usage",
			"quota":          remainingQuota,
			"used_quota":     usedQuota,
			"total_quota":    remainingQuota + usedQuota,
			"quota_per_unit": common.QuotaPerUnit,
			"unit":           "USD",
		},
	})
}
