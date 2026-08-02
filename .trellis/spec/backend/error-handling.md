# Error Handling

## General Rules

Use Go errors as values. Return errors from lower layers with context and let the
caller decide whether to retry, translate, log, or expose them. Use `%w` when
callers need `errors.Is` or `errors.As`, and preserve typed errors such as the
sentinels and domain errors in `model/` and `service/`.

Controllers validate input and convert failures into the project's API response
shape. Services and models should not write Gin responses or depend on a
request-specific response format.

## API Responses

The existing API commonly returns HTTP 200 with a JSON envelope. Use the shared
helpers in `common/gin.go` when their behavior matches the endpoint:

```go
if err != nil {
    common.ApiError(c, err)
    return
}
common.ApiSuccess(c, data)
```

`ApiError` returns `{ "success": false, "message": "..." }`; success responses
include `success: true` and `data`. Use `ApiErrorI18n` and `ApiSuccessI18n` for
translated messages. Some endpoints intentionally use explicit `c.JSON` with
an HTTP status, so preserve the local contract when modifying an existing
handler.

Bind and validate request data before database writes. Follow the existing
setup flow in `controller/setup.go` for early returns on invalid JSON, mismatched
passwords, and failed persistence.

## Relay and Upstream Errors

Relay code uses typed errors such as `types.NewAPIError` and provider-specific
wrappers. `RelayErrorHandler` parses provider JSON through `common.Unmarshal`,
preserves status information, and logs a bounded body preview when needed.
Do not expose raw upstream response bodies or credentials to clients. Use
`common.MaskSensitiveInfo` or the existing error wrapper before logging or
returning network-related details.

## Logging and Cleanup

Log an error once at the layer that can add useful context. Do not log the same
failure repeatedly in controller, service, and relay layers. Close response
bodies and other resources on both success and failure paths. For a transaction,
return the error from the callback so GORM can roll back.

## Common Mistakes

- Returning an error without checking it after a database write.
- Leaking provider response bodies, access tokens, passwords, or API keys in an
  error message.
- Replacing a typed error with a string and breaking `errors.Is` / `errors.As`.
- Changing an established endpoint's envelope or status behavior without a
  regression test.
