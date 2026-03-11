# Migration Guide: beads (`bd`) → squids (`sq`)

## Why migrate

Squids provides a beads-compatible task workflow on top of a local SQLite backend.

Design choices:
- Keep familiar task command UX.
- Remove dolt/server lifecycle complexity.
- Support concurrent access via SQLite (WAL + busy timeout).

## What changes operationally

With beads, task data commonly depends on Dolt/server mechanics. With squids:
- Data is local SQLite (`.sq/tasks.sqlite` by default).
- No `dolt start/stop` style lifecycle required.
- CLI commands run in-process against SQLite.

## Command mapping (supported today)

| beads | squids |
|---|---|
| `bd init --prefix bd --json` | `sq init --json` |
| `bd ready --json` | `sq ready --json` |
| `bd create ... --json` | `sq create ... --json` |
| `bd show <id> --json` | `sq show <id> --json` |
| `bd list --json --flat --no-pager` | `sq list --json --flat --no-pager` |
| `bd update <id> ... --json` | `sq update <id> ... --json` |
| `bd close <id> --reason ... --json` | `sq close <id> --reason ... --json` |
| `bd reopen <id> --json` | `sq reopen <id> --json` |
| `bd delete <id> --force --json` | `sq delete <id> --force --json` |
| `bd label add/list/remove/list-all ... --json` | `sq label add/list/remove/list-all ... --json` |
| `bd dep add/list/remove ... --json` | `sq dep add/list/remove ... --json` |
| `bd comments add <id> <text> --json` | `sq comments add <id> <text> --json` |
| `bd comments <id> --json` | `sq comments <id> --json` |
| `bd query "..." --json` | `sq query "..." --json` |
| `bd search "..." --json` | `sq search "..." --json` |
| `bd count --json` | `sq count --json` |
| `bd status --json` | `sq status --json` |
| `bd todo ... --json` | `sq todo ... --json` |
| `bd children <id> --json` | `sq children <id> --json` |
| `bd blocked --json` | `sq blocked --json` |
| `bd duplicate <id> --of <canonical> --json` | `sq duplicate <id> --of <canonical> --json` |
| `bd supersede <id> --with <replacement> --json` | `sq supersede <id> --with <replacement> --json` |
| `bd types --json` | `sq types --json` |

## Not supported (intentional)

These are intentionally excluded from squids:
- `bd dolt ...` command family
- Server startup/shutdown orchestration
- Dolt-specific replication/storage controls

## Getting started migration flow

1. Build squids:
   ```bash
   make build
   ```
2. In your target repo, initialize squids:
   ```bash
   ./bin/sq init --json
   ./bin/sq ready --json
   ```
3. Run your normal lifecycle commands with `sq` instead of `bd`.
4. Keep `bd` available while validating automation.
5. Run dual-target parity during transition:
   ```bash
   ./scripts/parity/compat-runner.sh
   ```

## Verification checklist

- [ ] Core workflows pass with `sq`.
- [ ] CI parity runner is green for both `bd` and `sq` (during migration period).
- [ ] No scripts rely on Dolt/server-specific commands.
- [ ] Team docs and runbooks reference `sq` for task operations.

## Operational notes

- Default DB path: `./.sq/tasks.sqlite`
- Override path with `SQ_DB_PATH=/path/to/tasks.sqlite`
- SQLite runs in-process (no external sqlite3 binary invocation)
