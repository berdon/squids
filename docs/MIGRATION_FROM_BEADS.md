# Migration Guide: beads (`bd`) → squids (`sq`)

## Why migrate

Squids provides a beads-compatible task workflow on top of a local SQLite backend.

Design choices:
- Keep familiar task command UX.
- Remove dolt/server lifecycle complexity.
- Support concurrent access via SQLite (WAL + busy timeout).

## Command mapping

| beads | squids |
|---|---|
| `bd init --prefix bd --json` | `sq init --prefix bd --json` |
| `bd ready --json` | `sq ready --json` |
| `bd create ... --json` | `sq create ... --json` |
| `bd show <id> --json` | `sq show <id> --json` |
| `bd list --json --flat --no-pager` | `sq list --json --flat --no-pager` |
| `bd update <id> ... --json` | `sq update <id> ... --json` |
| `bd close <id> --reason ... --json` | `sq close <id> --reason ... --json` |

## Not supported (intentional)

These are intentionally excluded from squids:
- `bd dolt ...` command family
- Server startup/shutdown orchestration
- dolt-specific replication/storage controls

## Install / build

From repo root:

```bash
make build
./bin/sq --help
```

## Smoke test

```bash
./bin/sq init --json
./bin/sq ready --json
./bin/sq create "Test task" --type task --priority 1 --json
./bin/sq list --json --flat --no-pager
```

## Compatibility verification

Use the built-in dual-target runner to compare beads and sq behavior:

```bash
./scripts/parity/compat-runner.sh
```

Logs are written to `.parity-results/`.

## Operational notes

- Default DB path: `./.sq/tasks.sqlite`
- Override path with `SQ_DB_PATH=/path/to/tasks.sqlite`
- SQLite runs in-process (no external sqlite3 binary invocation)

## Current compatibility envelope

Validated by parity automation for:
- init/ready
- create/show/list/update/close
- labels + metadata updates
- dependency linkage via `metadata.upstream`
- claim flow
- missing issue error path
- unknown flag error path
- empty assignee update path

## Rollout recommendation

1. Start new automation/scripts with `sq`.
2. Keep `bd` available during short transition period.
3. Run parity runner in CI for both targets while migrating.
4. Remove direct beads-only assumptions once sq is green in your workflows.
