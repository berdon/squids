# Test Plan: `sq update`

## Goal
Verify that `sq update` correctly modifies existing tasks, supports its implemented field updates, handles claim semantics, preserves unchanged fields, and fails cleanly on invalid ids, flags, and malformed values.

## Initialization Steps
1. Open a disposable shell in the repository root.
2. Build the CLI binary:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Create an isolated database path for this test run:
   ```bash
   TMP_ROOT="$(mktemp -d -t sq-update-test-XXXXXX)"
   export SQ_DB_PATH="$TMP_ROOT/tasks.sqlite"
   ```
4. Initialize sq storage:
   ```bash
   ./bin/sq init --json
   ```
5. Seed at least two tasks:
   ```bash
   A=$(./bin/sq create "Update target" --type task --priority 2 --description "before" --json | jq -r .id)
   B=$(./bin/sq create "Second target" --type bug --priority 1 --json | jq -r .id)
   ```
6. Confirm both tasks exist with `sq show <id> --json`.

## Testing Steps

### 1. Basic status update
1. Run:
   ```bash
   ./bin/sq update "$A" --status in_progress --json
   ```
2. Confirm the command exits successfully.
3. Confirm the returned JSON shows status `in_progress`.
4. Verify persistence with:
   ```bash
   ./bin/sq show "$A" --json
   ```

### 2. Assignee update
1. Run:
   ```bash
   ./bin/sq update "$A" --assignee alice --json
   ```
2. Confirm the assignee is set to `alice`.
3. Verify unchanged fields remain intact (title, type, existing labels unless explicitly changed).

### 3. Claim behavior
1. Run:
   ```bash
   ./bin/sq update "$B" --claim --json
   ```
2. Confirm the command exits successfully.
3. Confirm the returned task status becomes `in_progress`.
4. Verify the claim transition is persisted with `sq show "$B" --json`.

### 4. Label updates
1. Run:
   ```bash
   ./bin/sq update "$A" --add-label backend --json
   ./bin/sq update "$A" --add-label urgent --json
   ```
2. Confirm the returned task includes the new labels.
3. Verify labels persist in subsequent `show` or `list` output.
4. Confirm re-adding an existing label does not break the command and does not create harmful duplicates if current behavior avoids them.

### 5. Metadata updates
1. Run:
   ```bash
   ./bin/sq update "$A" --set-metadata team=platform --json
   ./bin/sq update "$A" --set-metadata env=test --json
   ```
2. Confirm metadata keys are present in the returned JSON.
3. Verify metadata persists via `sq show "$A" --json`.
4. Confirm metadata updates do not erase unrelated existing metadata unless current implementation does so intentionally.

### 6. Combined update path
1. Run a multi-field update:
   ```bash
   ./bin/sq update "$A" --status open --assignee bob --add-label api --set-metadata owner=docs --json
   ```
2. Confirm all intended fields are updated in one invocation.
3. Confirm previously set labels/metadata still behave consistently with current implementation.

### 7. Multiple issue coverage
1. Update both seeded tasks with different values.
2. Confirm updates remain isolated to the specified target id.
3. Verify `sq list --json` reflects both modified tasks accurately.

### 8. Invalid id handling
1. Run:
   ```bash
   ./bin/sq update bd-missing --status open --json
   ```
2. Confirm the command exits non-zero.
3. Confirm stderr reports `issue not found` or equivalent runtime failure.

### 9. Missing id / malformed invocation
1. Run:
   ```bash
   ./bin/sq update
   ./bin/sq update --json
   ```
2. Confirm both invocations fail.
3. Record the current observed error shape exactly.
4. Confirm no unintended task is created or mutated.

### 10. Invalid values and unknown flags
1. Verify unsupported status values are rejected if current implementation validates them; otherwise record the observed behavior.
2. Verify unknown flags fail cleanly:
   ```bash
   ./bin/sq update "$A" --wat
   ```
3. Verify malformed metadata input is handled or document the observed current behavior:
   ```bash
   ./bin/sq update "$A" --set-metadata broken --json
   ```
4. Confirm failures are non-zero and usage/runtime errors are understandable.

### 11. Help-surface observation
1. Run:
   ```bash
   ./bin/sq update "$A" --help
   ./bin/sq help update
   ```
2. Record the current observed behavior for both commands.
3. If direct `--help` is not implemented for `sq update`, document that observation for future parity work.
4. Confirm `sq help update` still provides usable command guidance if available.

### 12. No unintended field loss
1. Before an update, capture the full task JSON:
   ```bash
   ./bin/sq show "$A" --json > "$TMP_ROOT/before.json"
   ```
2. Apply one focused update, e.g. only assignee.
3. Capture the task again:
   ```bash
   ./bin/sq show "$A" --json > "$TMP_ROOT/after.json"
   ```
4. Confirm only the intended fields changed and unrelated fields remain stable.

## Cleanup Steps
1. Remove temporary database and artifacts:
   ```bash
   rm -rf "$TMP_ROOT"
   ```
2. Unset the database override:
   ```bash
   unset SQ_DB_PATH
   ```
3. Return to the repository root.

## Expected Result
- `sq update` successfully modifies existing tasks.
- implemented updates for status, assignee, labels, metadata, and claim semantics behave consistently.
- updates persist and are visible through `show` and `list`.
- invalid ids and malformed invocations fail cleanly.
- any current help-surface quirks are captured explicitly as observations for future follow-up.
