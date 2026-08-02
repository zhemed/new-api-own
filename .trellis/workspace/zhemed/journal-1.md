# Journal - zhemed (Part 1)

> AI development session journal
> Started: 2026-08-02

---



## Session 1: Reasoning effort usage log maintenance

**Date**: 2026-08-02
**Task**: Reasoning effort usage log maintenance
**Branch**: `main`

### Summary

Persisted OpenAI Chat reasoning_effort for non-O/GPT-5 models, deployed the image, and verified live deepseek-v4-flash logs record high.

### Main Changes

- Record generic OpenAI Chat reasoning effort in RelayInfo and usage logs.
- Add a focused adaptor regression test.

### Git Commits

| Hash | Message |
|------|---------|
| `355fc0b9` | (see git log) |

### Testing

- [OK] go test ./relay/channel/openai ./relay/common ./service
- [OK] Live logs 45-64 recorded reasoning_effort=high.

### Status

[OK] **Completed**

### Next Steps

- Keep the existing deployment task separate from this maintenance archive.
