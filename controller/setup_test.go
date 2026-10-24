package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupOperationModesTest 准备隔离环境：内存 SQLite（options/users/setups 三表）、
// 已存在的 root 用户（跳过建号分支，聚焦运营模式写入），并保存/恢复全部全局状态。
func setupOperationModesTest(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)

	previousDB := model.DB
	previousMainType := common.MainDatabaseType()
	previousOptionMap := common.OptionMap
	previousSelfUse := operation_setting.SelfUseModeEnabled
	previousDemoSite := operation_setting.DemoSiteEnabled
	previousSetup := constant.Setup

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.User{}, &model.Setup{}, &model.Token{}, &model.Redemption{}))
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)

	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	// 旧值：两个开关都是 false；初始化流程尚未完成。
	operation_setting.SelfUseModeEnabled = false
	operation_setting.DemoSiteEnabled = false
	constant.Setup = false

	// 已存在 root 用户 → PostSetup 跳过账号创建，只走运营模式写入路径。
	require.NoError(t, db.Create(&model.User{
		Username: "root",
		Role:     common.RoleRootUser,
		Status:   common.UserStatusEnabled,
		Quota:    100,
	}).Error)
	require.True(t, model.RootUserExists())

	t.Cleanup(func() {
		model.DB = previousDB
		common.SetMainDatabaseType(previousMainType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
		operation_setting.SelfUseModeEnabled = previousSelfUse
		operation_setting.DemoSiteEnabled = previousDemoSite
		constant.Setup = previousSetup
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

// postSetupRequest 发送一次初始化请求（两个开关都置 true），返回响应体原文。
func postSetupRequest(t *testing.T) string {
	t.Helper()
	body := bytes.NewBufferString(`{"SelfUseModeEnabled":true,"DemoSiteEnabled":true}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/setup", body)
	c.Request.Header.Set("Content-Type", "application/json")

	PostSetup(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	raw := recorder.Body.String()
	require.Contains(t, raw, `"success"`, "响应必须带 success 字段")
	return raw
}

func optionRowCount(t *testing.T, db *gorm.DB, key string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&model.Option{}).Where("key = ?", key).Count(&count).Error)
	return count
}

// TestPostSetupWriteFailureKeepsMemoryAndDBUntouched 是本缺陷的核心回归：
// 写库失败时接口必须报失败，且**内存两个开关保持旧值**、DB 未写入。
// 修复前 setup.go 在落库前就改了内存 → 失败后内存已是新值（DB/内存分叉）。
func TestPostSetupWriteFailureKeepsMemoryAndDBUntouched(t *testing.T) {
	db := setupOperationModesTest(t)

	// 让 options 表的 INSERT 必然失败：两个键都写不进去。
	require.NoError(t, db.Exec(
		`CREATE TRIGGER setup_option_insert_must_fail BEFORE INSERT ON options
		 BEGIN SELECT RAISE(ABORT, 'forced option insert failure'); END;`).Error)

	raw := postSetupRequest(t)

	require.Contains(t, raw, `"success":false`, "写库失败必须如实返回失败：%s", raw)
	assert.Contains(t, raw, "失败", "失败文案应说明保存失败：%s", raw)

	// 核心断言：内存必须保持旧值（false），不得被请求值（true）污染。
	assert.False(t, operation_setting.SelfUseModeEnabled,
		"写库失败后 SelfUseModeEnabled 必须保持旧值，不得在落库前就被改")
	assert.False(t, operation_setting.DemoSiteEnabled,
		"写库失败后 DemoSiteEnabled 必须保持旧值，不得在落库前就被改")

	// DB 未写入。
	assert.Zero(t, optionRowCount(t, db, "SelfUseModeEnabled"), "写库失败时不得留下 SelfUseModeEnabled 记录")
	assert.Zero(t, optionRowCount(t, db, "DemoSiteEnabled"), "写库失败时不得留下 DemoSiteEnabled 记录")
	// 初始化未被标记完成。
	assert.False(t, constant.Setup, "写库失败时不得把实例标记为已初始化")
}

// TestPostSetupSuccessSyncsMemoryAndDB：成功路径行为不变——内存与 DB 一致、初始化完成。
func TestPostSetupSuccessSyncsMemoryAndDB(t *testing.T) {
	db := setupOperationModesTest(t)

	raw := postSetupRequest(t)

	require.Contains(t, raw, `"success":true`, "正常路径必须成功：%s", raw)
	assert.True(t, operation_setting.SelfUseModeEnabled, "成功后内存应为请求值")
	assert.True(t, operation_setting.DemoSiteEnabled, "成功后内存应为请求值")

	stored, err := modelOptionValue(db, "SelfUseModeEnabled")
	require.NoError(t, err)
	assert.Equal(t, "true", stored, "成功后 DB 应为请求值")
	stored, err = modelOptionValue(db, "DemoSiteEnabled")
	require.NoError(t, err)
	assert.Equal(t, "true", stored, "成功后 DB 应为请求值")

	assert.True(t, constant.Setup, "成功后应标记初始化完成")
	assert.Positive(t, setupRowCount(t, db), "成功后应写入 setups 记录")
}

func modelOptionValue(db *gorm.DB, key string) (string, error) {
	var option model.Option
	if err := db.Where("key = ?", key).First(&option).Error; err != nil {
		return "", err
	}
	return option.Value, nil
}

// TestPostSetupWritesOperationModesAtomically 证明两个键是**原子**写入：
// 只让第二个键（DemoSiteEnabled）写入失败 → 第一个键也不得落库（不能半应用）。
// 若退回"两次独立 UpdateOption"，旧实现会先写成功 SelfUseModeEnabled 再失败，留下半应用状态。
func TestPostSetupWritesOperationModesAtomically(t *testing.T) {
	db := setupOperationModesTest(t)

	require.NoError(t, db.Exec(
		`CREATE TRIGGER setup_demo_insert_must_fail BEFORE INSERT ON options
		 WHEN NEW.key = 'DemoSiteEnabled'
		 BEGIN SELECT RAISE(ABORT, 'forced DemoSiteEnabled insert failure'); END;`).Error)

	raw := postSetupRequest(t)

	require.Contains(t, raw, `"success":false`, "任一键写入失败必须整体失败：%s", raw)
	assert.Zero(t, optionRowCount(t, db, "SelfUseModeEnabled"),
		"原子写入：另一个键失败时 SelfUseModeEnabled 也不得落库（不得半应用）")
	assert.Zero(t, optionRowCount(t, db, "DemoSiteEnabled"))
	// 内存同样两个都不得变。
	assert.False(t, operation_setting.SelfUseModeEnabled)
	assert.False(t, operation_setting.DemoSiteEnabled)
	assert.False(t, constant.Setup, "失败时不得标记初始化完成")
}

// TestPostSetupCreateFailureDoesNotMarkInitialized 覆盖同函数内的同类顺序问题：
// `constant.Setup` 必须在 setups 记录**落库成功之后**才置 true。
// 否则 Create 失败时内存已显示"已初始化"，重试会被「系统已经初始化完成」挡住，而 DB 无记录。
func TestPostSetupCreateFailureDoesNotMarkInitialized(t *testing.T) {
	db := setupOperationModesTest(t)

	// 让 setups 表的 INSERT 必然失败（options 两张表写入正常成功，精确定位到 Create(&setup)）。
	require.NoError(t, db.Exec(
		`CREATE TRIGGER setup_record_insert_must_fail BEFORE INSERT ON setups
		 BEGIN SELECT RAISE(ABORT, 'forced setup record insert failure'); END;`).Error)

	raw := postSetupRequest(t)

	require.Contains(t, raw, `"success":false`, "初始化记录写库失败必须如实返回失败：%s", raw)
	require.Contains(t, raw, "系统初始化失败", "失败文案应指向初始化失败：%s", raw)
	assert.False(t, constant.Setup, "Create 失败时不得把实例标记为已初始化")
	assert.Zero(t, setupRowCount(t, db), "Create 失败时不得留下 setups 记录")

	// 关键：故障排除后必须仍允许重试，不能被「系统已经初始化完成」挡住。
	require.NoError(t, db.Exec(`DROP TRIGGER setup_record_insert_must_fail`).Error)
	retryRaw := postSetupRequest(t)
	require.NotContains(t, retryRaw, "系统已经初始化完成",
		"Create 失败后必须允许重试，不得因内存已标记而永久卡住：%s", retryRaw)
	require.Contains(t, retryRaw, `"success":true`, "故障排除后重试应成功：%s", retryRaw)
	assert.True(t, constant.Setup, "重试成功后应标记初始化完成")
	assert.Equal(t, int64(1), setupRowCount(t, db), "重试成功后应有且仅有一条 setups 记录")
}

func setupRowCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&model.Setup{}).Count(&count).Error)
	return count
}
