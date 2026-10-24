package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 回归对照的两个默认模板：代码默认表里它们必须逐字符一致
// （deepseek 渠道 ModelList 与四张默认表都引用这两个名字）。
const (
	regressionModelRef   = "deepseek-flash"
	regressionModelDrift = "deepseek-v4.1-flash"
)

// setupPricingResetTest 准备隔离的测试环境：内存 SQLite + 选项表 + 已初始化的
// 倍率/计费默认（生产由 main 启动流程完成，这里显式做一遍）。
func setupPricingResetTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	originalDB := model.DB
	originalSQLitePath := common.SQLitePath
	originalMainType := common.MainDatabaseType()
	originalLogType := common.LogDatabaseType()
	originalIsMasterNode := common.IsMasterNode
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")

	t.Cleanup(func() {
		model.DB = originalDB
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainType, originalLogType)
		common.IsMasterNode = originalIsMasterNode
		if hadSQLDSN {
			_ = os.Setenv("SQL_DSN", originalSQLDSN)
		} else {
			_ = os.Unsetenv("SQL_DSN")
		}
	})

	// IsMasterNode=true 才会跑迁移建表（options 表是 model.UpdateOption 的前提）。
	common.IsMasterNode = true
	common.SQLitePath = fmt.Sprintf("file:%s?mode=memory&cache=shared",
		strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))
	require.NoError(t, model.InitDB())
	t.Cleanup(func() {
		if model.DB != nil {
			if sqlDB, err := model.DB.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
	})

	model.InitOptionMap()
	ratio_setting.InitRatioSettings()
}

// storedOption 读回某项落库的原始值（重置是否真的写到 DB，而不是只改了内存）。
func storedOption(t *testing.T, key string) string {
	t.Helper()
	var option model.Option
	require.NoError(t, model.DB.Where("key = ?", key).First(&option).Error, "选项 %s 应已落库", key)
	return option.Value
}

// resetModelRatioRequest 调用重置接口并返回响应体（接口始终 HTTP 200，业务结果看 success）。
func resetModelRatioRequest(t *testing.T) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/option/rest_model_ratio", nil)

	ResetModelRatio(c)

	require.Equal(t, http.StatusOK, recorder.Code, "接口路径与响应结构必须保持兼容（HTTP 200）")
	return recorder.Body.String()
}

func floatMapFromJSON(t *testing.T, raw string) map[string]float64 {
	t.Helper()
	parsed := map[string]float64{}
	require.NoError(t, common.UnmarshalJsonStr(raw, &parsed))
	return parsed
}

// TestResetModelRatioRestoresFullPricingDefaults 是本缺陷的回归用例：
// 实例里 deepseek-v4.1-flash 的计费表达式/模式被删掉后，「重置」必须把**整份定价**
// （倍率 + 缓存 + 价格 + 计费模式 + 计费表达式）恢复成代码默认，使两个默认模板逐字符一致；
// 同时不得顺带改动分组类键。
func TestResetModelRatioRestoresFullPricingDefaults(t *testing.T) {
	setupPricingResetTest(t)

	// 组内哨兵：重置前后都必须原样保留（分组倍率不属按钮语义）。
	sentinelGroupRatio := `{"default":1,"vip":7.5,"sentinel_group":3.25}`
	require.NoError(t, model.UpdateOption("GroupRatio", sentinelGroupRatio))
	require.NoError(t, model.UpdateOption("TopupGroupRatio", `{"default":1,"sentinel_topup":9.5}`))

	// 构造实例状态：drift 模型缺表达式、模式退化成 ratio。
	driftModes, err := common.Marshal(map[string]string{regressionModelDrift: billing_setting.BillingModeRatio})
	require.NoError(t, err)
	driftExprs, err := common.Marshal(map[string]string{regressionModelDrift: ""})
	require.NoError(t, err)
	require.NoError(t, model.UpdateOption("billing_setting."+billing_setting.BillingModeField, string(driftModes)))
	require.NoError(t, model.UpdateOption("billing_setting."+billing_setting.BillingExprField, string(driftExprs)))

	// 倍率表也弄脏：重置覆盖的必须是「整份」，不只是计费字段。
	dirtyRatios, err := common.Marshal(map[string]float64{regressionModelDrift: 99.5, "stale-model": 12.5})
	require.NoError(t, err)
	require.NoError(t, model.UpdateOption("ModelRatio", string(dirtyRatios)))

	// 前置条件成立才谈得上"修复后变一致"。
	require.Equal(t, billing_setting.BillingModeRatio, billing_setting.GetBillingMode(regressionModelDrift))
	preconditionExpr, _ := billing_setting.GetBillingExpr(regressionModelDrift)
	require.Empty(t, strings.TrimSpace(preconditionExpr), "前置条件：drift 模型此刻没有计费表达式")
	require.Equal(t, 99.5, ratio_setting.GetModelRatioCopy()[regressionModelDrift])

	body := resetModelRatioRequest(t)
	require.Contains(t, body, `"success":true`, "重置必须成功：%s", body)

	// 1) 本缺陷的直接症状：计费模式与表达式回到代码默认，两个默认模板逐字符相同。
	defaultModes := billing_setting.DefaultBillingMode2JSONString()
	defaultExprs := billing_setting.DefaultBillingExpr2JSONString()
	storedModes := storedOption(t, "billing_setting."+billing_setting.BillingModeField)
	storedExprs := storedOption(t, "billing_setting."+billing_setting.BillingExprField)
	assert.JSONEq(t, defaultModes, storedModes, "计费模式必须回到代码默认")
	assert.JSONEq(t, defaultExprs, storedExprs, "计费表达式必须回到代码默认")

	require.Equal(t, billing_setting.BillingModeTieredExpr, billing_setting.GetBillingMode(regressionModelDrift),
		"drift 模型必须恢复为表达式计费（这正是本缺陷的症状）")
	refExpr, ok := billing_setting.GetBillingExpr(regressionModelRef)
	require.True(t, ok)
	driftExpr, ok := billing_setting.GetBillingExpr(regressionModelDrift)
	require.True(t, ok)
	assert.Equal(t, refExpr, driftExpr, "两个默认模板的表达式必须逐字符相同")
	assert.NotEmpty(t, strings.TrimSpace(driftExpr))

	// 2) 三张倍率表：两个默认模板一致，且等于代码默认（内存与 DB 都要对）。
	defaultRatio := ratio_setting.GetDefaultModelRatioMap()
	defaultCompletion := ratio_setting.GetDefaultCompletionRatioMap()
	defaultCache := ratio_setting.GetDefaultCacheRatioMap()

	for _, testCase := range []struct {
		key        string
		liveValue  map[string]float64
		defaultRef float64
	}{
		{"ModelRatio", ratio_setting.GetModelRatioCopy(), defaultRatio[regressionModelRef]},
		{"CompletionRatio", ratio_setting.GetCompletionRatioCopy(), defaultCompletion[regressionModelRef]},
		{"CacheRatio", ratio_setting.GetCacheRatioCopy(), defaultCache[regressionModelRef]},
	} {
		stored := floatMapFromJSON(t, storedOption(t, testCase.key))
		assert.Equal(t, stored[regressionModelRef], stored[regressionModelDrift],
			"%s：两个默认模板必须一致（实例漂移应被重置抹平）", testCase.key)
		assert.Equal(t, testCase.defaultRef, stored[regressionModelDrift],
			"%s：DB 值必须回到代码默认", testCase.key)
		assert.Equal(t, stored[regressionModelDrift], testCase.liveValue[regressionModelDrift],
			"%s：内存值必须与重置后的 DB 值一致", testCase.key)
		assert.NotContains(t, stored, "stale-model", "%s：重置必须清掉陈旧条目", testCase.key)
	}

	// 3) 分组类键必须原样未动（DB 与内存都不得变）。
	groupRatio := storedOption(t, "GroupRatio")
	assert.JSONEq(t, sentinelGroupRatio, groupRatio, "重置不得改动 GroupRatio")
	assert.Contains(t, storedOption(t, "TopupGroupRatio"), "sentinel_topup", "重置不得改动 TopupGroupRatio")
	liveGroupRatio := ratio_setting.GetGroupRatioCopy()
	assert.Equal(t, 7.5, liveGroupRatio["vip"], "内存分组倍率同样不得被改")
	assert.NotContains(t, liveGroupRatio, "sentinel_group_never_written")
}

// TestResetModelRatioClearsEmptyDefaultTables 确认「默认是空表」的键被真正重置为空，
// 而不是把实例里残留的条目留在表内（空表默认也要如实落库）。
func TestResetModelRatioClearsEmptyDefaultTables(t *testing.T) {
	setupPricingResetTest(t)

	require.NoError(t, model.UpdateOption("ModelPrice", `{"legacy-model":1.5}`))
	require.NoError(t, model.UpdateOption("ImageRatio", `{"legacy-image":2.5}`))
	require.NoError(t, model.UpdateOption("AudioRatio", `{"legacy-audio":3.5}`))
	require.NoError(t, model.UpdateOption("AudioCompletionRatio", `{"legacy-audio-out":4.5}`))
	require.NoError(t, model.UpdateOption("CreateCacheRatio", `{"legacy-cache-create":5.5}`))

	body := resetModelRatioRequest(t)
	require.Contains(t, body, `"success":true`, body)

	for _, key := range []string{"ModelPrice", "ImageRatio", "AudioRatio", "AudioCompletionRatio", "CreateCacheRatio"} {
		assert.Equal(t, "{}", storedOption(t, key), "%s 的代码默认是空表，重置后必须为空", key)
	}
	assert.Empty(t, ratio_setting.GetModelPriceCopy())
	assert.Empty(t, ratio_setting.GetImageRatioCopy())
	assert.Empty(t, ratio_setting.GetAudioRatioCopy())
	assert.Empty(t, ratio_setting.GetAudioCompletionRatioCopy())
	assert.Empty(t, ratio_setting.GetCreateCacheRatioCopy())
}

// TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel 记录一个**已知的上游限制**（不在本任务改写范围）：
// `model.UpdateOption` 丢弃了 `DB.Save` 的错误，因此"写库失败"不会变成 error，重置接口会照旧返回成功。
// 本用例把这个行为钉住：一旦有人修好 model.UpdateOption 的静默失败，这里会失败并提醒同步复核控制器语义。
// 控制器侧的错误分支本身是就位的（任一步返回 error 即如实报失败），只是当前无法从 UpdateOption 触发。
func TestResetModelRatioWriteFailureIsSilentlySuccessfulInModel(t *testing.T) {
	setupPricingResetTest(t)

	require.NoError(t, model.UpdateOption("ModelRatio", `{"probe":1.5}`)) // 先建行，让后续走 UPDATE 分支
	require.NoError(t, model.DB.Exec(
		`CREATE TRIGGER reset_must_fail BEFORE UPDATE ON options
		 BEGIN SELECT RAISE(ABORT, 'forced option write failure'); END;`).Error)

	var option model.Option
	require.NoError(t, model.DB.Where("key = ?", "ModelRatio").First(&option).Error)
	option.Value = `{"probe":2.5}`
	require.Error(t, model.DB.Save(&option).Error, "DB 层确实写失败")

	// 上游限制：UpdateOption 把该失败吞掉了。
	require.NoError(t, model.UpdateOption("ModelRatio", `{"probe":3.5}`),
		"若此处开始报错，说明 model.UpdateOption 的静默失败已被修复，请同步复核 ResetModelRatio 的错误分支")

	body := resetModelRatioRequest(t)
	require.Contains(t, body, `"success":true`,
		"当前上游吞错，故接口仍报成功；这是记录在案的限制，不是本接口的期望语义")
}
