# Backend Development Guidelines

These guidelines describe the conventions already used by the new-api backend. The
repository's `AGENTS.md` remains the broader source of truth; this directory gives
Trellis focused context for implementation and review tasks.

## Stack and Boundaries

- Go backend using Gin, GORM v2, Redis, and SQLite/MySQL/PostgreSQL.
- `relaykit/` is a separate Go module and must remain independently buildable.
- `web/` is the React frontend and uses Bun for package management and scripts.
- The normal backend flow is Router -> Controller -> Service -> Model. Relay
  provider adapters are under `relay/channel/`.

## Guides

| Guide | Focus | Status |
| --- | --- | --- |
| [Directory Structure](./directory-structure.md) | Package ownership and file layout | Complete |
| [Database Guidelines](./database-guidelines.md) | GORM, transactions, migrations, dialects | Complete |
| [Error Handling](./error-handling.md) | API errors and relay errors | Complete |
| [Logging Guidelines](./logging-guidelines.md) | Levels, request IDs, sensitive data | Complete |
| [Quality Guidelines](./quality-guidelines.md) | Tests, checks, forbidden patterns | Complete |

## Before Changing Backend Code

1. Read the applicable guide and the nearest existing implementation.
2. Preserve the three-database contract and the independent `relaykit` module.
3. For billing or quota changes, read `pkg/billingexpr/expr.md` and trace
   validation, pre-consume, settlement, and refund paths.
4. Add a deterministic regression test for a user-visible or cross-module
   behavior, then run the narrowest relevant checks and `make test` when practical.
