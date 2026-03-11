# sq import-beads — Design

Bead: bd-965q
Status: proposed design

## Goal

Add an `sq import-beads` command that migrates data from an existing beads database into squids with deterministic mapping and idempotent behavior.

## CLI UX

```bash
sq import-beads [--source <path>] [--dry-run] [--json] [--no-comments] [--no-events]
```

### Flags

- `--source <path>`: explicit beads DB path (file or `.beads` directory).
- `--dry-run`: perform validation + planning only; no writes.
- `--json`: machine-readable summary report.
- `--no-comments`: skip comments import.
- `--no-events`: skip history/event-like rows if available.

### Exit codes

- `0`: import success (or dry-run success)
- `2`: source discovery/compatibility failure
- `3`: data mapping/validation failure
- `4`: write failure / transaction rollback

## Source Discovery Strategy

If `--source` is omitted:

1. Check `$BEADS_DIR` and `$BEADS_DATABASE`.
2. Check cwd for `.beads/`.
3. Check parent dirs upward for `.beads/` (stop at filesystem root).
4. Resolve candidate DB path from `.beads/config`/known filename conventions.

If multiple candidates are found, fail with actionable message and require `--source`.

## Compatibility Checks

Before importing:

1. Confirm source DB is readable.
2. Confirm required beads tables exist (issues/tasks, labels, deps, comments, metadata).
3. Validate critical columns used by mapping.
4. Detect unsupported schema variants and report exact missing/extra columns.

## Mapping Rules

### Core issue/task fields

- `id` -> `id` (preserve exactly)
- `title` -> `title`
- `description` -> `description`
- `status` -> `status`
- `priority` -> `priority`
- `type` -> `type`
- `created_at`/`updated_at` -> same semantics
- `assignee`/`owner` -> canonical squids assignee/owner fields

### Labels

- Import all labels as-is (case-preserving).
- Deduplicate per issue.

### Dependencies

- Preserve directional dependency edges.
- Skip invalid/self edges with warning.

### Comments

- Preserve body + author + timestamp when available.
- Keep stable ordering by created timestamp then source PK.

### Metadata

- Map scalar values directly.
- For complex values, store JSON-encoded string if needed.

## Idempotency

Default behavior should be upsert-like and idempotent:

- Re-running import on same source should not create duplicates.
- For existing IDs:
  - update mutable fields when source differs
  - preserve squids-only fields unless explicitly source-owned

Report counts:

- created
- updated
- unchanged
- skipped
- failed

## Transaction + Failure Modes

- Run full import in a single transaction by default.
- On failure, rollback all writes.
- Emit structured error with:
  - stage (`discover|validate|map|write`)
  - source object (table/id)
  - reason

## Performance

- Batch reads from source tables.
- Batch writes where safe.
- Stream progress for large imports (optional verbose mode).

## Observability / Reporting

Final report includes:

- source path
- schema version/signature detected
- table-level counts read/imported/skipped
- warnings list (truncated with count)
- elapsed time

## Test Plan

1. **Discovery tests**
   - explicit `--source` file/dir
   - env-based discovery
   - cwd/parent discovery
   - multi-candidate ambiguity

2. **Compatibility tests**
   - missing required table
   - missing required column
   - unsupported schema signature

3. **Mapping tests**
   - issue field parity
   - labels/deps/comments/metadata mapping
   - malformed rows are skipped + warned

4. **Idempotency tests**
   - import twice yields stable counts
   - changed source row updates existing target row

5. **Transaction tests**
   - injected write failure triggers rollback

6. **Dry-run tests**
   - no writes, accurate planned counts

## Implementation Outline

1. Add `import-beads` command route in CLI router.
2. Add source discovery module.
3. Add schema compatibility checker.
4. Add mapping layer (source rows -> squids model).
5. Add importer with transaction + reporting.
6. Add test fixtures for beads-like source DB variants.

## Non-Goals

- Supporting Dolt server lifecycle behavior.
- Perfect historical event replay parity; initial scope is core task data portability.
