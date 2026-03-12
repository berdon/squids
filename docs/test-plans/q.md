# Test Plan: `sq q`

## Goal
Verify that `sq q` performs fast task creation correctly, returns the expected compact output format, honors its supported flags, creates usable issues in the active database, and fails cleanly on invalid invocations.

## Initialization Steps
1. Open a disposable shell in the repository root.
2. Build the CLI binary:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Create an isolated test database path:
   ```bash
   TMP_ROOT="$(mktemp -d -t sq-q-test-XXXXXX)"
   export SQ_DB_PATH="$TMP_ROOT/tasks.sqlite"
   ```
4. Initialize sq storage:
   ```bash
   ./bin/sq init --json
   ```
5. Confirm the database is empty or in a known clean state:
   ```bash
   ./bin/sq list --json
   ```

## Testing Steps

### 1. Basic quick-capture behavior
1. Run:
   ```bash
   ./bin/sq q "Quick capture task"
   ```
2. Confirm the command exits successfully.
3. Confirm stdout is a single issue id in `bd-...` format.
4. Save that id and verify the task exists:
   ```bash
   ./bin/sq show <id> --json
   ```
5. Confirm the created task has:
   - title `Quick capture task`
   - default issue type `task`
   - default priority expected by the implementation
   - status `open`

### 2. JSON output mode
1. Run:
   ```bash
   ./bin/sq q "Quick capture json" --json
   ```
2. Confirm the command exits successfully.
3. Confirm stdout is valid JSON.
4. Confirm the JSON contains an `id` field.
5. Confirm the returned id can be shown successfully with `sq show <id> --json`.

### 3. Supported creation flags
1. Run:
   ```bash
   ./bin/sq q "Typed quick task" --type bug --priority 1 --description "Captured quickly" --json
   ```
2. Confirm the command exits successfully.
3. Confirm the created issue reflects:
   - issue type `bug`
   - priority `1`
   - description `Captured quickly`
4. Verify with:
   ```bash
   ./bin/sq show <id> --json
   ```

### 4. Multiple quick-capture operations
1. Run several `sq q` commands in sequence:
   ```bash
   ./bin/sq q "First quick task"
   ./bin/sq q "Second quick task"
   ./bin/sq q "Third quick task" --json
   ```
2. Confirm each invocation returns a distinct issue id.
3. Confirm all created issues appear in:
   ```bash
   ./bin/sq list --json
   ```

### 5. Integration with follow-up workflow
1. Create a task with `sq q` and then update/close it:
   ```bash
   ID=$(./bin/sq q "Lifecycle quick task")
   ./bin/sq update "$ID" --status in_progress --json
   ./bin/sq close "$ID" --reason "Done" --json
   ```
2. Confirm the task transitions correctly through the normal workflow.
3. Confirm `sq q` creates issues compatible with downstream commands.

### 6. Creator / actor context
1. Optionally set a creator context:
   ```bash
   export BD_ACTOR=tester
   ```
2. Run:
   ```bash
   ./bin/sq q "Actor quick task" --json
   ```
3. Confirm the issue is created successfully.
4. If creator metadata is surfaced by current commands, verify the task reflects the expected actor-derived creator information.

### 7. Failure-path checks
1. Verify missing title fails:
   ```bash
   ./bin/sq q
   ```
2. Confirm the command exits non-zero and prints a usage-style error.
3. Verify invalid priority fails:
   ```bash
   ./bin/sq q "Bad priority" --priority nope
   ```
4. Confirm the command exits non-zero and reports invalid priority.
5. Verify unknown flags fail:
   ```bash
   ./bin/sq q "Bad flag" --wat
   ```
6. Confirm the command exits non-zero and reports the unknown flag.

### 8. Help / flag-surface observation
1. Run:
   ```bash
   ./bin/sq q --help
   ```
2. Record the current observed behavior exactly.
3. If the command currently treats `--help` as positional input instead of help, document that as an implementation observation for future parity work.
4. If the command later gains proper help handling, confirm it prints usage and exits successfully.

### 9. Output shape checks
1. Confirm human output mode emits only the id and not a full JSON object.
2. Confirm JSON mode emits a compact object containing the id.
3. Confirm no extraneous logging appears on stdout in either success path.

## Cleanup Steps
1. Remove the temporary database and workspace:
   ```bash
   rm -rf "$TMP_ROOT"
   ```
2. Unset temporary environment variables:
   ```bash
   unset SQ_DB_PATH
   unset BD_ACTOR
   ```
3. Return to the repository root.

## Expected Result
- `sq q` creates tasks quickly and returns an id suitable for scripting.
- `--json` returns valid JSON with the created id.
- supported creation flags are applied to the created task.
- created issues work with normal `show`, `update`, `close`, and `list` flows.
- invalid invocations fail cleanly.
- any current help-surface quirks are captured as observations for future follow-up.
