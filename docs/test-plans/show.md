# Test Plan: `sq show`

## Scope

Validate `sq show` for:
- successful lookup of an existing issue
- JSON output structure and field preservation
- human-readable output mode
- behavior across multiple issue types and states
- handling of updated metadata after lifecycle transitions
- error handling for missing ids, nonexistent ids, unknown flags, and malformed compatibility-flag usage
- compatibility/global flag acceptance (`--actor`, `--db`, `--dolt-auto-commit`, `--json`, `--quiet`, `--verbose`, `--profile`, `--readonly`, `--sandbox`)

This plan targets the top-level `sq show` command only.

## Initialization

1. Create a disposable workspace:
   ```bash
   TMP_DIR="$(mktemp -d -t sq-show-XXXXXX)"
   cd "$TMP_DIR"
   ```
2. Point to the CLI under test:
   ```bash
   SQ_BIN="/Users/auhanson/workspace/hnsn/squids-docs-bd-1pp/bin/sq"
   $SQ_BIN version
   ```
3. Initialize a fresh database:
   ```bash
   $SQ_BIN init --json
   ```
4. Create representative issues for later lookups:
   ```bash
   OPEN_JSON="$($SQ_BIN create "show open task" --type task --priority 2 --json)"
   OPEN_ID="$(printf '%s' "$OPEN_JSON" | jq -r '.id')"

   BUG_JSON="$($SQ_BIN create "show bug" --type bug --priority 1 --description "bug description" --json)"
   BUG_ID="$(printf '%s' "$BUG_JSON" | jq -r '.id')"

   CLOSED_JSON="$($SQ_BIN create "show closed task" --type task --json)"
   CLOSED_ID="$(printf '%s' "$CLOSED_JSON" | jq -r '.id')"
   $SQ_BIN close "$CLOSED_ID" --reason "setup" --json
   ```
5. Update one issue so `show` can be checked after mutation:
   ```bash
   $SQ_BIN update "$OPEN_ID" --assignee alice --add-label important --json
   ```

## Test Steps

### 1. Help and discoverability

1. Run command help:
   ```bash
   $SQ_BIN help show
   ```
2. Confirm help documents:
   - positional issue id
   - `--json`
   - global flags
3. Verify generic help wiring for the command family is correct and not a fallback to unrelated help text.

### 2. Basic JSON lookup of an existing issue

1. Run:
   ```bash
   $SQ_BIN show "$OPEN_ID" --json
   ```
2. Verify the response is a single JSON object.
3. Confirm expected fields are present:
   - `id`
   - `title`
   - `status`
   - `issue_type`
   - `priority`
   - `created_at`
   - `updated_at`
4. Confirm field values match the seeded issue.

### 3. Human-readable output mode

1. Run:
   ```bash
   $SQ_BIN show "$OPEN_ID"
   ```
2. Confirm output is human-readable rather than JSON.
3. Confirm it includes at minimum:
   - the issue id
   - the issue type
   - priority
   - title
4. Confirm the command exits successfully.

### 4. Lookup after updates/mutations

1. Re-run:
   ```bash
   $SQ_BIN show "$OPEN_ID" --json
   ```
2. Confirm updated fields are visible, such as:
   - assignee `alice`
   - label `important`
3. Confirm `updated_at` changed from its original create timestamp if applicable.

### 5. Lookup across multiple issue types/states

1. Run:
   ```bash
   $SQ_BIN show "$BUG_ID" --json
   ```
2. Confirm bug-specific seeded values are preserved, including description and priority.
3. Run:
   ```bash
   $SQ_BIN show "$CLOSED_ID" --json
   ```
4. Confirm closed state is reflected correctly and any close reason metadata is visible if surfaced.

### 6. Compatibility/global flags

1. Verify accepted compatibility/global flags do not break reads:
   ```bash
   $SQ_BIN show "$OPEN_ID" --json --quiet
   $SQ_BIN show "$OPEN_ID" --json --verbose
   $SQ_BIN show "$OPEN_ID" --json --sandbox
   $SQ_BIN show "$OPEN_ID" --json --readonly
   $SQ_BIN show "$OPEN_ID" --json --profile
   ```
2. Verify `--actor` is accepted:
   ```bash
   $SQ_BIN show "$OPEN_ID" --actor tester --json
   ```
3. If testing `--db`, point it at the initialized sqlite database path and confirm the same issue is returned.
4. Verify `--dolt-auto-commit` is accepted as a compatibility flag and does not affect read behavior.

### 7. Validation and error handling

Verify each case exits non-zero and reports a useful error:

1. Missing id:
   ```bash
   $SQ_BIN show
   ```
2. Nonexistent id:
   ```bash
   $SQ_BIN show bd-missing --json
   ```
3. Unknown flag:
   ```bash
   $SQ_BIN show "$OPEN_ID" --wat
   ```
4. Missing value for compatibility flags:
   ```bash
   $SQ_BIN show "$OPEN_ID" --actor
   $SQ_BIN show "$OPEN_ID" --db
   $SQ_BIN show "$OPEN_ID" --dolt-auto-commit
   ```
5. Extra unexpected positional arguments, if accepted by the parser, should be checked to confirm whether they are rejected or ignored consistently.

### 8. Cross-check against list output

1. Run:
   ```bash
   $SQ_BIN list --json
   ```
2. Confirm the issues returned by `show` match the corresponding records visible in list output.
3. Spot-check a closed issue and an open issue to ensure consistency between commands.

## Cleanup

1. Leave the temporary workspace:
   ```bash
   cd /
   ```
2. Remove the disposable directory and sqlite database:
   ```bash
   rm -rf "$TMP_DIR"
   ```
3. Confirm no repository worktree artifacts remain.

## Expected Outcome

- `sq show` returns a single issue reliably for an existing id.
- JSON mode returns a structured object with the persisted issue fields.
- Human mode prints a concise readable summary.
- Updated and closed issue state is reflected accurately.
- Invalid invocations fail clearly without mutating data.
- Compatibility/global flags are accepted or rejected consistently with explicit errors for malformed usage.
