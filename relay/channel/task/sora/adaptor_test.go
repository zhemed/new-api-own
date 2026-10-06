package sora

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestEstimateBillingSecondsBound guards the billing invariant that the
// "seconds" OtherRatio this adaptor hands to quota math can never exceed
// MaxTaskDurationSeconds. The entry validator already rejects such input; this
// is the local backstop required for every adaptor that reads a multiplier.
func TestEstimateBillingSecondsBound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		req         relaycommon.TaskSubmitReq
		wantSeconds float64
	}{
		{
			name:        "seconds within the bound is used as-is",
			req:         relaycommon.TaskSubmitReq{Model: "sora-2", Prompt: "a cat", Seconds: "8"},
			wantSeconds: 8,
		},
		{
			// 只有该字段被 sora 计费优先读取，超界值时不得原样进入配额计算。
			name:        "oversized seconds is clamped to the bound",
			req:         relaycommon.TaskSubmitReq{Model: "sora-2", Prompt: "a cat", Duration: 4, Seconds: "100000"},
			wantSeconds: float64(relaycommon.MaxTaskDurationSeconds),
		},
		{
			name:        "missing duration falls back to the default",
			req:         relaycommon.TaskSubmitReq{Model: "sora-2", Prompt: "a cat"},
			wantSeconds: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set("task_request", tt.req)
			info := &relaycommon.RelayInfo{
				TaskRelayInfo: &relaycommon.TaskRelayInfo{Action: constant.TaskActionTextGenerate},
			}

			ratios := (&TaskAdaptor{}).EstimateBilling(c, info)

			require.Equal(t, tt.wantSeconds, ratios["seconds"])
		})
	}
}
