package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestGetLogFilesReportsMemoryLogBudget pins the panel contract for the database
// log payload usage: the three read-only fields are always present (even when
// file logging is off), the budget is 0 when unset and the configured value
// otherwise. No outbound call is involved.
func TestGetLogFilesReportsMemoryLogBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func() (*gin.Context, *httptest.ResponseRecorder) {
		previousLogDir := *common.LogDir
		*common.LogDir = t.TempDir()
		t.Cleanup(func() { *common.LogDir = previousLogDir })

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/performance/logs", nil)
		return c, recorder
	}

	t.Run("budget unset reports zero", func(t *testing.T) {
		t.Setenv("LOG_MEMORY_MAX_BYTES", "")

		c, recorder := newContext()
		GetLogFiles(c)

		require.Equal(t, http.StatusOK, recorder.Code)
		body := recorder.Body.String()
		require.Contains(t, body, `"memory_log_bytes":0`)
		require.Contains(t, body, `"memory_log_max_bytes":0`)
		require.Contains(t, body, `"memory_log_rows":0`)
	})

	t.Run("configured budget is echoed in bytes", func(t *testing.T) {
		t.Setenv("LOG_MEMORY_MAX_BYTES", "200MB")

		c, recorder := newContext()
		GetLogFiles(c)

		require.Equal(t, http.StatusOK, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"memory_log_max_bytes":209715200`)
	})
}
