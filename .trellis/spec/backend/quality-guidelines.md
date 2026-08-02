# Quality Guidelines

## Required Checks

Format Go changes with `gofmt`. Run focused tests first, then the repository
checks when the change crosses package boundaries:

```bash
make test
GOWORK=off go test ./...
cd relaykit && GOWORK=off go test ./...
```

The `relaykit/` module must be tested independently; a successful root build or
test does not prove relaykit is healthy. For frontend changes use Bun from
`web/` and follow `web/AGENTS.md`.

## Code Rules

- Keep code direct and readable; prefer early returns and clear branches.
- Do not add one-caller helpers unless they represent a durable domain concept,
  framework callback, reusable behavior, or logic worth testing directly.
- All JSON marshal/unmarshal operations in business code go through
  `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`,
  `common.DecodeJson`, or `common.GetJsonType` in `common/json.go`.
- Optional scalar request fields that must preserve explicit zero or false values
  use pointers with `omitempty`.
- New provider channels must verify StreamOptions support and update
  `streamSupportedChannels` when appropriate.

## Tests

Tests must protect user-visible behavior, API contracts, billing/accounting
invariants, database compatibility, or a real regression. Prefer deterministic
table tests with explicit inputs. Initialize database, request, auth, settings,
and cache state in the fixture rather than relying on global test order.

Use `require` for setup and fatal assertions and `assert` for non-fatal value
checks, as shown in `controller/token_auto_groups_test.go` and
`common/quota_math_test.go`. Keep regression tests next to the boundary they
protect; quota overflow, validation bypasses, and relay conversion boundaries
already have representative tests in `common/`, `relay/helper/`, and `relaykit/`.

## Forbidden Patterns

- Direct `encoding/json` marshal/unmarshal calls in business code.
- Bare numeric casts for unbounded quota or token calculations.
- Random, sleep-based, timing-sensitive, or log-only tests that do not assert a
  contract.
- Database-specific SQL without a valid fallback for every supported database.
- Root-module imports from `relaykit/`.
- Changing protected upstream project identity, attribution, license, module
  paths, or provider behavior merely to rename the fork.

## Review Checklist

Before committing, verify the change's layer ownership, error and log behavior,
database dialect compatibility, secret handling, tests, and independent relaykit
buildability. For public API or billing changes, add or update a regression test
that fails before the fix and passes afterward.
