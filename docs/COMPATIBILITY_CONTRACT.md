# Squids Compatibility Contract (sq-002)

## Scope

Squids is a SQLite-backed reimplementation of beads CLI behavior.

- **Canonical binary name:** `sq`
- Goal: match beads CLI semantics used by Village workflows and shell automation.
- **Dolt/server mechanics are intentionally not implemented** in squids.
- Runtime model is local SQLite file(s) with multi-process concurrency support.

## Compatibility Targets

This contract defines behavior that must match for parity tests to pass.

### Command surface (initial)

- `sq init --prefix bd --json`
- `sq ready --json`
- `sq create ... --json`
- `sq show <id> --json`
- `sq list --json --flat --no-pager`
- `sq update <id> ... --json`
- `sq close <id> [--reason <text>] --json`
- `sq todo [list] --json`
- `sq todo add <title> [--priority <n>] --json`
- `sq todo done <id> [--reason <text>] --json`
- `sq children <parent-id> --json`
- `sq blocked --json`
- `sq duplicate <id> --of <canonical-id> --json`
- `sq supersede <id> --with <replacement-id> --json`
- `sq types --json`

## JSON Contract Rules

1. JSON output must be parseable by standard JSON parsers.
2. Single-entity operations may return object or single-item array if parity harness normalizes; preferred shape for squids is object.
3. Required fields for issue entities:
   - `id`
   - `title`
   - `status`
   - `priority` (when provided)
   - `issue_type` (when provided)
   - `created_at` / `updated_at` timestamps
4. Status lifecycle values (minimum):
   - `open`
   - `in_progress`
   - `closed`
   - `resolved` (accepted where relevant)

## Behavioral Semantics (from shell parity)

### Init/ready
- `init` initializes task store and default config needed for subsequent commands.
- `ready --json` succeeds after init and returns JSON payload/list.

### Create/show
- `create` returns created issue with generated ID.
- `show <id>` returns the matching issue entity.
- Missing issue in `show` must:
  - return non-zero exit code
  - emit error text indicating issue not found.

### Update
- `update <id> --status in_progress --assignee <name>` mutates and returns updated issue.
- Label updates via `--add-label` preserve and append labels.
- Metadata updates via `--set-metadata key=value` persist and are visible in returned JSON.
- `--claim` transitions issue into active/in-progress ownership semantics.

### List
- `list --json --flat --no-pager` returns entries including newly created/updated issues.

### Close
- `close <id> --reason <text>` sets terminal status to `closed` and records reason.
- `show` after close reflects closed status.

### Todo convenience wrapper
- `todo add <title> ...` creates an `issue_type=task` entry (default priority 2 unless overridden).
- `todo` / `todo list` returns open `task` issues.
- `todo done <id>` closes the referenced todo issue.

### Children / blocked views
- `children <parent-id>` lists child issues linked by `parent-child` dependency edges.
- `blocked` lists open issues that are currently blocked by open blockers (`blocks` dependency edges), with `blocked_by_count` and `blocked_by` IDs.

### Duplicate / supersede flows
- `duplicate <id> --of <canonical-id>` closes `<id>` and records a `duplicates` relationship to the canonical issue.
- `supersede <id> --with <replacement-id>` closes `<id>` and records a `supersedes` relationship to the replacement issue.

### Types listing
- `types` returns supported core issue types via a JSON object containing `core_types`.

## Exit Code Contract

- Success paths: exit code `0`
- Invalid/missing entity operations: non-zero
- Parsing/validation failures: non-zero

## Normalization Rules (test harness)

To reduce backend formatting noise while preserving semantic checks:
- Treat top-level single-item JSON arrays and plain objects equivalently for field assertions.
- Compare semantic fields, not field order.
- Ignore cosmetic warning lines unless explicitly contract-bound.

## Non-Goals / Explicit Exclusions

The following beads mechanisms are explicitly excluded from squids:
- Dolt server controls and diagnostics commands:
  - `bd dolt start/stop/test/set ...`
- Any server lifecycle management requirement
- Networked/multi-node dolt replication behavior

These are not deferred features; they are intentionally omitted by design.

## Concurrency Requirements

Squids must support concurrent access from multiple processes/actors on the same host.

Baseline requirements:
- SQLite WAL mode enabled
- Busy timeout configured
- Retry/backoff behavior for lock contention in write paths
- Short transactions with deterministic commit boundaries
- No requirement for external DB server process

## Evolution Policy

As new parity cases are added:
1. Add failing shell automation first.
2. Update this contract with expected behavior.
3. Implement squids behavior to satisfy updated contract.
