# Design

The usage-log backend already serializes `RelayInfo.ReasoningEffort` to `other.reasoning_effort`, and the usage-log detail dialog already renders that field. The gap is in the OpenAI Chat adaptor: it currently copies the request value into `RelayInfo` only inside the O-series/GPT-5 compatibility branch.

After model-specific normalization, copy a non-empty Chat Completions `request.ReasoningEffort` into `info.ReasoningEffort` for every OpenAI-compatible model. Leave Responses handling and model-suffix normalization unchanged.

Cover the regression with the narrowest available adaptor test, then run the OpenAI package tests and verify the deployed service health. A live upstream request remains optional until explicitly authorized because it may incur provider cost.
