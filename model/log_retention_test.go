package model

import (
	"context"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveLogSQLiteTarget(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantDSN string
		wantLog bool
		wantMem bool
	}{
		{name: "empty dsn is not a sqlite target", dsn: ""},
		{name: "local token stays with the existing handling", dsn: "local"},
		{name: "mysql dsn is untouched", dsn: "user:pass@tcp(127.0.0.1:3306)/logs"},
		{name: "postgres dsn is untouched", dsn: "postgres://user:pass@127.0.0.1:5432/logs"},
		{name: "clickhouse dsn is untouched", dsn: "https://ch.example.com:8443"},
		{
			name:    "memory token selects an in-memory shared-cache database",
			dsn:     "memory",
			wantDSN: "file::memory:?cache=shared",
			wantLog: true,
			wantMem: true,
		},
		{
			name:    "sqlite-style memory dsn also selects memory",
			dsn:     ":memory:",
			wantDSN: "file::memory:?cache=shared",
			wantLog: true,
			wantMem: true,
		},
		{
			name:    "sqlite prefix selects a dedicated file",
			dsn:     "sqlite:/dev/shm/newapi-logs.db?_pragma=journal_mode(WAL)",
			wantDSN: "/dev/shm/newapi-logs.db?_pragma=journal_mode(WAL)",
			wantLog: true,
		},
		{
			name: "sqlite prefix without a path is ignored",
			dsn:  "sqlite:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := resolveLogSQLiteTarget(tt.dsn)
			assert.Equal(t, tt.wantLog, target.IsSQLite)
			assert.Equal(t, tt.wantMem, target.InMemory)
			assert.Equal(t, tt.wantDSN, target.DSN)
		})
	}
}

// TestInitLogDBWithInMemoryDSN pins the two properties an in-memory log
// database depends on: the pool keeps exactly one connection (so the shared
// cache survives) and rows written through it stay readable.
func TestInitLogDBWithInMemoryDSN(t *testing.T) {
	previousLogDB := LOG_DB
	t.Cleanup(func() {
		LOG_DB = previousLogDB
		common.SetLogDatabaseType(common.DatabaseTypeSQLite)
	})

	t.Setenv("LOG_SQL_DSN", "memory")
	require.True(t, UsingInMemoryLogDatabase())

	require.NoError(t, InitLogDB())
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))

	entry := Log{UserId: 7, Type: LogTypeConsume, CreatedAt: common.GetTimestamp(), Content: "in-memory retention fixture"}
	require.NoError(t, LOG_DB.Create(&entry).Error)

	sqlDB, err := LOG_DB.DB()
	require.NoError(t, err)
	assert.Equal(t, 1, sqlDB.Stats().MaxOpenConnections, "memory mode must pin a single connection")

	var count int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "row must stay readable after the pool reuses its connection")
}

// TestSharedCacheMemoryDatabaseDisappearsWithoutPinnedConnection is the negative
// control for the test above: with no idle connection kept, the in-memory
// database is dropped and the schema vanishes, which is exactly what pinning a
// connection in InitLogDB prevents.
func TestSharedCacheMemoryDatabaseDisappearsWithoutPinnedConnection(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxIdleConns(0)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(&Log{}))
	require.NoError(t, db.Create(&Log{UserId: 1, CreatedAt: common.GetTimestamp()}).Error)

	// Drop the only connection: with MaxIdleConns(0) the pool closes it as soon
	// as the query finishes, taking the in-memory schema with it.
	require.NoError(t, sqlDB.Close())

	err = db.Model(&Log{}).Count(new(int64)).Error
	assert.Error(t, err, "shared-cache memory database must not survive with no open connection")
}

func TestTrimLogToMaxRowsKeepsNewestRows(t *testing.T) {
	truncateTables(t)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, LOG_DB.Create(&Log{
			UserId:    i,
			Type:      LogTypeConsume,
			CreatedAt: common.GetTimestamp() + int64(i),
		}).Error)
	}

	for {
		affected, err := TrimLogToMaxRows(ctx, 2, 100)
		require.NoError(t, err)
		if affected == 0 {
			break
		}
	}

	var remaining []Log
	require.NoError(t, LOG_DB.Order("id asc").Find(&remaining).Error)
	require.Len(t, remaining, 2)
	assert.Equal(t, 3, remaining[0].UserId, "oldest rows are trimmed first")
	assert.Equal(t, 4, remaining[1].UserId)
}

func TestTrimLogToMaxRowsIsNoOpWithoutCap(t *testing.T) {
	truncateTables(t)

	require.NoError(t, LOG_DB.Create(&Log{UserId: 1, CreatedAt: common.GetTimestamp()}).Error)

	affected, err := TrimLogToMaxRows(context.Background(), 0, 100)
	require.NoError(t, err)
	assert.Zero(t, affected)

	var count int64
	require.NoError(t, LOG_DB.Model(&Log{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
