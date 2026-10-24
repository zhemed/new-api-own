package model

// twofa 查询的布尔字面量改用 commonTrueVal / commonFalseVal 之后的行为回归：
//   - 常量随主库方言取值（PostgreSQL: true/false；MySQL/SQLite: 1/0），拼接出的 SQL 必须语义等价；
//   - 三个曾被硬编码的查询（未使用备用码列表、未使用备用码计数、已启用 2FA 计数）行为不变。
//
// 背景：`is_used = false` / `is_enabled = true` 这类字面量在不同方言下都能跑，但不属项目约定的
// 跨库写法；改成常量后这两组变量"被读取"，从 lint 的 unused 名单里自然消失（而非靠白名单）。

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTwoFABooleanLiteralsPerDialect 校验常量取值与拼接出的 SQL 语义（SQLite 上实际执行）。
func TestTwoFABooleanLiteralsPerDialect(t *testing.T) {
	previousType := common.MainDatabaseType()
	t.Cleanup(func() {
		// 恢复主库类型后重新推导全部方言常量（含 logGroupCol/logKeyCol），
		// 避免本用例把 PH 分支的引号写法留给后续用例。
		common.SetMainDatabaseType(previousType)
		initCol()
	})

	// MySQL/SQLite 分支：布尔列按 1/0 存储。
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	require.Equal(t, "1", commonTrueVal, "SQLite/MySQL 分支的真值字面量应为 1")
	require.Equal(t, "0", commonFalseVal, "SQLite/MySQL 分支的假值字面量应为 0")

	var hit int
	require.NoError(t, DB.Raw("SELECT 1 WHERE 1 = "+commonTrueVal).Scan(&hit).Error)
	assert.Equal(t, 1, hit, "拼接出的真值条件必须在 SQLite 上成立")
	require.NoError(t, DB.Raw("SELECT 1 WHERE 0 = "+commonFalseVal).Scan(&hit).Error)
	assert.Equal(t, 1, hit, "拼接出的假值条件必须在 SQLite 上成立")
	hit = 0
	require.NoError(t, DB.Raw("SELECT 1 WHERE 1 = "+commonFalseVal).Scan(&hit).Error)
	assert.Zero(t, hit, "真值不得等于假值字面量（否则条件会被写成恒真）")

	// PostgreSQL 分支：布尔列只接受 true/false 字面量。
	common.SetMainDatabaseType(common.DatabaseTypePostgreSQL)
	initCol()
	require.Equal(t, "true", commonTrueVal, "PostgreSQL 分支的真值字面量应为 true")
	require.Equal(t, "false", commonFalseVal, "PostgreSQL 分支的假值字面量应为 false")
}

// TestTwoFABooleanQueriesBehavior 覆盖三处查询替换后的行为（SQLite 方言下实际跑）：
// 1) ValidateBackupCode 的「未使用备用码」查询；2) GetUnusedBackupCodeCount；
// 3) GetTwoFAStats 的「已启用 2FA」计数。
func TestTwoFABooleanQueriesBehavior(t *testing.T) {
	truncateTables(t)
	initCol() // 当前主库类型（TestMain 设为 SQLite）下常量就绪

	const userId = 7101
	codes, err := common.GenerateBackupCodes()
	require.NoError(t, err)
	require.NotEmpty(t, codes)
	rawCode := codes[0]
	codeHash, err := common.HashBackupCode(rawCode)
	require.NoError(t, err)

	require.NoError(t, DB.Create(&TwoFABackupCode{UserId: userId, CodeHash: codeHash}).Error) // IsUsed=false
	require.NoError(t, DB.Create(&TwoFABackupCode{UserId: userId, CodeHash: "already-used", IsUsed: true}).Error)

	unused, err := GetUnusedBackupCodeCount(userId)
	require.NoError(t, err)
	assert.Equal(t, 1, unused, "is_used 条件必须只命中未使用的备用码（区别于恒真）")

	ok, err := ValidateBackupCode(userId, rawCode)
	require.NoError(t, err)
	assert.True(t, ok, "未使用的备用码应验证通过")

	unused, err = GetUnusedBackupCodeCount(userId)
	require.NoError(t, err)
	assert.Zero(t, unused, "验证通过后该备用码应被标记为已使用")

	ok, err = ValidateBackupCode(userId, rawCode)
	require.NoError(t, err)
	assert.False(t, ok, "已使用的备用码不得再次通过（is_used 条件真的生效）")

	require.NoError(t, DB.Create(&TwoFA{UserId: 1, Secret: "secret-enabled", IsEnabled: true}).Error)
	require.NoError(t, DB.Create(&TwoFA{UserId: 2, Secret: "secret-disabled", IsEnabled: false}).Error)

	stats, err := GetTwoFAStats()
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats["enabled_users"], "is_enabled 条件必须只统计已启用的 2FA")
}
