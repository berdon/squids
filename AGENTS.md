# AGENTS.md — squids

## MANDATORY: Use td for Task Management

Run td usage --new-session at conversation start (or after /clear). This tells you what to work on next.

Sessions are automatic (based on terminal/agent context). Optional:
- td session "name" to label the current session
- td session --new to force a new session in the same context

Use td usage -q after first read.

## Project Overview

**Stack:** Go
**Package Manager:** go

## Directory Layout

```
cmd/
docs/
internal/
scripts/
```

## Key Files

- `Makefile`
- `go.mod`
- `README.md`

## Workflow Conventions

- Do not create pull requests for this repository.
- Do work on a branch, then merge that working branch directly into `master` when it is ready.
- Keep `master` current before branching or merging.
