package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	// LogMemoryMaxBytesEnv is the payload byte budget for the log table. It
	// accepts "200MB" or a plain byte count like "209715200"; 0/unset disables
	// the budget (and keeps the previous row-cap-only behaviour).
	LogMemoryMaxBytesEnv = "LOG_MEMORY_MAX_BYTES"
	// LogRetentionDaysEnv is the age-based retention window in days; 0 disables
	// age-based deletion.
	LogRetentionDaysEnv = "LOG_CLEANUP_RETENTION_DAYS"
	// LogMemoryMaxRowsEnv is the row cap for the log table. It stays a secondary
	// constraint next to the payload budget and is enforced in the same pass.
	LogMemoryMaxRowsEnv = "LOG_MEMORY_MAX_ROWS"
	// DefaultLogRetentionDays applies when neither an explicit value nor 0 is set.
	DefaultLogRetentionDays = 7
	// DefaultLogMemoryMaxRows bounds RAM when logs live in memory and no row cap
	// is configured.
	DefaultLogMemoryMaxRows = 200000

	// logPayloadTrimBatchSize is how many rows one payload trim batch removes.
	logPayloadTrimBatchSize = 500
	// logRowOverheadBytes is a fixed per-row overhead estimate. It is not the
	// physical row size; it only makes the payload estimate grow with row count.
	logRowOverheadBytes = 128
	// logPayloadTrimMaxBatches bounds one trim pass so it cannot hold a
	// connection forever when the table keeps being written to.
	logPayloadTrimMaxBatches = 200
	logSecondsPerDay         = 86400
)

// logPayloadBytes is the in-process payload estimate of the log table. It is
// maintained on the write path (O(1)) and re-calibrated with a single
// SUM(LENGTH(...)) on first use and at the end of every trim pass, so drift is
// self-correcting.
var (
	logPayloadBytes              atomic.Int64
	logByteTrimRunning           atomic.Bool
	logPayloadCalibrateOnce      sync.Once
	warnClickHouseByteBudgetOnce sync.Once
	warnClickHouseRowCapOnce     sync.Once
)

// LogRowCap returns the row cap enforced by every cleanup run (scheduled and
// write-triggered): the explicit LOG_MEMORY_MAX_ROWS value wins, an in-memory log
// database falls back to a bounded default, and on-disk logs stay uncapped.
func LogRowCap() int64 {
	if configured := common.GetEnvOrDefault(LogMemoryMaxRowsEnv, 0); configured > 0 {
		// ClickHouse 的 DELETE 是重写 data part 的 mutation，且 TrimLogToMaxRows 没有 CH
		// 方言分支（见 model/log.go 的 TrimLogToMaxRows 注释）——显式忽略而不是让它失败。
		if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
			warnClickHouseRowCapOnce.Do(func() {
				logger.LogWarn(context.Background(), fmt.Sprintf(
					"%s is ignored for the ClickHouse log database: row-cap trimming has no ClickHouse branch; use retention (%s) instead",
					LogMemoryMaxRowsEnv, LogRetentionDaysEnv))
			})
			return 0
		}
		return int64(configured)
	}
	if UsingInMemoryLogDatabase() {
		return DefaultLogMemoryMaxRows
	}
	return 0
}

// LogPayloadBudgetBytes returns the effective payload byte budget. 0 means "not
// configured" (the feature is off and the write path keeps its previous
// behaviour). ClickHouse log tables are excluded for the same reason
// LOG_MEMORY_MAX_ROWS is: trimming there is a data-part mutation with no
// dialect branch in this package.
func LogPayloadBudgetBytes() int64 {
	configured := common.GetEnvOrDefaultSize(LogMemoryMaxBytesEnv, 0)
	if configured <= 0 {
		return 0
	}
	if common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		warnClickHouseByteBudgetOnce.Do(func() {
			logger.LogWarn(context.Background(), fmt.Sprintf(
				"%s is ignored for the ClickHouse log database: payload trimming has no ClickHouse branch; use retention (%s) instead",
				LogMemoryMaxBytesEnv, LogRetentionDaysEnv))
		})
		return 0
	}
	return configured
}

// LogPayloadUsageStats describes the current log payload usage for the admin
// panel. Bytes is the same estimate the trim path uses (text columns plus a
// fixed per-row overhead); it does NOT include SQLite pages, indexes, or WAL
// storage.
type LogPayloadUsageStats struct {
	Bytes  int64 `json:"bytes"`
	Budget int64 `json:"budget"`
	Rows   int64 `json:"rows"`
}

// LogPayloadUsage reports the current payload estimate, the effective budget and
// the current row count. It shares the estimator with the trim path on purpose:
// the panel must show exactly the number the trim decision is based on. When a
// budget is configured, reading the stats also triggers the one-time
// calibration so a cold process reports a real number.
func LogPayloadUsage(ctx context.Context) LogPayloadUsageStats {
	stats := LogPayloadUsageStats{Budget: LogPayloadBudgetBytes()}
	if stats.Budget > 0 {
		ensureLogPayloadCalibration()
	}
	stats.Bytes = logPayloadBytes.Load()
	if LOG_DB == nil {
		return stats
	}
	rows, err := countLogRows(ctx)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("log payload row count failed: %v", err))
		return stats
	}
	stats.Rows = rows
	return stats
}

// logPayloadBytesOf estimates one row's payload: the text columns' lengths plus
// the fixed per-row overhead. The SQL side of the same estimate uses LENGTH(),
// so on SQLite/PostgreSQL multi-byte text counts characters there and bytes in
// Go; the value is a relative budget, not an exact byte count.
func logPayloadBytesOf(log *Log) int64 {
	if log == nil {
		return 0
	}
	textBytes := len(log.Content) + len(log.Username) + len(log.TokenName) + len(log.ModelName) +
		len(log.Group) + len(log.Ip) + len(log.RequestId) + len(log.UpstreamRequestId) + len(log.Other)
	return int64(textBytes) + logRowOverheadBytes
}

// logPayloadSQLColumns lists the text columns measured by the SQL side of the
// payload estimate. `group` is a reserved word on MySQL/PostgreSQL and is quoted
// through commonGroupCol.
func logPayloadSQLColumns() []string {
	columns := []string{"content", "username", "token_name", "model_name", "ip", "request_id", "upstream_request_id", "other"}
	if commonGroupCol != "" {
		columns = append(columns, commonGroupCol)
	}
	return columns
}

func logPayloadSQLExpression() string {
	columns := logPayloadSQLColumns()
	parts := make([]string, 0, len(columns)+1)
	for _, column := range columns {
		parts = append(parts, "COALESCE(LENGTH("+column+"), 0)")
	}
	parts = append(parts, fmt.Sprintf("%d", logRowOverheadBytes))
	return "(" + strings.Join(parts, " + ") + ")"
}

// noteLogPayloadBytes accumulates the payload estimate of one written row and
// triggers an asynchronous trim when the budget is exceeded. It performs no IO
// and never blocks the caller; with no budget configured it returns immediately,
// which keeps the pre-existing behaviour byte for byte.
func noteLogPayloadBytes(log *Log) {
	budget := LogPayloadBudgetBytes()
	if budget <= 0 {
		return
	}
	total := logPayloadBytes.Add(logPayloadBytesOf(log))
	ensureLogPayloadCalibration()
	if total > budget {
		triggerLogPayloadTrim()
	}
}

// ensureLogPayloadCalibration runs the one-time SUM calibration in the
// background so the first log write never waits for a table scan.
func ensureLogPayloadCalibration() {
	logPayloadCalibrateOnce.Do(func() {
		gopool.Go(func() {
			if err := calibrateLogPayloadCounter(context.Background()); err != nil {
				logger.LogWarn(context.Background(), fmt.Sprintf("log payload calibration failed: %v", err))
			}
		})
	})
}

// triggerLogPayloadTrim starts one trim pass unless one is already running
// (single-flight).
func triggerLogPayloadTrim() {
	if !beginLogPayloadTrim() {
		return
	}
	gopool.Go(func() {
		budget := LogPayloadBudgetBytes()
		if budget > 0 {
			ctx := context.Background()
			if err := trimLogsToPayloadBudget(ctx, budget, logPayloadTrimBatchSize); err != nil {
				// A failed trim is only logged: the write path must never fail
				// because housekeeping could not keep up.
				logger.LogWarn(ctx, fmt.Sprintf("log payload trim failed: %v", err))
			}
		}
		endLogPayloadTrim()
		// 裁剪期间可能又有写入把用量顶回预算之上，而那次写入的触发会被单飞挡掉；
		// 所以释放单飞后再自检一次，避免"最后一次触发被丢弃"导致长期停在预算之上。
		if budget > 0 && logPayloadBytes.Load() > budget {
			triggerLogPayloadTrim()
		}
	})
}

// beginLogPayloadTrim reports whether this caller owns the trim pass. Exposed
// for the single-flight invariant test.
func beginLogPayloadTrim() bool {
	return logByteTrimRunning.CompareAndSwap(false, true)
}

func endLogPayloadTrim() {
	logByteTrimRunning.Store(false)
}

// trimLogsToPayloadBudget runs one write-triggered cleanup pass: age-based
// retention first, then oldest-first deletion until the payload fits the budget,
// and finally a calibration so the in-process counter matches the table.
func trimLogsToPayloadBudget(ctx context.Context, budgetBytes int64, batchSize int) error {
	if batchSize <= 0 {
		batchSize = logPayloadTrimBatchSize
	}
	if err := trimLogsByRetention(ctx, batchSize); err != nil {
		return err
	}
	if rowCap := LogRowCap(); rowCap > 0 {
		if err := trimOldestLogsToRowCap(ctx, rowCap, batchSize); err != nil {
			return err
		}
	}
	if budgetBytes > 0 {
		if err := trimOldestLogsToPayloadBudget(ctx, budgetBytes, batchSize); err != nil {
			return err
		}
	}
	return calibrateLogPayloadCounter(ctx)
}

// trimLogsByRetention deletes rows older than LOG_CLEANUP_RETENTION_DAYS in the
// same pass as the payload trim. Retention 0 keeps the previous semantics: no
// age-based deletion at all (the cutoff must not become "delete everything").
func trimLogsByRetention(ctx context.Context, batchSize int) error {
	retentionDays := common.GetEnvOrDefault(LogRetentionDaysEnv, DefaultLogRetentionDays)
	if retentionDays <= 0 {
		return nil
	}
	cutoff := common.GetTimestamp() - int64(retentionDays)*logSecondsPerDay
	for i := 0; i < logPayloadTrimMaxBatches; i++ {
		affected, err := DeleteOldLogBatch(ctx, cutoff, batchSize)
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}
	}
	return nil
}

// trimOldestLogsToRowCap enforces the row cap as the secondary constraint next
// to the payload budget, reusing the shared TrimLogToMaxRows helper.
func trimOldestLogsToRowCap(ctx context.Context, rowCap int64, batchSize int) error {
	for i := 0; i < logPayloadTrimMaxBatches; i++ {
		affected, err := TrimLogToMaxRows(ctx, rowCap, batchSize)
		if err != nil {
			return err
		}
		if affected == 0 {
			return nil
		}
	}
	return nil
}

// trimOldestLogsToPayloadBudget deletes the oldest rows until the estimated
// payload fits the budget. A single row larger than the budget is kept as the
// newest row: the table is never emptied.
func trimOldestLogsToPayloadBudget(ctx context.Context, budgetBytes int64, batchSize int) error {
	total, err := sumLogPayloadBytes(ctx)
	if err != nil {
		return err
	}
	for i := 0; i < logPayloadTrimMaxBatches && total > budgetBytes; i++ {
		rows, err := countLogRows(ctx)
		if err != nil {
			return err
		}
		if rows <= 1 {
			return nil
		}
		limit := int64(batchSize)
		if limit > rows-1 {
			limit = rows - 1
		}
		deletedBytes, err := deleteOldestLogPayloadBatch(ctx, int(limit))
		if err != nil {
			return err
		}
		if deletedBytes <= 0 {
			return nil
		}
		total -= deletedBytes
		if total < 0 {
			total = 0
		}
	}
	return nil
}

// calibrateLogPayloadCounter replaces the counter with the real SUM of the
// table's payload estimate. Concurrent writes landing during the query are
// absorbed by this overwrite; the next calibration (end of the next trim pass or
// a panel read) corrects any drift.
func calibrateLogPayloadCounter(ctx context.Context) error {
	total, err := sumLogPayloadBytes(ctx)
	if err != nil {
		return err
	}
	logPayloadBytes.Store(total)
	return nil
}

func countLogRows(ctx context.Context) (int64, error) {
	if LOG_DB == nil {
		return 0, errors.New("log database is not initialized")
	}
	var rows int64
	if err := LOG_DB.WithContext(ctx).Model(&Log{}).Count(&rows).Error; err != nil {
		return 0, err
	}
	return rows, nil
}

func sumLogPayloadBytes(ctx context.Context) (int64, error) {
	if LOG_DB == nil {
		return 0, errors.New("log database is not initialized")
	}
	var total int64
	expression := "COALESCE(SUM(" + logPayloadSQLExpression() + "), 0)"
	if err := LOG_DB.WithContext(ctx).Model(&Log{}).Select(expression).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// deleteOldestLogPayloadBatch removes the oldest rows and returns the payload
// estimate they held. It selects ids plus per-row payload first and then deletes
// by id, so the statement stays portable: neither DELETE ... LIMIT (MySQL only)
// nor RETURNING (PostgreSQL/SQLite only) is required.
func deleteOldestLogPayloadBatch(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}
	if LOG_DB == nil {
		return 0, errors.New("log database is not initialized")
	}
	type payloadRow struct {
		Id           int
		PayloadBytes int64
	}
	var batch []payloadRow
	expression := logPayloadSQLExpression() + " AS payload_bytes"
	if err := LOG_DB.WithContext(ctx).Model(&Log{}).
		Select("id, " + expression).
		Order("id asc").
		Limit(limit).
		Scan(&batch).Error; err != nil {
		return 0, err
	}
	if len(batch) == 0 {
		return 0, nil
	}

	ids := make([]int, 0, len(batch))
	var deletedBytes int64
	for _, row := range batch {
		ids = append(ids, row.Id)
		deletedBytes += row.PayloadBytes
	}
	result := LOG_DB.WithContext(ctx).Where("id IN ?", ids).Delete(&Log{})
	if result.Error != nil {
		return 0, result.Error
	}
	return deletedBytes, nil
}
