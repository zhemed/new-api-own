# Directory Structure

## Overview

The backend uses a layered layout. Keep HTTP transport concerns in controllers,
business rules in services, persistence in models, and provider translation in
relay adapters. Do not move business logic into router registration or model
types merely to shorten a controller.

## Directory Layout

```text
router/       Route registration and HTTP route groups
controller/   Gin handlers and request/response orchestration
service/      Business workflows, billing, auth, and external operations
model/        GORM models, database access, migrations, and persistence helpers
relay/        Provider relay logic and channel adapters
relaykit/     Independent Go module for reusable relay conversion
middleware/   Auth, rate limits, request IDs, logging, CORS, and recovery
common/       Shared JSON, database, cache, validation, crypto, and HTTP helpers
dto/          Request and response data-transfer structs
constant/     Shared constants and context keys
types/        Relay formats, errors, and shared type definitions
setting/      Runtime and operation settings
oauth/        OAuth provider implementations
logger/       Application logging
pkg/          Internal reusable packages such as billingexpr
web/          React frontend; follow web/AGENTS.md
```

The backend module is rooted at `main.go` and `go.mod`. `relaykit/go.mod` defines
the separate module boundary. Do not import root-module packages from `relaykit/`.

## Feature Placement

- Add routes in `router/`, handlers in `controller/`, and reusable business
  operations in `service/`.
- Add or change persisted entities in `model/`; keep database-specific logic
  behind the existing database helpers.
- Add provider-specific translation under the matching directory in
  `relay/channel/`. Shared relay behavior belongs in `relay/common/` or
  `relay/helper/`.
- Put shared JSON operations in `common/json.go`, not in individual features.
- Put request/response structs in the existing `dto/` or feature-local DTO
  package instead of duplicating anonymous maps across controllers.

## Naming and Examples

Use short, descriptive lower-case Go filenames with underscores only when the
existing feature uses them, for example `controller/channel.go`,
`controller/channel_authz.go`, and `middleware/request-id.go`. Tests use the
standard `_test.go` suffix, such as `router/channel_router_test.go` and
`model/locking_test.go`.

Representative flows are `router/` route registration into
`controller/channel.go`, controller operations that call `service/`, and
database setup and migration in `model/main.go`. Provider-specific behavior is
represented by packages under `relay/channel/`.
