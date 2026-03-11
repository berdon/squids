# squids CLI Reference (`sq`)

This document describes the current `sq` command surface and accepted flags.

## Global usage

```bash
sq <command> [args]
```

Behavior conventions:
- JSON output is the primary output format.
- Unknown flags return usage errors (exit code `2`) unless explicitly tolerated for compatibility.
- Runtime/storage failures return exit code `1`.

Environment:
- `SQ_DB_PATH=/path/to/tasks.sqlite` — override default DB path (`./.sq/tasks.sqlite`)
- `BD_ACTOR` / `USER` — used as creator/author defaults in create/comment flows.

---

## Core commands

### `sq init`
Initialize task storage and schema.

```bash
sq init --json
```

### `sq ready`
List ready tasks (open and unblocked).

```bash
sq ready --json
```

### `sq create <title>`
Create a task/issue.

Flags:
- `--type <task|bug|feature|chore|epic|decision>`
- `--priority <int>`
- `--description <text>`
- `--deps <id>` or `--deps <type:id>`
- `--json`

### `sq q <title>`
Quick create and print ID (or `{id}` with `--json`).

Flags:
- `--type <type>`
- `--priority <int>`
- `--description <text>`
- `--json`

### `sq show <id>`
Show one task.

### `sq list`
List tasks.

Accepted flags:
- `--json`
- `--flat`
- `--no-pager`

### `sq update <id>`
Update task fields.

Flags:
- `--status <open|in_progress|blocked|deferred|closed|resolved>`
- `--assignee <name>`
- `--add-label <label>` (repeatable)
- `--set-metadata <key=value>` (repeatable)
- `--claim`
- `--json`

### `sq close <id>`
Close task.

Flags:
- `--reason <text>`
- `--json`

### `sq reopen <id>`
Reopen task.

Flags:
- `--json`

### `sq delete <id>`
Delete task.

Flags:
- `--json`
- `--force` (accepted compatibility flag)

### `sq defer <id> [<id>...]`
Set status to `deferred` for one or more IDs.

Flags:
- `--json`

### `sq undefer <id> [<id>...]`
Set status to `open` for one or more IDs.

Flags:
- `--json`

### `sq rename <old-id> <new-id>`
Rename a task ID.

Flags:
- `--json`

### `sq rename-prefix <new-prefix>`
### `sq rename-prefix <old-prefix> <new-prefix>`
Batch-rename ID prefixes.

Flags:
- `--json`

---

## Relationship + metadata commands

### `sq label <subcommand>`
- `sq label add <id> <label> [--json]`
- `sq label remove <id> <label> [--json]`
- `sq label list <id> [--json]`
- `sq label list-all [--json]`

### `sq dep <subcommand>`
- `sq dep add <issue-id> <depends-on-id> [--json]`
- `sq dep remove <issue-id> <depends-on-id> [--json]`
- `sq dep rm <issue-id> <depends-on-id> [--json]` (alias)
- `sq dep list <issue-id> [--json]`

### `sq comments`
- `sq comments <issue-id> [--json]` (list)
- `sq comments add <issue-id> <text> [--json]`

### `sq todo`
- `sq todo [--json]` (list open task-type items)
- `sq todo add <title> [--priority N] [--description TEXT] [--json]`
- `sq todo done <id> [--reason TEXT] [--json]`

### `sq children <parent-id>`
List child tasks for parent.

Flags:
- `--json`

### `sq blocked`
List blocked tasks.

Flags:
- `--json`
- `--parent <id>` (accepted compatibility flag)

### `sq duplicate <id> --of <canonical-id>`
Marks issue as duplicate and closes it.

Flags:
- `--of <canonical-id>` (required)
- `--json`

### `sq supersede <id> --with <replacement-id>`
Marks issue as superseded and closes it.

Flags:
- `--with <replacement-id>` (required)
- `--json`

---

## Query and reporting commands

### `sq query <expression>`
Run query expression (currently supports key patterns used in parity tests, including priority comparisons).

Accepted compatibility flags (ignored for now):
- `--json`
- `-a`, `--all`
- `--sort ...`
- `--reverse`
- `--long`
- `--parse-only`
- `--limit ...`

### `sq stale`
List stale open tasks.

Flags:
- `--days <N>` / `-d <N>` (default `30`)
- `--json`

### `sq orphans`
List tasks with orphaned dependency references.

Flags:
- `--json`

### `sq search [query]`
Search tasks.

Flags:
- `--query <text>`
- `--limit <N>` / `-n <N>`
- `--json`

Accepted compatibility flags:
- `--status ...`
- `--sort ...`
- `--reverse`
- `--long`
- unknown flags are currently ignored in `search` for compatibility.

### `sq count`
Count tasks.

Flags:
- `--status <status>` / `-s <status>`
- `--json`

### `sq status`
### `sq stats`
Show status summary (alias: `stats`).

---

## Informational commands

### `sq types`
List supported core types.

Flags:
- `--json`

### `sq help`
### `sq --help`
Show top-level command usage.
