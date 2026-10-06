package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newScheduledCleanupPayload(t *testing.T) LogCleanupPayload {
	t.Helper()

	payload, ok := (logCleanupHandler{}).NewPayload().(LogCleanupPayload)
	require.True(t, ok, "scheduled cleanup must produce a LogCleanupPayload")
	return payload
}

// TestLogCleanupSchedulingStaysOptInOnDisk pins the default: with usage logs on
// disk nothing changes unless LOG_CLEANUP_INTERVAL is set.
func TestLogCleanupSchedulingStaysOptInOnDisk(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "")
	t.Setenv("LOG_CLEANUP_INTERVAL", "")
	t.Setenv("LOG_CLEANUP_RETENTION_DAYS", "")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "")
	t.Setenv("LOG_MEMORY_MAX_BYTES", "")

	handler := logCleanupHandler{}
	assert.False(t, handler.Enabled())
	assert.Zero(t, handler.Interval())
	assert.Zero(t, newScheduledCleanupPayload(t).MaxRows, "disk logs stay uncapped unless a cap is configured")
}

// TestLogCleanupSchedulingFollowsExplicitInterval pins the opt-in path: setting
// LOG_CLEANUP_INTERVAL enables the scheduled run on the configured cadence.
func TestLogCleanupSchedulingFollowsExplicitInterval(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "")
	t.Setenv("LOG_CLEANUP_INTERVAL", "10m")
	t.Setenv("LOG_CLEANUP_RETENTION_DAYS", "3")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "500")

	handler := logCleanupHandler{}
	assert.True(t, handler.Enabled())
	assert.Equal(t, 10*time.Minute, handler.Interval())

	payload := newScheduledCleanupPayload(t)
	assert.Equal(t, int64(500), payload.MaxRows)
	assert.InDelta(t, common.GetTimestamp()-3*secondsPerDay, payload.TargetTimestamp, 2)
}

// TestLogCleanupSchedulingDefaultsForInMemoryLogs pins the safety net: an
// in-memory log database is scheduled on its own with a bounded row cap, so RAM
// cannot grow without limit even if the operator configures nothing.
func TestLogCleanupSchedulingDefaultsForInMemoryLogs(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "memory")
	t.Setenv("LOG_CLEANUP_INTERVAL", "")
	t.Setenv("LOG_CLEANUP_RETENTION_DAYS", "")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "")
	t.Setenv("LOG_MEMORY_MAX_BYTES", "")

	handler := logCleanupHandler{}
	assert.True(t, handler.Enabled(), "in-memory logs must be cleaned automatically")
	assert.Equal(t, defaultLogCleanupMemoryInterval, handler.Interval())

	payload := newScheduledCleanupPayload(t)
	assert.Equal(t, int64(defaultLogCleanupMemoryMaxRows), payload.MaxRows)
	assert.InDelta(t, common.GetTimestamp()-defaultLogCleanupRetentionDays*secondsPerDay, payload.TargetTimestamp, 2)
}

// TestLogCleanupSchedulingWithoutRetentionKeepsOnlyTheRowCap pins that
// LOG_CLEANUP_RETENTION_DAYS=0 disables age-based deletion instead of deleting
// everything up to "now".
func TestLogCleanupSchedulingWithoutRetentionKeepsOnlyTheRowCap(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "memory")
	t.Setenv("LOG_CLEANUP_INTERVAL", "")
	t.Setenv("LOG_CLEANUP_RETENTION_DAYS", "0")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "")

	payload := newScheduledCleanupPayload(t)
	assert.Equal(t, int64(1), payload.TargetTimestamp, "cutoff must not delete any row by age")
	assert.Equal(t, int64(defaultLogCleanupMemoryMaxRows), payload.MaxRows)
}

// TestLogCleanupSchedulingDisabledByByteBudget pins the byte-budget default: a
// payload byte budget (LOG_MEMORY_MAX_BYTES) turns the scheduled cleanup off,
// because the write path trims as soon as the budget is exceeded. This holds for
// an in-memory log database too, where the timer used to be the safety net.
func TestLogCleanupSchedulingDisabledByByteBudget(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "memory")
	t.Setenv("LOG_CLEANUP_INTERVAL", "")
	t.Setenv("LOG_MEMORY_MAX_BYTES", "200MB")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "")

	handler := logCleanupHandler{}
	assert.False(t, handler.Enabled(), "配置字节预算后不应再跑定时清理")
	assert.Zero(t, handler.Interval())

	// 行数上限仍作为次级约束保留。
	assert.Equal(t, int64(defaultLogCleanupMemoryMaxRows), newScheduledCleanupPayload(t).MaxRows)
}

// TestLogCleanupSchedulingExplicitIntervalWinsOverByteBudget pins that an
// explicitly configured LOG_CLEANUP_INTERVAL still runs with a byte budget set.
func TestLogCleanupSchedulingExplicitIntervalWinsOverByteBudget(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "memory")
	t.Setenv("LOG_CLEANUP_INTERVAL", "10m")
	t.Setenv("LOG_MEMORY_MAX_BYTES", "200MB")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "")

	handler := logCleanupHandler{}
	assert.True(t, handler.Enabled())
	assert.Equal(t, 10*time.Minute, handler.Interval())
}

// TestLogCleanupSchedulingUnchangedWithoutByteBudget pins backward
// compatibility: without a byte budget nothing changes (in-memory logs keep the
// 5-minute cadence and the 200k row cap).
func TestLogCleanupSchedulingUnchangedWithoutByteBudget(t *testing.T) {
	t.Setenv("LOG_SQL_DSN", "memory")
	t.Setenv("LOG_CLEANUP_INTERVAL", "")
	t.Setenv("LOG_MEMORY_MAX_BYTES", "")
	t.Setenv("LOG_MEMORY_MAX_ROWS", "")

	handler := logCleanupHandler{}
	assert.True(t, handler.Enabled())
	assert.Equal(t, defaultLogCleanupMemoryInterval, handler.Interval())
	assert.Equal(t, int64(defaultLogCleanupMemoryMaxRows), newScheduledCleanupPayload(t).MaxRows)
}
