# bd ↔ sq Deep Parity Matrix (Command/Flag/Subcommand Gaps)

_Generated: 2026-03-11 06:03_

## Top-level parity summary
- bd commands: **93**
- sq commands: **30**
- Shared commands: **30**
- Missing in sq: **63**

### Missing top-level commands in sq

`admin`, `agent`, `audit`, `backup`, `branch`, `compact`, `completion`, `config`, `cook`, `create-form`, `diff`, `doctor`, `dolt`, `duplicates`, `edit`, `epic`, `export`, `federation`, `find-duplicates`, `flatten`, `forget`, `formula`, `gate`, `gc`, `gitlab`, `graph`, `help`, `history`, `hooks`, `human`, `info`, `jira`, `kv`, `linear`, `lint`, `mail`, `memories`, `merge-slot`, `migrate`, `mol`, `move`, `onboard`, `preflight`, `prime`, `promote`, `purge`, `quickstart`, `recall`, `refile`, `remember`, `repo`, `restore`, `set-state`, `setup`, `ship`, `slot`, `sql`, `state`, `swarm`, `upgrade`, `vc`, `where`, `worktree`

## Shared commands: missing-flag count

| Command | Missing flags in sq |
|---|---:|
| `blocked` | 13 |
| `children` | 15 |
| `close` | 20 |
| `comments` | 14 |
| `count` | 45 |
| `create` | 60 |
| `defer` | 13 |
| `delete` | 17 |
| `dep` | 14 |
| `duplicate` | 13 |
| `init` | 29 |
| `label` | 12 |
| `list` | 75 |
| `orphans` | 15 |
| `q` | 19 |
| `query` | 21 |
| `ready` | 39 |
| `rename` | 13 |
| `rename-prefix` | 14 |
| `reopen` | 15 |
| `search` | 44 |
| `show` | 23 |
| `stale` | 18 |
| `status` | 16 |
| `supersede` | 13 |
| `todo` | 18 |
| `types` | 12 |
| `undefer` | 12 |
| `update` | 47 |
| `version` | 13 |

## Guardrails
- Implement full parity for in-scope commands, including shell parity tests.
- Skip dolt/server internals in sq; mark those as out-of-scope in bead notes and closure rationale.
- Validate behavior using both `bd <command> --help` and bd source before implementation.