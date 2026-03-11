# bd ↔ sq Deep Parity Matrix (Command/Flag/Subcommand Gaps)

_Generated: 2026-03-11 06:26_

## Top-level parity summary
- bd commands: **93**
- sq commands: **35**
- Shared commands: **35**
- Missing in sq: **58**

### Missing top-level commands in sq

`admin`, `agent`, `audit`, `backup`, `branch`, `compact`, `completion`, `config`, `cook`, `create-form`, `diff`, `doctor`, `dolt`, `duplicates`, `edit`, `epic`, `export`, `federation`, `find-duplicates`, `flatten`, `forget`, `formula`, `gate`, `gc`, `gitlab`, `graph`, `history`, `hooks`, `jira`, `kv`, `linear`, `lint`, `mail`, `memories`, `merge-slot`, `migrate`, `mol`, `move`, `onboard`, `preflight`, `prime`, `promote`, `purge`, `recall`, `refile`, `remember`, `repo`, `restore`, `set-state`, `setup`, `ship`, `slot`, `sql`, `state`, `swarm`, `upgrade`, `vc`, `worktree`

## Shared commands: flag/subcommand gaps

| Command | Missing flags in sq (count) | Missing subcommands in sq |
|---|---:|---|
| `blocked` | 12 | — |
| `children` | 15 | — |
| `close` | 19 | — |
| `comments` | 15 | `add` |
| `count` | 43 | — |
| `create` | 55 | — |
| `defer` | 13 | — |
| `delete` | 16 | — |
| `dep` | 15 | `cycles`, `relate`, `tree`, `unrelate` |
| `duplicate` | 12 | — |
| `help` | 0 | — |
| `human` | 0 | — |
| `info` | 0 | — |
| `init` | 29 | — |
| `label` | 13 | `propagate` |
| `list` | 73 | — |
| `orphans` | 15 | — |
| `q` | 16 | — |
| `query` | 14 | `assignee`, `closed`, `created`, `description`, `ephemeral`, `id` …(+12) |
| `quickstart` | 0 | — |
| `ready` | 39 | — |
| `rename` | 12 | — |
| `rename-prefix` | 14 | — |
| `reopen` | 15 | — |
| `search` | 36 | — |
| `show` | 24 | — |
| `stale` | 16 | — |
| `status` | 16 | — |
| `supersede` | 12 | — |
| `todo` | 16 | — |
| `types` | 12 | — |
| `undefer` | 12 | — |
| `update` | 42 | — |
| `version` | 0 | — |
| `where` | 0 | — |

## Detailed gaps (shared commands)

### `blocked`
- Missing flags in sq (12): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `children`
- Missing flags in sq (15): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--json`, `--parent`, `--pretty`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `close`
- Missing flags in sq (19): `--actor`, `--continue`, `--db`, `--dolt-auto-commit`, `--force`, `--help`, `--no-auto`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--session`, `--suggest-next`, `--verbose`, `-f`, `-h`, `-q`, `-r`, `-v`
- Missing subcommands in sq: none

### `comments`
- Missing flags in sq (15): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--json`, `--local-time`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-f`, `-h`, `-q`, `-v`
- Missing subcommands in sq: `add`

### `count`
- Missing flags in sq (43): `--actor`, `--assignee`, `--by-assignee`, `--by-label`, `--by-priority`, `--by-status`, `--by-type`, `--closed-after`, `--closed-before`, `--created-after`, `--created-before`, `--db`, `--desc-contains`, `--dolt-auto-commit`, `--empty-description`, `--help`, `--id`, `--json`, `--label`, `--label-any`, `--no-assignee`, `--no-labels`, `--notes-contains`, `--priority`, `--priority-max`, `--priority-min`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--title`, `--title-contains`, `--type`, `--updated-after`, `--updated-before`, `--verbose`, `-a`, `-h`, `-l`, `-p`, `-q`, `-t`, `-v`
- Missing subcommands in sq: none

### `create`
- Missing flags in sq (55): `--acceptance`, `--actor`, `--agent-rig`, `--append-notes`, `--assignee`, `--body-file`, `--db`, `--defer`, `--design`, `--dolt-auto-commit`, `--dry-run`, `--due`, `--ephemeral`, `--estimate`, `--event-actor`, `--event-category`, `--event-payload`, `--event-target`, `--external-ref`, `--file`, `--force`, `--help`, `--id`, `--labels`, `--metadata`, `--mol-type`, `--no-inherit-labels`, `--notes`, `--parent`, `--prefix`, `--profile`, `--quiet`, `--readonly`, `--repo`, `--rig`, `--sandbox`, `--silent`, `--spec-id`, `--stdin`, `--title`, `--validate`, `--verbose`, `--waits-for`, `--waits-for-gate`, `--wisp-type`, `-a`, `-d`, `-e`, `-f`, `-h`, `-l`, `-p`, `-q`, `-t`, `-v`
- Missing subcommands in sq: none

### `defer`
- Missing flags in sq (13): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--until`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `delete`
- Missing flags in sq (16): `--actor`, `--cascade`, `--db`, `--dolt-auto-commit`, `--dry-run`, `--from-file`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-f`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `dep`
- Missing flags in sq (15): `--actor`, `--blocks`, `--db`, `--dolt-auto-commit`, `--help`, `--json`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-b`, `-h`, `-q`, `-v`
- Missing subcommands in sq: `cycles`, `relate`, `tree`, `unrelate`

### `duplicate`
- Missing flags in sq (12): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `help`
- Missing flags in sq (0): none
- Missing subcommands in sq: none

### `human`
- Missing flags in sq (0): none
- Missing subcommands in sq: none

### `info`
- Missing flags in sq (0): none
- Missing subcommands in sq: none

### `init`
- Missing flags in sq (29): `--actor`, `--agents-template`, `--backend`, `--contributor`, `--database`, `--db`, `--dolt-auto-commit`, `--force`, `--from-jsonl`, `--help`, `--json`, `--prefix`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--server`, `--server-host`, `--server-port`, `--server-user`, `--setup-exclude`, `--skip-hooks`, `--stealth`, `--team`, `--verbose`, `-h`, `-p`, `-q`, `-v`
- Missing subcommands in sq: none

### `label`
- Missing flags in sq (13): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--json`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: `propagate`

### `list`
- Missing flags in sq (73): `--actor`, `--all`, `--assignee`, `--closed-after`, `--closed-before`, `--created-after`, `--created-before`, `--db`, `--defer-after`, `--defer-before`, `--deferred`, `--desc-contains`, `--dolt-auto-commit`, `--due-after`, `--due-before`, `--empty-description`, `--format`, `--has-metadata-key`, `--help`, `--id`, `--include-gates`, `--include-infra`, `--include-templates`, `--label`, `--label-any`, `--label-pattern`, `--label-regex`, `--limit`, `--long`, `--metadata-field`, `--mol-type`, `--no-assignee`, `--no-labels`, `--no-parent`, `--no-pinned`, `--notes-contains`, `--overdue`, `--parent`, `--pinned`, `--pretty`, `--priority`, `--priority-max`, `--priority-min`, `--profile`, `--quiet`, `--readonly`, `--ready`, `--reverse`, `--rig`, `--sandbox`, `--sort`, `--spec`, `--status`, `--title`, `--title-contains`, `--tree`, `--type`, `--updated-after`, `--updated-before`, `--verbose`, `--watch`, `--wisp-type`, `-a`, `-h`, `-l`, `-n`, `-p`, `-q`, `-r`, `-s`, `-t`, `-v`, `-w`
- Missing subcommands in sq: none

### `orphans`
- Missing flags in sq (15): `--actor`, `--db`, `--details`, `--dolt-auto-commit`, `--fix`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-f`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `q`
- Missing flags in sq (16): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--labels`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-l`, `-p`, `-q`, `-t`, `-v`
- Missing subcommands in sq: none

### `quickstart`
- Missing flags in sq (0): none
- Missing subcommands in sq: none

### `query`
- Missing flags in sq (14): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-n`, `-q`, `-r`, `-v`
- Missing subcommands in sq: `assignee`, `closed`, `created`, `description`, `ephemeral`, `id`, `label`, `notes`, `owner`, `parent`, `pinned`, `priority`, `spec`, `status`, `template`, `title`, `type`, `updated`

### `ready`
- Missing flags in sq (39): `--actor`, `--assignee`, `--db`, `--dolt-auto-commit`, `--gated`, `--has-metadata-key`, `--help`, `--include-deferred`, `--include-ephemeral`, `--json`, `--label`, `--label-any`, `--limit`, `--metadata-field`, `--mol`, `--mol-type`, `--parent`, `--plain`, `--pretty`, `--priority`, `--profile`, `--quiet`, `--readonly`, `--rig`, `--sandbox`, `--sort`, `--type`, `--unassigned`, `--verbose`, `-a`, `-h`, `-l`, `-n`, `-p`, `-q`, `-s`, `-t`, `-u`, `-v`
- Missing subcommands in sq: none

### `rename`
- Missing flags in sq (12): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `rename-prefix`
- Missing flags in sq (14): `--actor`, `--db`, `--dolt-auto-commit`, `--dry-run`, `--help`, `--profile`, `--quiet`, `--readonly`, `--repair`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `reopen`
- Missing flags in sq (15): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--reason`, `--sandbox`, `--status`, `--verbose`, `-h`, `-q`, `-r`, `-v`
- Missing subcommands in sq: none

### `search`
- Missing flags in sq (36): `--actor`, `--assignee`, `--closed-after`, `--closed-before`, `--created-after`, `--created-before`, `--db`, `--desc-contains`, `--dolt-auto-commit`, `--empty-description`, `--has-metadata-key`, `--help`, `--label`, `--label-any`, `--metadata-field`, `--no-assignee`, `--no-labels`, `--notes-contains`, `--priority-max`, `--priority-min`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--type`, `--updated-after`, `--updated-before`, `--verbose`, `-a`, `-h`, `-l`, `-q`, `-r`, `-s`, `-t`, `-v`
- Missing subcommands in sq: none

### `show`
- Missing flags in sq (24): `--actor`, `--as-of`, `--children`, `--current`, `--db`, `--dolt-auto-commit`, `--help`, `--id`, `--json`, `--local-time`, `--long`, `--profile`, `--quiet`, `--readonly`, `--refs`, `--sandbox`, `--short`, `--thread`, `--verbose`, `--watch`, `-h`, `-q`, `-v`, `-w`
- Missing subcommands in sq: none

### `stale`
- Missing flags in sq (16): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--limit`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--status`, `--verbose`, `-h`, `-n`, `-q`, `-s`, `-v`
- Missing subcommands in sq: none

### `status`
- Missing flags in sq (16): `--actor`, `--all`, `--assigned`, `--db`, `--dolt-auto-commit`, `--help`, `--json`, `--no-activity`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `supersede`
- Missing flags in sq (12): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `todo`
- Missing flags in sq (16): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--status`, `--type`, `--verbose`, `-h`, `-p`, `-q`, `-t`, `-v`
- Missing subcommands in sq: none

### `types`
- Missing flags in sq (12): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `undefer`
- Missing flags in sq (12): `--actor`, `--db`, `--dolt-auto-commit`, `--help`, `--profile`, `--quiet`, `--readonly`, `--sandbox`, `--verbose`, `-h`, `-q`, `-v`
- Missing subcommands in sq: none

### `update`
- Missing flags in sq (42): `--acceptance`, `--actor`, `--append-notes`, `--await-id`, `--body-file`, `--db`, `--defer`, `--description`, `--design`, `--dolt-auto-commit`, `--due`, `--ephemeral`, `--estimate`, `--external-ref`, `--help`, `--metadata`, `--notes`, `--parent`, `--persistent`, `--priority`, `--profile`, `--quiet`, `--readonly`, `--remove-label`, `--sandbox`, `--session`, `--set-labels`, `--spec-id`, `--stdin`, `--title`, `--type`, `--unset-metadata`, `--verbose`, `-a`, `-d`, `-e`, `-h`, `-p`, `-q`, `-s`, `-t`, `-v`
- Missing subcommands in sq: none

### `version`
- Missing flags in sq (0): none
- Missing subcommands in sq: none

### `where`
- Missing flags in sq (0): none
- Missing subcommands in sq: none

## Guardrails
- Do **not** add dolt/server lifecycle or storage-internal commands to sq.
- Before implementing parity behavior, verify semantics using both `bd <command> --help` and bd source.