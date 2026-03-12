# Test Plan: `sq ready`

## Goal
Verify that `sq ready` correctly reports actionable work, honors its supported filters and sort modes, handles accepted compatibility flags, and avoids mutating task state while reading from the sq database.

## Initialization Steps
1. Open a disposable shell in the repository root.
2. Build the CLI binary:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Create a clean test workspace and explicit database path:
   ```bash
   TMP_ROOT="$(mktemp -d -t sq-ready-test-XXXXXX)"
   export SQ_DB_PATH="$TMP_ROOT/tasks.sqlite"
   ```
4. Initialize sq storage:
   ```bash
   ./bin/sq init --json
   ```
5. Seed representative tasks for filtering and blocker checks:
   ```bash
   A=$(./bin/sq create "Ready task" --type task --priority 1 --json | jq -r .id)
   B=$(./bin/sq create "Blocked task" --type task --priority 2 --json | jq -r .id)
   C=$(./bin/sq create "Parent task" --type task --priority 2 --json | jq -r .id)
   D=$(./bin/sq create "Child task" --type task --priority 2 --deps parent-child:$C --json | jq -r .id)
   E=$(./bin/sq create "Assigned backend task" --type task --priority 2 --json | jq -r .id)
   ./bin/sq dep add $B $A --json
   ./bin/sq update $D --assignee bob --add-label backend --set-metadata team=platform --json
   ./bin/sq update $E --assignee alice --add-label api --json
   ```
6. Confirm the database has both blocked and unblocked tasks.

## Testing Steps

### 1. Basic ready output
1. Run:
   ```bash
   ./bin/sq ready --json
   ```
2. Confirm the command exits successfully.
3. Confirm the output is a JSON array.
4. Confirm open, unblocked tasks are present.
5. Confirm blocked tasks are excluded.

### 2. Help surface
1. Run:
   ```bash
   ./bin/sq ready --help
   ```
2. Confirm output includes:
   - `Show ready work (open issues with no active blockers).`
   - `sq ready [flags]`
   - filters such as `--assignee`, `--label`, `--label-any`, `--limit`, `--priority`, `--sort`, `--type`, `--unassigned`
   - `Global Flags:`
3. Confirm the command exits successfully.

### 3. Assignee and unassigned filters
1. Run:
   ```bash
   ./bin/sq ready --assignee bob --json
   ```
2. Confirm only tasks assigned to `bob` are returned.
3. Run:
   ```bash
   ./bin/sq ready --unassigned --json
   ```
4. Confirm assigned tasks are excluded and unassigned ready tasks remain.

### 4. Label and metadata filters
1. Run:
   ```bash
   ./bin/sq ready --label backend --json
   ./bin/sq ready --label-any backend,api --json
   ./bin/sq ready --metadata-field team=platform --json
   ./bin/sq ready --has-metadata-key team --json
   ```
2. Confirm each filter narrows results correctly.
3. Confirm malformed metadata-field values fail or are documented as invalid behavior if observed.

### 5. Parent and type filters
1. Run:
   ```bash
   ./bin/sq ready --parent $C --type task --json
   ```
2. Confirm descendants of the specified parent are returned.
3. Confirm unrelated ready tasks are excluded.

### 6. Priority, limit, and sorting
1. Run:
   ```bash
   ./bin/sq ready --priority 2 --json
   ./bin/sq ready --limit 1 --json
   ./bin/sq ready --sort priority --json
   ./bin/sq ready --sort oldest --json
   ```
2. Confirm priority filtering works.
3. Confirm limit truncates the returned list.
4. Confirm `priority` sorting orders by priority then creation time.
5. Confirm `oldest` sorting orders by oldest creation time first.
6. If supported, also verify `--sort hybrid` behaves like current priority ordering.

### 7. Accepted compatibility flags
1. Run the command with accepted parity flags:
   ```bash
   ./bin/sq ready --gated --json
   ./bin/sq ready --include-deferred --json
   ./bin/sq ready --include-ephemeral --json
   ./bin/sq ready --plain --json
   ./bin/sq ready --pretty --json
   ./bin/sq ready --mol work-1 --json
   ./bin/sq ready --mol-type work --json
   ./bin/sq ready --rig bd --json
   ./bin/sq ready --actor tester --db "$SQ_DB_PATH" --dolt-auto-commit off --json
   ```
2. Confirm each invocation exits successfully.
3. Confirm these accepted flags do not corrupt output formatting or produce unexpected writes.

### 8. Error handling
1. Verify missing values fail cleanly:
   ```bash
   ./bin/sq ready --assignee
   ./bin/sq ready --limit
   ./bin/sq ready --priority
   ./bin/sq ready --sort
   ```
2. Verify invalid numeric values fail cleanly:
   ```bash
   ./bin/sq ready --limit x
   ./bin/sq ready --priority x
   ```
3. Verify unknown flags fail:
   ```bash
   ./bin/sq ready --wat
   ```
4. Confirm each invalid invocation exits non-zero and prints a usage-style error.

### 9. No side effects
1. Capture task state before and after running read-only ready invocations:
   ```bash
   ./bin/sq list --json > "$TMP_ROOT/before.json"
   ./bin/sq ready --json
   ./bin/sq ready --assignee bob --label backend --metadata-field team=platform --json
   ./bin/sq list --json > "$TMP_ROOT/after.json"
   ```
2. Confirm the tasks are unchanged except for any intentional setup created earlier.

## Cleanup Steps
1. Remove temporary files and database state:
   ```bash
   rm -rf "$TMP_ROOT"
   ```
2. Unset environment overrides:
   ```bash
   unset SQ_DB_PATH
   ```
3. Return to the repository root.

## Expected Result
- `sq ready` returns only actionable, unblocked work.
- supported filters and sort modes behave consistently.
- accepted compatibility flags do not break command execution.
- invalid invocations fail cleanly with usage errors.
- the command remains read-only and does not mutate task state.
