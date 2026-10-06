package model

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// insertLogRow 通过唯一的写入入口 createLog 插入一行（与生产路径一致）。
func insertLogRow(t *testing.T, userId int, content string) {
	t.Helper()
	require.NoError(t, createLog(&Log{
		UserId:    userId,
		Type:      LogTypeConsume,
		CreatedAt: common.GetTimestamp(),
		Content:   content,
	}))
}

func TestLogPayloadBudgetTrimsOldestRowsToBudget(t *testing.T) {
	truncateTables(t)
	logPayloadBytes.Store(0)
	t.Setenv(LogMemoryMaxBytesEnv, "8192")
	t.Setenv(LogRetentionDaysEnv, "0") // 只考察字节预算这条路径

	content := strings.Repeat("x", 400)
	for i := 0; i < 60; i++ {
		insertLogRow(t, i, content)
	}

	require.Eventually(t, func() bool {
		total, err := sumLogPayloadBytes(context.Background())
		return err == nil && total <= 8192
	}, 5*time.Second, 20*time.Millisecond, "写入触发的裁剪应把载荷带回预算内")

	var newest Log
	require.NoError(t, LOG_DB.Order("id desc").First(&newest).Error)
	assert.Equal(t, 59, newest.UserId, "最新的日志必须保留")

	var remaining int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&remaining).Error)
	assert.Greater(t, remaining, int64(0))
	assert.Less(t, remaining, int64(60), "超出预算的老记录应被删除")
}

func TestLogPayloadTrimKeepsNewestRowWhenSingleRowExceedsBudget(t *testing.T) {
	truncateTables(t)
	logPayloadBytes.Store(0)
	t.Setenv(LogMemoryMaxBytesEnv, "256")
	t.Setenv(LogRetentionDaysEnv, "0")

	for i := 0; i < 5; i++ {
		insertLogRow(t, i, strings.Repeat("y", 4096))
	}

	require.Eventually(t, func() bool {
		var rows int64
		if err := LOG_DB.Model(&Log{}).Count(&rows).Error; err != nil {
			return false
		}
		return rows == 1
	}, 5*time.Second, 20*time.Millisecond, "单行超预算时只保留最新一行，绝不删空")

	var survivor Log
	require.NoError(t, LOG_DB.Order("id desc").First(&survivor).Error)
	assert.Equal(t, 4, survivor.UserId)
}

func TestLogPayloadTrimIsSingleFlight(t *testing.T) {
	require.True(t, beginLogPayloadTrim())
	require.False(t, beginLogPayloadTrim(), "同一时刻只能有一路裁剪")
	endLogPayloadTrim()
	require.True(t, beginLogPayloadTrim())
	endLogPayloadTrim()

	var winners int32
	var waitGroup sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			if beginLogPayloadTrim() {
				atomic.AddInt32(&winners, 1)
			}
		}()
	}
	close(start)
	waitGroup.Wait()
	endLogPayloadTrim()

	assert.Equal(t, int32(1), winners, "并发触发只允许一路赢得单飞")
}

func TestLogPayloadTrimRunsRetentionInSamePass(t *testing.T) {
	truncateTables(t)
	logPayloadBytes.Store(0)
	t.Setenv(LogMemoryMaxBytesEnv, "1048576") // 预算足够大：只考察保留天数是否同遍生效
	t.Setenv(LogRetentionDaysEnv, "1")

	require.NoError(t, createLog(&Log{
		UserId:    1,
		Type:      LogTypeConsume,
		CreatedAt: common.GetTimestamp() - 3*logSecondsPerDay,
		Content:   "expired row",
	}))
	insertLogRow(t, 2, "fresh row")

	require.NoError(t, trimLogsToPayloadBudget(context.Background(), 1048576, logPayloadTrimBatchSize))

	var rows []Log
	require.NoError(t, LOG_DB.Order("id asc").Find(&rows).Error)
	require.Len(t, rows, 1, "保留天数删除必须与字节裁剪在同一遍执行")
	assert.Equal(t, 2, rows[0].UserId)
}

func TestLogPayloadBudgetDisabledByDefault(t *testing.T) {
	truncateTables(t)
	logPayloadBytes.Store(0)
	t.Setenv(LogMemoryMaxBytesEnv, "")
	t.Setenv(LogRetentionDaysEnv, "0")

	require.Zero(t, LogPayloadBudgetBytes(), "未配置预算时应为关闭状态")
	for i := 0; i < 10; i++ {
		insertLogRow(t, i, strings.Repeat("z", 4096))
	}

	assert.Zero(t, logPayloadBytes.Load(), "未配置预算时不记账、不触发裁剪")
	assert.False(t, logByteTrimRunning.Load())

	var rows int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&rows).Error)
	assert.Equal(t, int64(10), rows, "未配置预算时行为与改造前一致：不做任何裁剪")
}

func TestLogPayloadBudgetWorksOnDiskBackedLogs(t *testing.T) {
	previousLogDB := LOG_DB
	diskDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "logs.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, diskDB.AutoMigrate(&Log{}))
	LOG_DB = diskDB
	t.Cleanup(func() {
		LOG_DB = previousLogDB
		sqlDB, dbErr := diskDB.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	logPayloadBytes.Store(0)
	t.Setenv(LogMemoryMaxBytesEnv, "4096")
	t.Setenv(LogRetentionDaysEnv, "0")

	for i := 0; i < 30; i++ {
		insertLogRow(t, i, strings.Repeat("d", 400))
	}

	require.Eventually(t, func() bool {
		total, sumErr := sumLogPayloadBytes(context.Background())
		return sumErr == nil && total <= 4096
	}, 5*time.Second, 20*time.Millisecond, "机制不依赖内存库：磁盘模式同样被裁到预算内")
}

func TestLogPayloadTrimEnforcesRowCapInSamePass(t *testing.T) {
	truncateTables(t)
	logPayloadBytes.Store(0)
	t.Setenv(LogMemoryMaxBytesEnv, "0") // 只看行数上限这一次级约束
	t.Setenv(LogMemoryMaxRowsEnv, "3")
	t.Setenv(LogRetentionDaysEnv, "0")

	for i := 0; i < 10; i++ {
		insertLogRow(t, i, "row")
	}

	require.NoError(t, trimLogsToPayloadBudget(context.Background(), 0, logPayloadTrimBatchSize))

	var rows int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&rows).Error)
	assert.Equal(t, int64(3), rows, "行数上限与字节预算在同一遍里同时生效")

	var newest Log
	require.NoError(t, LOG_DB.Order("id desc").First(&newest).Error)
	assert.Equal(t, 9, newest.UserId)
}

func TestLogPayloadUsageReportsBudgetAndGrowth(t *testing.T) {
	truncateTables(t)
	logPayloadBytes.Store(0)
	t.Setenv(LogRetentionDaysEnv, "0")

	// 未配置预算：面板看到 max=0（此时也不做记账，与改造前一致）。
	t.Setenv(LogMemoryMaxBytesEnv, "")
	assert.Zero(t, LogPayloadUsage(context.Background()).Budget)

	// 配置预算后：max 按配置回传（200MB -> 209715200），bytes 随插入增长，rows 反映行数。
	t.Setenv(LogMemoryMaxBytesEnv, "200MB")
	require.NoError(t, calibrateLogPayloadCounter(context.Background()))
	before := LogPayloadUsage(context.Background())
	require.Equal(t, int64(200*1024*1024), before.Budget)

	insertLogRow(t, 1, strings.Repeat("g", 500))

	after := LogPayloadUsage(context.Background())
	assert.Greater(t, after.Bytes, before.Bytes, "bytes 随插入增长")
	assert.Equal(t, int64(1), after.Rows)
}
