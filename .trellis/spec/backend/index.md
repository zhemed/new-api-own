# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

This directory contains the backend conventions for this repository (Go API gateway: Gin + GORM v2 +
React frontend under `web/`). The rules here are extracted from the project's own `AGENTS.md`,
`web/AGENTS.md`, and the real code; every example points at an existing file and line.

The authoritative source for anything not covered here is `AGENTS.md` at the repository root.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module organization, where new code goes, `relaykit` independence | Done |
| [Database Guidelines](./database-guidelines.md) | GORM patterns, three-dialect compatibility, migrations | Done |
| [Error Handling](./error-handling.md) | `common.ApiError*` surface, `types.NewAPIError`, upstream error conversion | Done |
| [Quality Guidelines](./quality-guidelines.md) | Forbidden/required patterns, billing invariants, tests, governance, CI commands | Done |
| [Logging Guidelines](./logging-guidelines.md) | `logger` vs `common.SysLog`, format, rotation, what never to log | Done |

---

## Pre-Development Checklist

Read the guides that match the work you are about to do — not all of them.

| Work you are starting | Read |
| --- | --- |
| Any backend change | [quality-guidelines.md](./quality-guidelines.md) |
| New endpoint / handler / service / model file | [directory-structure.md](./directory-structure.md) |
| Anything touching `model/`, migrations, or raw SQL | [database-guidelines.md](./database-guidelines.md) |
| Anything touching `controller/`, relay errors, or `types.NewAPIError` | [error-handling.md](./error-handling.md) |
| Anything producing log output (including quota saturation auditing) | [logging-guidelines.md](./logging-guidelines.md) |
| Any change at all | `../guides/index.md` (cross-layer + code-reuse thinking guides) |

Mandatory checks before declaring work done:

```bash
GOWORK=off go vet ./...
GOWORK=off go build ./...
cd relaykit && GOWORK=off go build ./...
make test
```

---

## How to Keep These Guidelines Useful

1. Document what the code **actually does**, not an aspiration.
2. Add a real file+line example for every rule you add.
3. Record forbidden patterns together with the bug they caused.
4. When a long review thread ends with a convention decision, capture it here.
