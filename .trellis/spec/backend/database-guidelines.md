# Database Guidelines

## Supported Databases

The primary database must work with SQLite, MySQL >= 5.7.8, and PostgreSQL >=
9.6. The optional log database may also use ClickHouse. Database selection and
dialect setup live in `model/main.go` through `chooseDB`, `initCol`, and the
`common.UsingMainDatabase` / `common.UsingLogDatabase` helpers.

## Query and Transaction Patterns

- Use GORM v2 methods (`Create`, `Find`, `Where`, `Updates`, `Transaction`) for
  normal persistence.
- Keep row-locking queries inside a transaction and call the shared
  `lockForUpdate(tx)` helper in `model/locking.go`. It emits `FOR UPDATE` for
  MySQL/PostgreSQL and skips it for SQLite.
- Use `model.DB.Transaction` or the transaction handle passed into a workflow;
  do not mix the global DB handle into a transaction callback.
- Let GORM create primary keys. Do not introduce `AUTO_INCREMENT` or `SERIAL`.
- Prefer model methods and shared query helpers over duplicated SQL in
  controllers or services.

Example from `model/locking.go`:

```go
func lockForUpdate(tx *gorm.DB) *gorm.DB {
    if common.UsingMainDatabase(common.DatabaseTypeSQLite) {
        return tx
    }
    return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}
```

## Raw SQL and Migrations

Raw SQL is allowed only when GORM cannot express the operation. Every raw query
must account for dialect differences. Use `commonGroupCol`, `commonKeyCol`,
`commonTrueVal`, and `commonFalseVal` for reserved columns and boolean literals.
Do not use MySQL-only functions, PostgreSQL-only operators, SQLite-unsupported
`ALTER COLUMN`, or database-specific JSON types without a compatible fallback.

Schema changes belong in the existing migration flow in `model/main.go` and
must be safe for all supported databases. SQLite migrations use
`ALTER TABLE ... ADD COLUMN` patterns rather than `ALTER COLUMN`.

Avoid GORM boolean default tags such as `gorm:"default:true"` when the default
is a business rule. Normalize defaults in request validation, constructors,
hooks, or service logic so AutoMigrate does not repeatedly alter schemas across
MySQL and PostgreSQL.

## Billing and Data Safety

Quota values are 32-bit database values. Validate user-controlled multipliers
before billing and use `common.QuotaFromFloat`, `common.QuotaRound`, or
`common.QuotaFromDecimal` instead of bare numeric casts. Use the `*Checked`
variants when a billing path needs to audit saturation, and attach the clamp to
the consume/task log as described in `AGENTS.md`.

When changing billing, trace validation -> estimate -> pre-consume -> settle or
refund. Never allow overflow or invalid input to become a negative charge.

## Examples and Forbidden Patterns

- `model/main.go` is the reference for database selection, migration, and
  dialect-specific column names.
- `model/locking.go` is the reference for row locks.
- `controller/oauth.go` demonstrates a transaction callback for multi-step
  writes.
- Do not use the legacy GORM v1 `Set("gorm:query_option", "FOR UPDATE")`;
  GORM v2 silently ignores it.
- Do not write to `PriceData.OtherRatios` directly; use
  `types.PriceData.AddOtherRatio`.
