# Getting Started with squids (`sq`)

This guide gets you from zero to an active workflow in a few minutes.

## 1) Build and verify

From repo root:

```bash
make build
./bin/sq --help
```

## 2) Initialize a workspace

By default, squids stores data at `./.sq/tasks.sqlite` in your current repo/folder.

```bash
./bin/sq init --json
```

## 3) Understand and use `ready`

`ready` is your “what can I work on now?” view.

It returns tasks that are:
- status `open`
- not dependency-blocked

```bash
./bin/sq ready --json
```

Tip: start each work session with `sq ready --json`.

## 4) Basic task lifecycle

```bash
./bin/sq create "Ship migration docs" --type task --priority 1 --description "Document sq rollout" --json
./bin/sq list --json --flat --no-pager
./bin/sq show <id> --json
./bin/sq update <id> --status in_progress --assignee guppy --json
./bin/sq close <id> --reason "Done" --json
```

## 5) Common command families

```bash
# labels
./bin/sq label add <id> triage --json
./bin/sq label list <id> --json

# dependencies
./bin/sq dep add <issue-id> <depends-on-id> --json
./bin/sq dep list <issue-id> --json

# comments
./bin/sq comments add <issue-id> "needs review" --json
./bin/sq comments <issue-id> --json

# todo convenience
./bin/sq todo add "Follow up" --json
./bin/sq todo --json
./bin/sq todo done <id> --reason "Completed" --json
```

## 6) Reports and triage views

```bash
./bin/sq blocked --json
./bin/sq stale --days 30 --json
./bin/sq orphans --json
./bin/sq query "status=open AND priority<=2" --json
./bin/sq status --json
```

## 7) Environment

Override DB location when needed:

```bash
SQ_DB_PATH=/path/to/tasks.sqlite ./bin/sq list --json
```

## 8) Compatibility checks

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

## Next docs

- Full CLI reference: `docs/CLI_REFERENCE.md`
- Migration: `docs/MIGRATION_FROM_BEADS.md`
- Compatibility contract: `docs/COMPATIBILITY_CONTRACT.md`
