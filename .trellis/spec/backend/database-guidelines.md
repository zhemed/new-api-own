# Database Guidelines

> GORM v2 patterns, cross-dialect compatibility, and migrations.

---

## Overview

- ORM: **GORM v2** (`gorm.io/gorm`). Drivers: `github.com/glebarez/sqlite`, `gorm.io/driver/mysql`,
  `gorm.io/driver/postgres`; the log database may additionally be ClickHouse (`gorm.io/driver/clickhouse`).
- Every statement MUST work on **SQLite, MySQL >= 5.7.8, and PostgreSQL >= 9.6** simultaneously.
  A change that only works on one dialect is a bug, not a tradeoff.
- Two connections: `model.DB` (main data) and `model.LOG_DB` (logs). They can point at different
  servers; `LOG_SQL_DSN` selects the log DB, otherwise `LOG_DB = DB` (`model/main.go:212-218`).
- Setup and migrations live in `model/main.go` (`InitDB`, `InitLogDB`, `migrateDB`) and run only on
  the master node (`common.IsMasterNode`, `model/main.go:197-205`).
- Pool settings come from env: `SQL_MAX_IDLE_CONNS`, `SQL_MAX_OPEN_CONNS`, `SQL_MAX_LIFETIME`
  (`model/main.go:193-195`).

---

## Query Patterns

- Prefer GORM methods (`Create`, `Find`, `Where`, `Updates`, ...) over raw SQL.
- Branch on the active database with `common.UsingMainDatabase(...)` / `common.UsingLogDatabase(...)`,
  never with an ad-hoc type comparison.
- Reserved-word columns `group` and `key` must be quoted per dialect. Use the helpers initialized in
  `model/main.go:30-51`: `commonGroupCol`, `commonKeyCol`, `commonTrueVal`, `commonFalseVal` for the
  main DB, `logGroupCol` / `logKeyCol` for the log DB.
- Booleans in raw SQL go through `commonTrueVal` / `commonFalseVal` (PostgreSQL `true`/`false`,
  MySQL/SQLite `1`/`0`).
- Row locks: use `lockForUpdate(tx)` from `model/locking.go:20`. It emits `FOR UPDATE` on
  MySQL/PostgreSQL and is a no-op on SQLite, where the syntax is unsupported.

```go
// model/locking.go:20
func lockForUpdate(tx *gorm.DB) *gorm.DB {
	if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}
```

```go
// model/user_session.go:573 — typical use inside a transaction
if err := lockForUpdate(tx).Where("sid = ? AND user_id = ?", sid, userID).First(&current).Error; err != nil {
```

- Transactions use `DB.Transaction(func(tx *gorm.DB) error { ... })` — examples:
  `model/checkin.go:96`, `model/auth_flow.go:125`, `model/external_identity_claim.go:95`.
- Log records are written through `LOG_DB` (its schema is migrated by `LOG_DB.AutoMigrate(&Log{})`,
  `model/main.go:403`).

---

## Migrations

- Schema owner: `migrateDB()` in `model/main.go`. The main model list is passed to `DB.AutoMigrate(...)`
  at `model/main.go:261`.
- **Adding a column to an existing table**: append an entry to the `required` slice
  (`Name` + dialect-neutral `DDL`) and let the loop issue a guarded `ALTER TABLE ... ADD COLUMN`
  (`model/main.go:544-576`). This helper is idempotent and safe to re-run on every boot:

```go
// model/main.go:568-575
for _, col := range required {
	if _, ok := existing[col.Name]; ok {
		continue
	}
	if err := DB.Exec("ALTER TABLE `" + tableName + "` ADD COLUMN " + col.DDL).Error; err != nil {
		return err
	}
}
```

- **Changing a column type**: write a dedicated migration function that is guarded and dialect-aware.
  Check `DB.Migrator().HasTable(...)` / `HasColumn(...)` first, skip SQLite when type affinity makes the
  change unnecessary, and query `information_schema` for PostgreSQL. Model: `migrateTokenModelLimitsToText`
  (`model/main.go:581-604`).
- SQLite has no `ALTER COLUMN` — use `ADD COLUMN` only. MySQL uses `MODIFY COLUMN`, PostgreSQL uses
  `ALTER COLUMN ... TYPE` (`model/main.go:608-618`, `model/main.go:665-677`).
- Migrations must be re-runnable: a partially failed boot must be able to retry.

---

## Naming Conventions

- Table names follow GORM defaults; explicit overrides use `TableName()` (for example
  `model/user_session.go:61`).
- Columns are `snake_case`. Timestamps are unix seconds stored in `int64` columns with
  `autoCreateTime` / `autoUpdateTime` tags (`model/user.go:109`: `gorm:"autoCreateTime;column:created_at"`).
- Let GORM generate primary keys — do not write `AUTO_INCREMENT` or `SERIAL` in DDL.

---

## Common Mistakes

- `tx.Set("gorm:query_option", "FOR UPDATE")` — the GORM v1 pattern. GORM v2 silently ignores it and
  **no lock is taken**. Always call `lockForUpdate(tx)`.
- Duplicating `clause.Locking{Strength: "UPDATE"}` at call sites instead of using the shared helper.
- `gorm:"default:true"` on booleans that encode a business rule: MySQL and PostgreSQL normalize boolean
  defaults differently, so GORM `AutoMigrate` keeps issuing `ALTER TABLE` on every restart. Set such
  defaults in request normalization, model hooks, constructors, or service logic instead.
- `default:1` as a "fix" for the above without verifying behavior on all three databases.
- Dialect-specific SQL without a fallback: MySQL-only functions, PostgreSQL-only operators,
  database-specific JSON column types without a `TEXT` fallback.
- Testing only against the default SQLite dev database and shipping MySQL/PostgreSQL breakage.
- Unstated assumption that `DB` and `LOG_DB` are the same connection.
