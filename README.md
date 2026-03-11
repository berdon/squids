# squids

SQLite-backed, beads-compatible task CLI.

## Why squids exists

`sq` exists to preserve the task workflow ergonomics of beads while removing Dolt/server operational complexity.

Goals:
- Keep familiar command semantics for day-to-day task operations.
- Use a local SQLite backend (single file, no daemon lifecycle).
- Support concurrent local access safely (WAL + busy timeout).
- Grow compatibility with test-first parity automation.

## Current status

Squids is actively expanding parity coverage against beads via shell automation.

- Binary name: `sq`
- Backend: SQLite (`.sq/tasks.sqlite` by default)
- Intentional exclusions: Dolt/server command families

## Build

```bash
make build
./bin/sq --help
```

## Supported commands (today)

Core lifecycle:
- `init`
- `ready`
- `create`
- `show`
- `list`
- `update`
- `close`
- `reopen`
- `delete`

Command families and views:
- `label` (`add`, `remove`, `list`, `list-all`)
- `dep` (`add`, `remove`, `list`)
- `comments` (`add`, `list`)
- `todo` (`add`, `list`, `done`)
- `children`
- `blocked`
- `duplicate`
- `supersede`

Query/reporting:
- `query`
- `search`
- `count`
- `status`
- `types`

## Known gaps vs beads

Squids intentionally does **not** implement:
- `bd dolt ...` command family
- Dolt server lifecycle/start-stop operations
- Dolt-specific history/replication/storage controls

Squids also does not yet claim full parity for the broader beads surface area beyond the commands listed above.

## Next steps

Near-term priorities:
1. Continue expanding parity coverage command-by-command.
2. Tighten edge-case compatibility (error messages, filtering behavior, advanced flags).
3. Add richer integration and stress tests for concurrent mutation scenarios.
4. Keep docs and compatibility contract synced with parity suite growth.

## Parity shell automation

### Run against beads

```bash
./scripts/parity/run-parity.sh
```

### Run against squids binary (`sq`)

```bash
TARGET_BIN=./bin/sq ./scripts/parity/run-parity.sh
```

### Dual-target compatibility run (beads vs sq)

```bash
./scripts/parity/compat-runner.sh
```

Outputs logs and deltas under `.parity-results/`.

### SQLite concurrency smoke test

```bash
./scripts/parity/concurrency-smoke.sh
```

### Optional target args

If your target needs global args before the command:

```bash
TARGET_ARGS="--json" ./scripts/parity/run-parity.sh
```

## Documentation

- Getting started: `docs/GETTING_STARTED.md`
- Migration guide: `docs/MIGRATION_FROM_BEADS.md`
- Compatibility contract: `docs/COMPATIBILITY_CONTRACT.md`
- SQLite concurrency strategy: `docs/SQLITE_CONCURRENCY.md`
- Release notes: `docs/RELEASE_NOTES.md`
