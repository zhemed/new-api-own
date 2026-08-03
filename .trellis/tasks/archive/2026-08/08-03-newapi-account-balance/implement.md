# Implementation

1. Add the account usage controller handler.
2. Register the token-authenticated account usage route.
3. Add controller tests for successful quota projection and database errors.
4. Run gofmt and the focused controller test.
5. Build/deploy the New API container and verify the live endpoint with a
   sanitized request.
6. Add the My Codex CC Switch usage script, back up the database, and verify
   the live provider without changing the active provider.
7. Validate and finish the Trellis task, commit, push, and verify the remote
   commit pointer.
