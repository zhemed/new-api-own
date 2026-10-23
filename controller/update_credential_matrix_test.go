package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupUpdateCredentialTest 提供最小 DB（users + user_sessions + logs），用于跑真实鉴权链路。
// 说明：SQLite 的 ":memory:" 是"每连接一个库"，必须把连接池钉成单连接，否则并发请求会打到
// 没有建表的连接上；审计链路会写 logs 表，所以 LOG_DB 也要指向同一个测试库。
func setupUpdateCredentialTest(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousType := common.MainDatabaseType()
	previousRedis := common.RedisEnabled
	previousSecret := common.SessionSecret

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Log{}))
	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.SessionSecret = "update-credential-matrix-secret"
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSecret
		_ = sqlDB.Close()
	})
}

// createUpdateTestCredential 建一个用户并返回可用的 Authorization 值：
// pat=false 走"登录会话"（user_sessions + dashboard access token），pat=true 走系统访问令牌（PAT）。
func createUpdateTestCredential(t *testing.T, username string, role int, pat bool) string {
	t.Helper()
	user := &model.User{
		Username: username, Password: "password-placeholder", Role: role,
		Status: common.UserStatusEnabled, Group: "default", AuthVersion: 1,
		AffCode: "update-matrix-" + username,
	}
	if pat {
		token := "pat-" + username
		user.AccessToken = &token
	}
	require.NoError(t, model.DB.Create(user).Error)
	if pat {
		return "pat-" + username
	}

	session := model.UserSession{
		SID: "sid-" + username, UserID: user.Id, Version: 1, UserAuthVersion: 1,
		Status: model.UserSessionStatusActive, LoginMethod: "password",
		RefreshHash:  strings.Repeat("b", 64),
		LastActiveAt: time.Now().Unix(),
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, model.DB.Create(&session).Error)
	accessToken, _, err := service.IssueAccessToken(service.AuthIdentity{
		UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1,
	})
	require.NoError(t, err)
	return accessToken
}

func newUpdateCredentialRouter() *gin.Engine {
	router := gin.New()
	router.GET("/api/status/update", middleware.RootAuth(), GetUpdateStatus)
	router.POST("/api/status/update/apply", middleware.RootAuth(), ApplyUpdate)
	router.POST("/api/status/update/rollback", middleware.RootAuth(), RollbackUpdate)
	return router
}

func performUpdateRequest(router *gin.Engine, method string, path string, credential string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	if credential != "" {
		request.Header.Set("Authorization", "Bearer "+credential)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func requireErrorCode(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, response.Code, "响应体: %s", response.Body.String())
	require.Contains(t, response.Body.String(), `"code":"`+code+`"`, "响应体: %s", response.Body.String())
}

// TestUpdateEndpointsCredentialMatrix 是自更新的权限矩阵：
//   - root 会话：允许（进入业务逻辑）；
//   - 普通 admin 会话：被 RootAuth 拒绝（AUTH_INSUFFICIENT_PRIVILEGE）；
//   - root 的系统访问令牌（PAT）：被 handler 的凭据检查拒绝（pat_not_allowed）。
//
// 开关全部关闭：任何"通过鉴权"的请求都只会拿到 disabled，绝不会外呼或替换二进制。
func TestUpdateEndpointsCredentialMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupUpdateCredentialTest(t)
	setUpdateCheckOption(t, false, false)

	rootSession := createUpdateTestCredential(t, "matrix-root", common.RoleRootUser, false)
	adminSession := createUpdateTestCredential(t, "matrix-admin", common.RoleAdminUser, false)
	rootPAT := createUpdateTestCredential(t, "matrix-root-pat", common.RoleRootUser, true)

	router := newUpdateCredentialRouter()
	endpoints := []struct {
		name          string
		method        string
		path          string
		allowedStatus int
		allowedCode   string
	}{
		{name: "status", method: http.MethodGet, path: "/api/status/update", allowedStatus: http.StatusOK},
		{name: "apply", method: http.MethodPost, path: "/api/status/update/apply", allowedStatus: http.StatusForbidden, allowedCode: "disabled"},
		{name: "rollback", method: http.MethodPost, path: "/api/status/update/rollback", allowedStatus: http.StatusForbidden, allowedCode: "disabled"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name+"/root session is allowed", func(t *testing.T) {
			response := performUpdateRequest(router, endpoint.method, endpoint.path, rootSession)
			require.Equal(t, endpoint.allowedStatus, response.Code, "响应体: %s", response.Body.String())
			if endpoint.allowedCode != "" {
				require.Contains(t, response.Body.String(), `"code":"`+endpoint.allowedCode+`"`)
			}
		})

		t.Run(endpoint.name+"/admin session is rejected by RootAuth", func(t *testing.T) {
			response := performUpdateRequest(router, endpoint.method, endpoint.path, adminSession)
			requireErrorCode(t, response, http.StatusForbidden, "AUTH_INSUFFICIENT_PRIVILEGE")
		})

		t.Run(endpoint.name+"/root PAT is rejected", func(t *testing.T) {
			response := performUpdateRequest(router, endpoint.method, endpoint.path, rootPAT)
			requireErrorCode(t, response, http.StatusForbidden, middleware.CodePATNotAllowed)
		})

		t.Run(endpoint.name+"/anonymous is rejected", func(t *testing.T) {
			response := performUpdateRequest(router, endpoint.method, endpoint.path, "")
			requireErrorCode(t, response, http.StatusUnauthorized, "AUTH_UNAUTHORIZED")
		})
	}
}
