# Test Plan: `sq comments`

## Scope

Validate the top-level `sq comments` command for:
- command help and discoverability
- listing comments for an existing issue in JSON mode
- listing comments for an existing issue in human-readable mode
- local-time formatting behavior via `--local-time`
- empty-comment-list behavior
- error handling for missing issue ids, unknown flags, missing compatibility-flag values, and nonexistent issue ids
- compatibility/global flag handling (`--actor`, `--db`, `--dolt-auto-commit`, `--json`, `--quiet`, `--verbose`, `--profile`, `--readonly`, `--sandbox`)

This plan targets `sq comments` itself, not the `sq comments add` subcommand.

## Initialization

1. Create a disposable workspace:
   ```bash
   TMP_DIR="$(mktemp -d -t sq-comments-XXXXXX)"
   cd "$TMP_DIR"
   ```
2. Point to the CLI under test:
   ```bash
   SQ_BIN="/Users/auhanson/workspace/hnsn/squids-docs-bd-46f/bin/sq"
   $SQ_BIN version
   ```
3. Initialize a fresh sqlite-backed sq database:
   ```bash
   $SQ_BIN init --json
   ```
4. Create two issues:
   - one issue that will receive comments
   - one issue left without comments to test empty-list behavior
   ```bash
   ISSUE_JSON="$($SQ_BIN create "comment target" --json)"
   ISSUE_ID="$(printf '%s' "$ISSUE_JSON" | jq -r '.id')"

   EMPTY_JSON="$($SQ_BIN create "no comments yet" --json)"
   EMPTY_ID="$(printf '%s' "$EMPTY_JSON" | jq -r '.id')"
   ```
5. Seed comments on the target issue using the CLI:
   ```bash
   $SQ_BIN comments add "$ISSUE_ID" "first comment" --json
   $SQ_BIN comments add "$ISSUE_ID" "second comment" --author alice --json
   ```

## Test Steps

### 1. Help and command surface

1. Run command help:
   ```bash
   $SQ_BIN comments --help
   ```
2. Confirm help includes:
   - command purpose text
   - usage for `sq comments [issue-id] [flags]`
   - command form `sq comments [command]`
   - `add` in available commands
   - `--local-time`
   - global flags
3. Verify umbrella help wiring:
   ```bash
   $SQ_BIN help comments
   ```
4. Confirm the output is command-specific rather than generic fallback help.

### 2. JSON listing for an issue with comments

1. List comments in JSON mode:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --json
   ```
2. Verify the payload is a JSON array.
3. Confirm each element includes the expected fields:
   - `id`
   - `issue_id`
   - `body`
   - `created_at`
   - optional `author`
4. Confirm results are ordered oldest-first by insertion/id.
5. Confirm `issue_id` matches `$ISSUE_ID` for every returned comment.

### 3. Human-readable listing for an issue with comments

1. List comments without `--json`:
   ```bash
   $SQ_BIN comments "$ISSUE_ID"
   ```
2. Confirm output includes:
   - numeric comment ids
   - timestamps
   - author names where present
   - fallback author text for comments without explicit authors
   - indented multi-line-friendly comment body display
3. Confirm both seeded comments are visible in the output.

### 4. `--local-time` behavior

1. Run:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --local-time
   ```
2. Confirm the command succeeds and still lists all comments.
3. Compare the displayed timestamp format with plain output and verify the command applies local-time formatting rather than failing or switching to JSON unexpectedly.

### 5. Empty list behavior

1. Run on the issue with no comments:
   ```bash
   $SQ_BIN comments "$EMPTY_ID"
   ```
2. Confirm the command exits successfully.
3. Confirm output clearly indicates there are no comments (for example, `No comments found`).
4. Also verify JSON mode on the empty issue:
   ```bash
   $SQ_BIN comments "$EMPTY_ID" --json
   ```
5. Confirm the result is a valid empty JSON array.

### 6. Compatibility/global flags

1. Verify accepted no-op/compat flags do not break listing behavior:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --json --quiet
   $SQ_BIN comments "$ISSUE_ID" --json --verbose
   $SQ_BIN comments "$ISSUE_ID" --json --sandbox
   $SQ_BIN comments "$ISSUE_ID" --json --readonly
   $SQ_BIN comments "$ISSUE_ID" --json --profile
   ```
2. Verify `--actor` is accepted:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --actor tester --json
   ```
3. If testing `--db`, point it at the initialized sqlite database and confirm the same comment list is returned.
4. Verify `--dolt-auto-commit` is accepted as a compatibility flag and does not change sqlite read behavior.

### 7. Validation and error handling

Run each case and verify non-zero exit status with a clear error:

1. Missing issue id:
   ```bash
   $SQ_BIN comments
   ```
2. Unknown flag:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --wat
   ```
3. Missing value for `--actor`:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --actor
   ```
4. Missing value for `--db`:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --db
   ```
5. Missing value for `--dolt-auto-commit`:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --dolt-auto-commit
   ```
6. Nonexistent issue id:
   ```bash
   $SQ_BIN comments bd-missing --json
   ```
7. Help typo/incorrect subcommand path handling, if desired:
   ```bash
   $SQ_BIN comments nope extra
   ```
   Confirm it fails rather than silently ignoring extra positional args.

### 8. Persistence cross-check

1. Re-run:
   ```bash
   $SQ_BIN comments "$ISSUE_ID" --json
   ```
2. Confirm the number of returned comments is unchanged from the seeded state.
3. Confirm bodies and authors still match the inserted values.

## Cleanup

1. Leave the temporary directory:
   ```bash
   cd /
   ```
2. Remove the temporary workspace and sqlite database:
   ```bash
   rm -rf "$TMP_DIR"
   ```
3. Confirm no artifacts were created in the repository worktree.

## Expected Outcome

- `sq comments` help is discoverable and documents the correct command surface.
- JSON mode returns a stable array of stored comments for an existing issue.
- Human mode returns readable, non-JSON output.
- `--local-time` affects human timestamp formatting without breaking output shape.
- Empty issues are handled cleanly in both human and JSON modes.
- Invalid invocations fail clearly and do not mutate state.
- Compatibility/global flags are accepted or rejected consistently with explicit errors for missing values.
