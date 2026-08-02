# Implementation Plan

1. Read the live source, Compose state, and existing task context.
2. Confirm the current OpenAI Chat adaptor drops reasoning effort for `deepseek-v4-flash` while the log serializer and frontend detail view already support the field.
3. Add the generic `RelayInfo` assignment and a focused regression test.
4. Run targeted Go tests and build the New API image using the repository's Compose workflow.
5. Verify service health and existing log state without switching providers or sending an unapproved paid request.
6. Validate the Trellis task and record a sanitized journal entry after the user authorizes or performs a live request.
