# New API Account Balance Endpoint

## Goal

Expose the New API account quota that owns a relay API token through a
read-only, token-authenticated endpoint so CC Switch can display the panel
account balance without a dashboard PAT.

## Requirements

- Add GET /api/usage/account/ protected by TokenAuthReadOnly.
- Resolve the account from the authenticated relay token's user ID.
- Return remaining quota, used quota, total quota, quota_per_unit, and USD
  unit metadata.
- Never use the token's unlimited_quota flag to override the account values.
- Do not change the existing /api/usage/token/ or dashboard billing response.
- Do not expose passwords, user access tokens, API keys, or upstream balances.
- Add focused controller coverage for the account response and error path.

## Acceptance Criteria

- [x] The endpoint authenticates with the existing My Codex relay API key.
- [x] The endpoint returns the panel user's real quota and used_quota values.
- [x] An unlimited relay token does not produce an artificial account balance.
- [x] Focused Go tests pass.
- [x] The deployed endpoint returns the expected sanitized JSON.
- [x] The CC Switch My Codex usage query displays the account balance.
- [x] The change is committed and pushed after verification.

## Notes

- Keep this task independent from the existing deployment task.
- Preserve the currently active CC Switch provider and unrelated providers.
