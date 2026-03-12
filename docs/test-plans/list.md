# Test Plan: `sq list`

## Goal
Verify that `sq list` enumerates tasks from the active database correctly, returns stable JSON output, accepts the documented compatibility flags it currently supports, and fails cleanly for unsupported flags or malformed invocations.

## Initialization Steps
1. Open a disposable shell in the repository root.
2. Build the CLI binary:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Create an isolated database location for this test run:
   ```bash
   TMP_ROOT="$(mktemp -d -t sq-list-test-XXXXXX)"
   export SQ_DB_PATH="$TMP_ROOT/tasks.sqlite"
   ```
4. Initialize sq storage:
   ```bash
   ./bin/sq init --json
   ```
5. Seed several tasks with varied titles, priorities, and states:
   ```bash
   A=$(./bin/sq create "List smoke A" --type task --priority 1 --json | jq -r .id)
   B=$(./bin/sq create "List smoke B" --type bug --priority 2 --json | jq -r .id)
   C=$(./bin/sq create "List smoke C" --type feature --priority 3 --json | jq -r .id)
   ./bin/sq update "$B" --status in_progress --json
   ./bin/sq close "$C" --reason "Done" --json
   ```
6. Confirm the seed data exists with `sq show` for at least one created id.

## Testing Steps

### 1. Basic list behavior
1. Run:
   ```bash
   ./bin/sq list --json
   ```
2. Confirm the command exits successfully.
3. Confirm stdout is valid JSON.
4. Confirm the output is a JSON array.
5. Confirm all created tasks appear in the list, regardless of state.

### 2. Empty database behavior
1. In a second clean temporary database, run:
   ```bash
   export SQ_DB_PATH="$TMP_ROOT/empty.sqlite"
   ./bin/sq init --json
   ./bin/sq list --json
   ```
2. Confirm the result is an empty JSON array.
3. Confirm the command still exits successfully.

### 3. Compatibility flags currently accepted
1. Restore `SQ_DB_PATH` to the populated test database.
2. Run:
   ```bash
   ./bin/sq list --json --flat --no-pager
   ```
3. Confirm the command exits successfully.
4. Confirm the returned array still contains the seeded tasks.
5. Confirm accepted compatibility flags do not alter the JSON structure unexpectedly.

### 4. Ordering observation
1. Run:
   ```bash
   ./bin/sq list --json > "$TMP_ROOT/list.json"
   ```
2. Record the current order of returned tasks.
3. Confirm the ordering is consistent with current implementation expectations (currently creation-order based if observed).
4. Note this observed ordering explicitly if relied upon by tests or operators.

### 5. Integration with downstream commands
1. Capture an id from `sq list --json`.
2. Run:
   ```bash
   ./bin/sq show <id> --json
   ```
3. Confirm ids surfaced by `sq list` are immediately usable with `show`, `update`, and `close`.
4. This validates that list output is not only syntactically correct but operationally useful.

### 6. State coverage
1. Confirm the list includes:
   - open tasks
   - in-progress tasks
   - closed tasks
2. Verify state transitions remain visible:
   ```bash
   ./bin/sq update "$A" --status in_progress --json
   ./bin/sq list --json
   ./bin/sq close "$A" --reason "Done" --json
   ./bin/sq list --json
   ```
3. Confirm the list output reflects these changes accurately.

### 7. Unsupported flag behavior
1. Verify unsupported flags fail cleanly:
   ```bash
   ./bin/sq list --help
   ./bin/sq list --wat
   ./bin/sq list --status open
   ```
2. Confirm each invocation exits non-zero.
3. Confirm stderr reports `unknown flag` or equivalent usage-style failure.
4. Record that `--help` is not currently implemented for `sq list` if that remains true.

### 8. Read-only / no side-effect expectations
1. Capture list output before and after repeated read-only calls:
   ```bash
   ./bin/sq list --json > "$TMP_ROOT/before.json"
   ./bin/sq list --json
   ./bin/sq list --json --flat --no-pager
   ./bin/sq list --json > "$TMP_ROOT/after.json"
   ```
2. Confirm repeated list operations do not mutate task state.
3. Confirm any differences are only due to intentional writes from setup or explicit test steps.

### 9. Environment-based database selection
1. Point `SQ_DB_PATH` at a different initialized database and run:
   ```bash
   export SQ_DB_PATH="$TMP_ROOT/alt.sqlite"
   ./bin/sq init --json
   ./bin/sq create "Alt DB task" --json
   ./bin/sq list --json
   ```
2. Confirm the returned tasks are from the active database selected by the environment variable.
3. Confirm switching `SQ_DB_PATH` changes the list scope accordingly.

## Cleanup Steps
1. Remove the temporary databases and output files:
   ```bash
   rm -rf "$TMP_ROOT"
   ```
2. Unset the database override:
   ```bash
   unset SQ_DB_PATH
   ```
3. Return to the repository root.

## Expected Result
- `sq list --json` returns a valid JSON array of tasks from the active database.
- empty and populated database scenarios both behave correctly.
- accepted compatibility flags (`--flat`, `--no-pager`) are tolerated.
- unsupported flags fail cleanly.
- output ids are usable with downstream commands.
- `sq list` remains read-only and does not mutate task state.
