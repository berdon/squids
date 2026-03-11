# Getting Started with squids (`sq`)

## Why squids exists

`sq` exists to keep the **beads task workflow** people already use, while removing the Dolt/server operational overhead.

In short:
- Same task-centric CLI ergonomics.
- Local single-file SQLite backend.
- No server lifecycle management.
- Concurrency-safe for multi-process local use.

## Install / build

From repo root:

```bash
make build
./bin/sq --help
```

## Initialize a workspace

By default, squids stores data at `./.sq/tasks.sqlite` in your current repo/folder.

```bash
./bin/sq init --json
./bin/sq ready --json
```

## Basic workflow

Create + move a task through a normal lifecycle:

```bash
./bin/sq create "Ship migration docs" --type task --priority 1 --description "Document sq rollout" --json
./bin/sq list --json --flat --no-pager
./bin/sq update <id> --status in_progress --assignee guppy --json
./bin/sq close <id> --reason "Done" --json
```

## Useful command families

```bash
# labels
./bin/sq label add <id> triage --json
./bin/sq label list <id> --json

# dependencies
./bin/sq dep add <id> <depends-on-id> --json
./bin/sq dep list <id> --json

# comments
./bin/sq comments add <id> "needs review" --json
./bin/sq comments <id> --json

# convenience
./bin/sq todo add "Follow up" --json
./bin/sq todo --json
./bin/sq todo done <id> --json
```

## Compatibility checks

Run parity against sq only:

```bash
TARGET_BIN=./bin/sq ./scripts/parity/run-parity.sh
```

Run dual-target parity (`bd` vs `sq`):

```bash
./scripts/parity/compat-runner.sh
```

Run SQLite concurrency smoke:

```bash
./scripts/parity/concurrency-smoke.sh
```

## Environment

- `SQ_DB_PATH=/path/to/tasks.sqlite` to override DB location.

## Next docs

- Migration: `docs/MIGRATION_FROM_BEADS.md`
- Compatibility contract: `docs/COMPATIBILITY_CONTRACT.md`
