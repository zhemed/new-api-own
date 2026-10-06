package controller

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// setUpdateCheckOption 走生产同款的选项更新路径切换开关（不重启即时生效）。
func setUpdateCheckOption(t *testing.T, checkEnabled bool, applyEnabled bool) {
	t.Helper()
	configObject := config.GlobalConfig.Get("update_check_setting")
	require.NotNil(t, configObject)
	require.NoError(t, config.UpdateConfigFromMap(configObject, map[string]string{
		"enabled":       strconv.FormatBool(checkEnabled),
		"apply_enabled": strconv.FormatBool(applyEnabled),
	}))
	t.Cleanup(func() {
		_ = config.UpdateConfigFromMap(configObject, map[string]string{
			"enabled":       "false",
			"apply_enabled": "true",
		})
	})
}

// TestGetUpdateStatusEndpointReportsDisabledWithoutFetching 断言关闭语义：
// 面板拿到的三态是中性值（latest 为空 / has_update=null），开关字段如实回填，且不外呼。
func TestGetUpdateStatusEndpointReportsDisabledWithoutFetching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setUpdateCheckOption(t, false, true)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/status/update", nil)

	GetUpdateStatus(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, `"enabled":false`)
	require.Contains(t, body, `"check_enabled":false`)
	// 检查关闭不代表"立即更新"被禁用：应用开关独立回填，前端不必再去读 /api/option/。
	require.Contains(t, body, `"apply_enabled":true`)
	require.Contains(t, body, `"state":"disabled"`)
	require.Contains(t, body, `"latest_version":""`)
	require.Contains(t, body, `"has_update":null`)
}

func TestApplyUpdateEndpointRejectsWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setUpdateCheckOption(t, true, false)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/status/update/apply", nil)

	ApplyUpdate(c)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	body := recorder.Body.String()
	require.Contains(t, body, `"success":false`)
	require.Contains(t, body, `"code":"disabled"`)
}
