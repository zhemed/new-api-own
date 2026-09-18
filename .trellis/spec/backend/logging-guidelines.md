# Logging Guidelines

> How logging is done in this project: levels, format, and what must never be logged.

---

## Overview

Two facades, one destination:

| Facade | Use when | Location |
| --- | --- | --- |
| `logger.LogInfo` / `LogWarn` / `LogError` / `LogDebug` | Anything with a `context.Context` or `*gin.Context` (handlers, services, relay) | `logger/logger.go:76-95` |
| `common.SysLog` / `SysError` / `FatalLog` | Startup, migrations, background workers, code without a request context | `common/sys_log.go:17-37` |

About 366 call sites use `logger.Log*` and 362 use `common.SysLog` / `SysError` — both are current; the
difference is whether a request context exists.

---

## Log Levels

| Level | Helper | When |
| --- | --- | --- |
| DEBUG | `logger.LogDebug(ctx, msg, args...)` | Verbose diagnostics; a no-op unless `common.DebugEnabled` (`logger/logger.go:88-95`) |
| INFO | `logger.LogInfo(ctx, msg)` / `common.SysLog` | Normal lifecycle: startup, migrations, rotations, successful state transitions |
| WARN | `logger.LogWarn(ctx, msg)` | Recoverable anomalies: retries, fallbacks, upstream degradation, quota saturation |
| ERROR | `logger.LogError(ctx, msg)` / `common.SysError` | Failed operations that the caller must see |
| FATAL | `common.FatalLog(v...)` | Unrecoverable bootstrap failure; prints then `os.Exit(1)` (`common/sys_log.go:31-37`) |

---

## Structured Logging

`logger` writes one line per event (`logger/logger.go:97-111`):

```
[LEVEL] 2006/01/02 - 15:04:05 | <request-id|SYSTEM> | <message>
```

- The request ID comes from the context value `common.RequestIdKey`; without a context, or without that
  value, the field is literally `SYSTEM` (`logger/logger.go:98-103`).
- `INFO` goes to `gin.DefaultWriter` (stdout); `WARN`, `ERR`, `DEBUG` go to `gin.DefaultErrorWriter`
  (stderr).
- When `LOG_DIR` (`common.LogDir`) is set, both writers are also an appended file
  `oneapi-<timestamp>.log` (`logger/logger.go:42-73`).
- Rotation is line-count driven, not time driven: after `maxLogCount = 1000000` lines a new file is
  opened in the background (`logger/logger.go:27`, `logger/logger.go:112-119`).
- `common.SysLog` / `SysError` print `[SYS] <time> | <msg>` to the same writers.

Message style: build the text with `fmt.Sprintf` and `key=value` pairs, quoting free-form values with
`%q` (`controller/topup_stripe.go:97`, `controller/subscription_payment_stripe.go:84`):

```go
logger.LogError(c.Request.Context(),
	fmt.Sprintf("Stripe 创建 Checkout Session 失败 user_id=%d trade_no=%s amount=%d error=%q",
		id, referenceId, req.Amount, err.Error()))
```

Existing messages mix Chinese and English prose; keep the surrounding file's language, keep identifiers
and keys in English.

---

## What to Log

- Request-scoped failures with the request context, so the line correlates with the request and the
  consume log.
- Upstream failures, retries, channel selection, and fallbacks (`WARN` / `ERROR`).
- Startup, DB init, and migration progress via `common.SysLog` (`model/main.go:203` is the pattern).
- Quota saturation is audited twice by design: the clamp marker is nested into the consume log's
  `other.admin_info.quota_saturation`, and `attachQuotaSaturation` emits a request-correlated
  `logger.LogWarn` (`service/log_info_generate.go:25-37`). New billing paths must surface clamps the
  same way.
- Admin action auditing is persisted to the database through `middleware/audit.go`, not to the text log.

---

## What NOT to Log

- Secrets of any kind: API keys, access tokens, passwords, cookies, webhook signing secrets, upstream
  credentials.
- Full request/response bodies or streamed payloads — use `common.LocalLogPreview`, which truncates to
  `LocalLogContentLimit = 2048` bytes unless debug logging is on (`common/str.go:14-22`).
- Anything that lets a log reader reconstruct a user's prompt content or recover a key.

---

## Legacy Patterns (do not copy)

- The stdlib `log` package still appears in about 17 non-test spots (`relay/mjproxy_handler.go:94`,
  `common/utils.go:40`, `logger/logger.go:49`). New code uses `logger.Log*` or `common.SysLog`.
- `fmt.Println` for operational output.

---

## Common Mistakes

- `logger.LogError(nil, ...)` or dropping the context inside a handler — the line loses its request ID.
- `common.SysLog` inside a request path (no request ID), or `logger.Log*` in startup code (no context).
- Logging debug content without `LogDebug`, which bypasses the `DebugEnabled` gate.
- Logging the same error twice, once in the service and once in the controller, with different wording.
- `%v` on a whole struct that contains secrets or payloads.
