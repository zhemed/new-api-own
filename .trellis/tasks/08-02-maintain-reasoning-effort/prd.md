# Record reasoning effort in usage logs

## Goal

Persist the requested reasoning effort in New API usage logs for OpenAI-compatible requests, including non-O/GPT-5 models such as `deepseek-v4-flash`.

## Requirements

- Record an incoming OpenAI Chat Completions `reasoning_effort` value in the request's `other.reasoning_effort` log field regardless of model family.
- Preserve existing model-suffix parsing and OpenAI Responses reasoning handling.
- Keep credentials, provider selection, channel settings, and unrelated logs unchanged.
- Keep the existing usage-log detail rendering contract; it already renders `other.reasoning_effort` when present.

## Acceptance Criteria

- [x] A focused automated test covers a non-O/GPT-5 OpenAI Chat request with a reasoning effort and verifies it reaches `RelayInfo`.
- [x] Relevant Go tests pass.
- [x] A live panel request creates usage logs whose sanitized `other` data contains the requested reasoning effort and the UI can display it.
- [x] Trellis validation succeeds and deployment is verified without switching CC Switch providers.
