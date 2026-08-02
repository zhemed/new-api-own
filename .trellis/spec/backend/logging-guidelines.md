# Logging Guidelines

## Logger and Levels

Use the project logger in `logger/logger.go` for request-aware application logs:

- `logger.LogInfo(ctx, message)` for normal lifecycle and operational events.
- `logger.LogWarn(ctx, message)` for recoverable anomalies or rejected unsafe
  conditions.
- `logger.LogError(ctx, message)` for failed operations requiring attention.
- `logger.LogDebug(ctx, format, args...)` only for diagnostics when debug mode
  is enabled.

Use `common.SysLog`, `common.SysError`, and related helpers for startup and
low-level shared-package events when there is no request context. Match the
existing package and nearby call sites rather than introducing another logger.

## Request Context and Output

Pass the request context to logger calls when available. The logger extracts the
request ID from `common.RequestIdKey` and writes a line containing the level,
timestamp, request ID, and message. Configured files use names like
`oneapi-YYYYMMDDHHMMSS.log`; output is also written to the process streams.

Use stable event context in messages: operation, resource identifier, and the
relevant error. Keep messages concise and avoid logging a whole request or
response body by default.

## Sensitive Data

Never log passwords, access tokens, provider API keys, cookies, authorization
headers, full email verification data, or raw request bodies. Mask URLs and
network errors with `common.MaskSensitiveInfo` when they may contain secrets.
Use bounded previews such as `common.LocalLogPreview` for upstream bodies.

Billing saturation is an auditable event: use the existing checked quota
helpers, attach the clamp to `admin_info.quota_saturation`, and emit the
request-correlated warning through the established service path.

## Common Mistakes

- Calling `fmt.Println` or creating a new package logger for ordinary backend
  events.
- Logging the same error at every layer.
- Logging an entire provider response or a credential-bearing URL.
- Using debug logging for required audit or billing events, because debug logs
  may be disabled.
