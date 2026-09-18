# Error Handling

> How errors are caught, typed, logged, and returned.

---

## Overview

There are **two error surfaces** with different contracts:

| Surface | Who uses it | Shape |
| --- | --- | --- |
| Dashboard / management API | `controller/` handlers calling `common.*` helpers | HTTP 200 + `{"success": false, "message": "..."}` |
| Relay API (OpenAI/Claude/Gemini/...) | `relay/`, `service/`, `relaykit/` | `*types.NewAPIError` with an `ErrorCode`, `ErrorType`, and HTTP status |

Both surfaces log through the `logger` package (`logging-guidelines.md`).

---

## Error Types

Relay and billing code uses the typed error from the standalone module
(`relaykit/types/error.go:90`):

```go
type NewAPIError struct {
	Err            error
	RelayError     any
	skipRetry      bool
	recordErrorLog *bool
	errorType      ErrorType
	errorCode      ErrorCode
	StatusCode     int
	Metadata       json.RawMessage
}
```

- `Unwrap()` exposes `Err`, so `errors.Is` / `errors.As` keep working (`relaykit/types/error.go:102`).
- Read the classification through the accessors `GetErrorCode()` / `GetErrorType()` — never through
  string matching on the message.
- `ErrorType` and `ErrorCode` are closed enums (`relaykit/types/error.go:26-88`), e.g.
  `ErrorCodeInsufficientUserQuota`, `ErrorCodeChannelNoAvailableKey`, `ErrorCodeDoRequestFailed`,
  `ErrorCodeModelPriceError`. **Add a constant to the enum instead of inventing a string literal.**
- Constructors: `NewError(err, code, opts...)`, `NewErrorWithStatusCode(...)`,
  `InitOpenAIError(code, statusCode)`, `WithOpenAIError(...)` (`relaykit/types/error.go:244-299`).

---

## Error Handling Patterns

- Propagate with `return err`; wrap with `fmt.Errorf("context: %w", err)` when the caller needs
  `errors.Is` / `errors.As` (used in `relay/common/override.go:91`).
- Upstream HTTP failures are converted centrally:
  `service.RelayErrorHandler(ctx, resp, showBodyWhenFail)` (`service/error.go:87`) reads the body,
  tries `dto.GeneralErrorResponse` → OpenAI/Claude/Gemini shapes, and builds a typed `NewAPIError`.
  Protocol-specific wrappers live next to it (`ClaudeErrorWrapper`, `MidjourneyErrorWrapper`, ...).
- Downstream decisions branch on the typed code, so the code must survive the whole path:
  - quota exhaustion: `err.GetErrorCode() == types.ErrorCodeInsufficientUserQuota`
    (`service/billing_session.go:413`, `service/billing_session.go:431`)
  - violation fees: `IsViolationFeeCode(err.GetErrorCode())` (`service/violation_fee.go:65`)
- Panicking is reserved for unrecoverable startup conditions (for example the MySQL charset check,
  `model/main.go:186` and `model/main.go:230`). Use `common.FatalLog` for bootstrap exits
  (`common/sys_log.go:31`) — it logs to stderr and calls `os.Exit(1)`.
- Never swallow errors on the billing chain (validation → estimate → other ratios → quota conversion →
  pre-consume → settle/refund). A dropped error there becomes a wrong charge
  (`quality-guidelines.md`).

---

## API Error Responses

Dashboard endpoints always answer HTTP 200 and signal failure in the body (`common/gin.go:199-229`):

```go
func ApiError(c *gin.Context, err error) {          // common/gin.go:199
	c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
}

func ApiSuccess(c *gin.Context, data any) {         // common/gin.go:213
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": data})
}
```

- Use `common.ApiErrorMsg(c, "…")` for a literal message and `common.ApiError(c, err)` when you have an
  error value (`controller/` alone has 338 `common.ApiError(c, ...)` call sites).
- User-visible, translatable failures use `common.ApiErrorI18n(c, key, args...)` (`common/gin.go:223`)
  with keys from `i18n/keys.go` — do not hand-roll translated strings.
- Relay endpoints must keep the upstream protocol's error envelope; do not leak internal errors,
  upstream response bodies, or credentials. Log the detail, return the safe part.
- For request-scoped logging of the failure use `logger.LogError(ctx, ...)` with the request context
  so the request ID is attached (`logging-guidelines.md`).

---

## Common Mistakes

- Returning a non-200 status (or a new JSON error shape) for dashboard endpoints — the frontend contract
  is `success` + `message` at HTTP 200.
- Converting a `*types.NewAPIError` into a plain `fmt.Errorf(...)`, which silently disables retry,
  refund, and violation-fee logic downstream.
- Comparing error text (`strings.Contains(err.Error(), ...)`) instead of using `GetErrorCode()`.
- Logging the raw request/response body; use `common.LocalLogPreview` (2048-byte cap) instead
  (`common/str.go:14-22`).
- Emitting an upstream error message straight to the client, including upstream payloads.
- Ignoring the error returned by `DB.*` / `DB.Transaction(...)` and continuing as if the write succeeded.
