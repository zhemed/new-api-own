package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func TestGetAccountUsage(t *testing.T) {
	db := openTokenControllerTestDB(t)
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate user table: %v", err)
	}

	user := &model.User{
		Id:        1,
		Username:  "account-user",
		Password:  "password123",
		Status:    common.UserStatusEnabled,
		Quota:     123456,
		UsedQuota: 654,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/usage/account/", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Set("id", user.Id)

	GetAccountUsage(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["success"] != true {
		t.Fatal("expected success=true")
	}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object, got %#v", response["data"])
	}
	if data["quota"] != float64(user.Quota) || data["used_quota"] != float64(user.UsedQuota) {
		t.Fatalf("unexpected quota data: %#v", data)
	}
	if data["total_quota"] != float64(user.Quota+user.UsedQuota) {
		t.Fatalf("unexpected total quota: %#v", data["total_quota"])
	}
	if data["quota_per_unit"] != common.QuotaPerUnit || data["unit"] != "USD" {
		t.Fatalf("unexpected display metadata: %#v", data)
	}
}

func TestGetAccountUsageRequiresUserContext(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/usage/account/", nil)

	GetAccountUsage(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
