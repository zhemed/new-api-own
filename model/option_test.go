package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// optionProbeKey 是测试专用键：updateOptionMap 对它只做"写进 OptionMap"这一步，
// 不触发任何业务分支，便于断言内存与 DB 是否一致。
const optionProbeKey = "UpdateOptionErrorPropagationProbe"

// useOptionTestDB 建一个隔离的内存 SQLite（只迁移 options 表），
// 并保存/恢复 OptionMap，避免污染同包其它用例。
func useOptionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	previousDB := DB
	previousType := common.MainDatabaseType()
	previousOptionMap := common.OptionMap

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Option{}))

	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func optionMemoryValue(key string) (string, bool) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	value, ok := common.OptionMap[key]
	return value, ok
}

func optionDBValue(t *testing.T, db *gorm.DB, key string) string {
	t.Helper()
	var option Option
	require.NoError(t, db.Where("key = ?", key).First(&option).Error)
	return option.Value
}

// 写库失败（UPDATE 被触发器打断）时：UpdateOption 必须返回 error，
// 且内存不得被更新（否则 DB 与内存不一致）。
func TestUpdateOptionPropagatesSaveErrorAndKeepsMemory(t *testing.T) {
	db := useOptionTestDB(t)

	require.NoError(t, UpdateOption(optionProbeKey, "old-value"))
	require.Equal(t, "old-value", optionDBValue(t, db, optionProbeKey))
	require.Equal(t, "old-value", mustMemoryValue(t, optionProbeKey))

	// 让后续 UPDATE 必然失败；行已存在，FirstOrCreate 只做 SELECT，命中 Save。
	require.NoError(t, db.Exec(
		`CREATE TRIGGER update_option_must_fail BEFORE UPDATE ON options
		 BEGIN SELECT RAISE(ABORT, 'forced option write failure'); END;`).Error)

	err := UpdateOption(optionProbeKey, "new-value")

	require.Error(t, err, "Save 失败必须传播为 error（修复前这里返回 nil）")
	assert.Contains(t, err.Error(), optionProbeKey, "错误信息必须能定位到 key")
	assert.NotContains(t, err.Error(), "new-value", "错误信息不得泄露值内容")
	// 内存必须保持旧值：写库失败时不得调用 updateOptionMap。
	assert.Equal(t, "old-value", mustMemoryValue(t, optionProbeKey),
		"写库失败后内存必须保持旧值，不得与 DB 分叉")
	// DB 也仍是旧值。
	assert.Equal(t, "old-value", optionDBValue(t, db, optionProbeKey))
}

// 写库失败（INSERT 被触发器打断）时：FirstOrCreate 的错误也必须传播。
func TestUpdateOptionPropagatesCreateError(t *testing.T) {
	db := useOptionTestDB(t)

	require.NoError(t, db.Exec(
		`CREATE TRIGGER insert_option_must_fail BEFORE INSERT ON options
		 BEGIN SELECT RAISE(ABORT, 'forced option insert failure'); END;`).Error)

	err := UpdateOption(optionProbeKey, "some-value")

	require.Error(t, err, "FirstOrCreate 失败必须传播为 error（修复前这里返回 nil）")
	assert.Contains(t, err.Error(), optionProbeKey, "错误信息必须能定位到 key")
	assert.NotContains(t, err.Error(), "some-value", "错误信息不得泄露值内容")
	// 内存不得被写入：键本应不存在。
	_, exists := optionMemoryValue(optionProbeKey)
	assert.False(t, exists, "写库失败时不得更新内存")
}

// 正常路径：DB 与内存一致，返回值 nil。
func TestUpdateOptionSuccessKeepsDBAndMemoryInSync(t *testing.T) {
	db := useOptionTestDB(t)

	require.NoError(t, UpdateOption(optionProbeKey, "first-value"))
	assert.Equal(t, "first-value", optionDBValue(t, db, optionProbeKey))
	assert.Equal(t, "first-value", mustMemoryValue(t, optionProbeKey))

	// 第二次是 UPDATE 路径（行已存在），同样要一致。
	require.NoError(t, UpdateOption(optionProbeKey, "second-value"))
	assert.Equal(t, "second-value", optionDBValue(t, db, optionProbeKey))
	assert.Equal(t, "second-value", mustMemoryValue(t, optionProbeKey))
}

func mustMemoryValue(t *testing.T, key string) string {
	t.Helper()
	value, ok := optionMemoryValue(key)
	require.True(t, ok, "内存 OptionMap 应包含 %s", key)
	return value
}
