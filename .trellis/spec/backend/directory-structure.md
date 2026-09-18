# Directory Structure

> How backend code is organized in this project.

---

## Overview

Go monolith with a layered architecture: `router → controller → service → model`.
Relay (AI proxy) traffic runs on a parallel path: `router → middleware → relay/*_handler.go → relay/channel/<provider>`.

Two Go modules live in one repository, both built with `GOWORK=off`:

| Module | Path | Module path |
| --- | --- | --- |
| Root app | `.` | `github.com/QuantumNous/new-api` (Go 1.25.1, Gin 1.9, GORM v2) |
| Standalone relay primitives | `relaykit/` | `github.com/QuantumNous/new-api/relaykit` |

The root module embeds the built frontend (`web/dist`), so a root build needs
`web/dist/index.html` to exist — CI creates an empty placeholder for that reason
(`.github/workflows/ci.yml:38-41`).

---

## Directory Layout

```
.
├── main.go                  # entry: CLI/flags, DB init, HTTP server bootstrap
├── router/                  # route registration only (api-, relay-, web-, dashboard-, video-, channel-, authz-router.go)
├── controller/              # HTTP handlers for the dashboard/management API
├── middleware/              # Gin middleware: auth, authz, rate limit, CORS, i18n, distributor, audit, gzip
├── service/                 # business logic: quota/billing, tasks, auth sessions, oauth, channel selection
├── model/                   # GORM models + ALL database access, one file per entity
├── relay/                   # AI relay/proxy path
│   ├── *_handler.go         # per-protocol entry handlers (chat_completions, responses, claude, gemini, image, audio, ...)
│   ├── channel/             # one package per upstream provider (openai/, claude/, gemini/, aws/, ...)
│   ├── common/              # RelayInfo and shared relay state/utilities
│   ├── helper/              # request validation and response helpers
│   ├── constant/            # relay-specific constants
│   └── common_handler/      # handlers shared across protocols
├── relaykit/                # standalone module: dto/, types/, relayconvert/, kitutil/
├── setting/                 # runtime configuration registries (model, ratio, operation, system, performance)
├── common/                  # cross-cutting utilities: JSON wrapper, crypto, redis, env, gin helpers, quota math
├── dto/                     # task-style API DTOs (midjourney.go, suno.go, task.go, video.go)
├── constant/                # API/channel type constants and gin context keys
├── types/                   # root-module value types (set.go, rw_map.go, price_data.go)
├── i18n/                    # backend translations (go-i18n): locales/, keys.go
├── oauth/                   # OAuth provider implementations
├── logger/                  # logging facade + log rotation
├── pkg/                     # self-contained internal packages (billingexpr/, cachex/, ionet/, ...)
├── web/                     # React 19 frontend (Bun + Rsbuild); see web/AGENTS.md
└── electron/                # desktop wrapper
```

`dto/` (root) holds task/callback payloads; `relaykit/dto/` holds the
OpenAI/Claude/Gemini protocol DTOs used by the relay layer. When a file needs both,
alias one — `taskdto "github.com/QuantumNous/new-api/dto"` in `service/error.go:15`.

---

## Where new code goes

| You are adding | Put it in | Notes |
| --- | --- | --- |
| Dashboard/management endpoint | `controller/` + route in `router/api-router.go` (or the matching `*-router.go`) | Handlers stay thin: parse → call `service/` → `common.ApiSuccess` / `common.ApiError` |
| Relay endpoint for a new protocol | `relay/<protocol>_handler.go` + route in `router/relay-router.go` | Reuse `relay/helper` validation; bound user-controlled multipliers (`quality-guidelines.md`) |
| New upstream provider | `relay/channel/<provider>/` | Implement the channel adaptor, register it in `relay/relay_adaptor.go:GetAdaptor` (`relay/relay_adaptor.go:56`), and add the channel type to `streamSupportedChannels` when the provider supports stream options (`relay/common/relay_info.go:328`) |
| Business rule (quota, billing, task settlement, channel selection) | `service/` | No SQL here — call `model/` |
| New table/entity or query | `model/<entity>.go` | Add the struct to the `AutoMigrate` list in `model/main.go` |
| Protocol DTO or conversion shared with the standalone module | `relaykit/` | Must not import the root module |
| Task-style request/response payload | `dto/` | |
| Cross-cutting helper | `common/` | Search for an existing helper first (`../guides/code-reuse-thinking-guide.md`) |
| Runtime configuration knob | `setting/` | |
| Backend user-facing message | `i18n/keys.go` + `i18n/locales/*` | Paired with `common.ApiErrorI18n` |
| Frontend user-facing string | `web/src/i18n/locales/{lang}.json` | English source string as the key; see `web/AGENTS.md` |

---

## Layering rules

- `router/` only registers routes and middleware chains — no business logic.
- `controller/` parses the request, calls `service/`, writes the response. It MUST NOT run raw SQL.
- `service/` owns business decisions and may call `model/` and other services; it must not import `controller/`.
- `model/` owns all GORM access; queries are exported functions on the entity (for example `model/channel.go`).
- `relay/` handlers compose `relay/helper`, `relay/common`, and a provider adaptor from `relay/channel/`.
- Shared low-level code lives in `common/` or `pkg/` and must stay free of import cycles with the layers above.

---

## Naming conventions

- Files: lowercase `snake_case` (`channel_upstream_update.go`, `billing_session.go`).
- Packages: short, no underscores (`operation_setting`, `billing_setting`); the relay common package is imported as `relaycommon`.
- One entity per file in `model/` (`user.go`, `token.go`, `channel.go`, `log.go`).
- Custom table names use a `TableName()` method (`model/user_session.go:61`), not a global naming-strategy override.
- Tests sit next to the code as `<name>_test.go` (154 such files outside `web/`).
- Backend i18n keys are constants in `i18n/keys.go`; frontend keys are English source strings.

---

## Hard rule: `relaykit/` independence

`relaykit/` is its own Go module and MUST stay independently buildable:

- No imports of root-module packages (`github.com/QuantumNous/new-api/<pkg>`) from inside `relaykit/`.
- No dependency on root-only configuration, generated files, or workspace wiring.
- Verify with `cd relaykit && GOWORK=off go build ./...` — a green root build proves nothing here.

---

## Examples to copy from

- Entity layering end to end: `controller/channel.go` → `service/channel.go` → `model/channel.go`.
- Provider adaptor package layout: `relay/channel/openai/`, `relay/channel/claude/`, `relay/channel/aws/`.
- Shared DB helper placement: `model/locking.go` (one small file, documented helper).
- Cross-cutting gin helpers: `common/gin.go` (context key accessors, API responses).
