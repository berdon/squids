# squids

SQLite-backed, beads-compatible task CLI.

## Why squids exists

`sq` exists to preserve beads-style task workflows while removing Dolt/server operational complexity.

Goals:
- Keep familiar command semantics for day-to-day task operations.
- Use a local SQLite backend (single file, no daemon lifecycle).
- Support concurrent local access safely (WAL + busy timeout).
- Grow compatibility through test-first parity automation.

---

## Installation

### Option 1: Build from source (recommended today)

```bash
git clone https://github.com/berdon/squids.git
cd squids
make build
./bin/sq --help
```

### Option 2: Go install

```bash
go install github.com/berdon/squids/cmd/sq@latest
sq --help
```

> If `sq` is not found after `go install`, ensure your Go bin path is in `PATH`:
>
> ```bash
> export PATH="$PATH:$(go env GOPATH)/bin"
> ```

### Option 3: GitHub Releases (when using release tags)

On release tags, CI publishes platform binaries in GitHub Releases (`.github/workflows/release.yml`).

---

## Getting started (5 minutes)

1. **Initialize sq in your repo/workspace**

```bash
sq init --json
```

2. **Check ready work (open + unblocked)**

```bash
sq ready --json
```

`ready` returns the current set of actionable tasks: status `open` and not dependency-blocked.

3. **Create your first task**

```bash
sq create "Ship migration docs" --type task --priority 1 --description "Document sq rollout" --json
```

4. **List and inspect tasks**

```bash
sq list --json --flat --no-pager
sq show <id> --json
```

5. **Work the task**

```bash
sq update <id> --status in_progress --assignee guppy --json
sq close <id> --reason "Completed" --json
```

By default, data is stored at:

- `./.sq/tasks.sqlite`

Override location with:

```bash
SQ_DB_PATH=/path/to/tasks.sqlite sq list --json
```

---

## Tutorials

### Tutorial 1: Basic lifecycle

```bash
sq init --json
sq create "Fix login edge case" --type bug --priority 1 --json
sq update <id> --status in_progress --assignee alice --json
sq comments add <id> "Investigating production logs" --json
sq close <id> --reason "Patched and verified" --json
```

### Tutorial 2: Labels + dependencies + blocked view

```bash
# create tasks
sq create "Parent epic" --type epic --priority 1 --json
sq create "Child task" --type task --priority 2 --deps parent-child:<parent-id> --json
sq create "Blocking task" --type task --priority 1 --json

# add metadata and dependency relations
sq label add <child-id> area:backend --json
sq dep add <blocker-id> <child-id> --json

# inspect planning views
sq children <parent-id> --json
sq blocked --json
```

### Tutorial 3: Todo shortcuts

```bash
sq todo add "Follow up with QA" --priority 2 --json
sq todo --json
sq todo done <todo-id> --reason "Handled" --json
```

### Tutorial 4: Querying and reporting

```bash
sq query "status=open AND priority<=2" --json
sq search "login" --json -n 10
sq count --json
sq count --status open --json
sq status --json
sq types --json
```

### Tutorial 5: Duplicate/supersede flows

```bash
sq duplicate <duplicate-id> --of <canonical-id> --json
sq supersede <old-id> --with <new-id> --json
```

---

## Examples by use case

### Daily personal task tracking

```bash
sq create "Plan sprint" --type task --priority 2 --json
sq create "Refactor auth middleware" --type feature --priority 1 --json
sq ready --json
```

### Bug triage

```bash
sq create "Crash when token expires" --type bug --priority 0 --json
sq label add <bug-id> severity:critical --json
sq update <bug-id> --assignee oncall --status in_progress --json
```

### Content/decision tracking

```bash
sq create "ADR: auth provider strategy" --type decision --priority 2 --json
sq comments add <id> "Option B has lower operational risk" --json
```

---

## Supported commands (today)

Core lifecycle:
- `init`
- `ready`
- `create`
- `q`
- `show`
- `list`
- `update`
- `close`
- `reopen`
- `delete`
- `defer`
- `undefer`
- `rename`
- `rename-prefix`

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
- `stale`
- `orphans`
- `search`
- `count`
- `status`
- `types`

---

## Known gaps vs beads

Squids intentionally does **not** implement:
- `bd dolt ...` command family
- Dolt server lifecycle/start-stop operations
- Dolt-specific history/replication/storage controls

Squids also does not yet claim full parity for the broader beads surface area beyond the commands listed above.

---

## Parity and testing

### Run parity against sq

```bash
TARGET_BIN=./bin/sq ./scripts/parity/run-parity.sh
```

### Compare beads vs sq

```bash
./scripts/parity/compat-runner.sh
```

### Concurrency smoke test

```bash
./scripts/parity/concurrency-smoke.sh
```

---

## CI/CD (GitHub Actions)

- CI workflow: `.github/workflows/ci.yml`
  - Runs on pushes/PRs
  - Executes tests + build + parity + concurrency checks
  - Enforces coverage gate (>= 90%)
- Release workflow: `.github/workflows/release.yml`
  - Runs on tags like `v0.1.0`
  - Builds cross-platform binaries and publishes a GitHub Release with checksums

---

## Documentation

- Getting started: `docs/GETTING_STARTED.md`
- CLI reference: `docs/CLI_REFERENCE.md`
- Migration guide: `docs/MIGRATION_FROM_BEADS.md`
- Compatibility contract: `docs/COMPATIBILITY_CONTRACT.md`
- APE/cosmopolitan build notes: `docs/APE_BUILD.md`
- SQLite concurrency strategy: `docs/SQLITE_CONCURRENCY.md`
- Release notes: `docs/RELEASE_NOTES.md`
