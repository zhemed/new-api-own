# Design

## Endpoint

Add GET /api/usage/account/ under the existing /api/usage group. Apply
TokenAuthReadOnly so the relay key is resolved to a user while disabled and
invalid tokens retain the existing authentication behavior.

## Data

Read the account remaining quota through model.GetUserQuota with the normal
cache-aware path and read used quota through model.GetUserUsedQuota. Return the
raw quota fields plus common.QuotaPerUnit and unit=USD. The endpoint must not
reuse GetSubscription because that response intentionally supports token-level
unlimited behavior and can return a synthetic 100000000 value.

## Compatibility

Keep /api/usage/token/ and /dashboard/billing/subscription unchanged. The new
response is additive and intended for a CC Switch usage script.

## Verification

Use a focused controller test with an isolated database, then query the live
endpoint with the existing My Codex key and inspect only status and non-secret
balance fields.
