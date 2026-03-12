# Test Plan: `sq todo`

## Scope

Validate the top-level `sq todo` command family for:
- default listing behavior (`sq todo`)
- explicit list behavior (`sq todo list`)
- interaction with task-type/open-state filtering
- JSON output structure
- empty-list behavior
- discoverability/help behavior for the command family
- error handling for unknown subcommands and invalid arguments
- interoperability with `sq todo add` and `sq todo done` as the lifecycle used to exercise list behavior

This test plan targets the `sq todo` command family entry point. Subcommand-specific deep coverage for `add` and `done` should live in their own dedicated plans.

## Initialization

1. Create a disposable workspace:
   ```bash
   TMP_DIR="$(mktemp -d -t sq-todo-XXXXXX)"
   cd "$TMP_DIR"
   ```
2. Set the binary under test:
   ```bash
   SQ_BIN="/Users/auhanson/workspace/hnsn/squids-docs-bd-dsr/bin/sq"
   $SQ_BIN version
   ```
3. Initialize a fresh sq database:
   ```bash
   $SQ_BIN init --json
   ```
4. Seed baseline issues to verify filtering:
   ```bash
   OPEN_TASK_JSON="$($SQ_BIN create "plain open task" --type task --json)"
   OPEN_TASK_ID="$(printf '%s' "$OPEN_TASK_JSON" | jq -r '.id')"

   BUG_JSON="$($SQ_BIN create "bug not todo" --type bug --json)"
   BUG_ID="$(printf '%s' "$BUG_JSON" | jq -r '.id')"

   CLOSED_TASK_JSON="$($SQ_BIN create "task to close" --type task --json)"
   CLOSED_TASK_ID="$(printf '%s' "$CLOSED_TASK_JSON" | jq -r '.id')"
   $SQ_BIN close "$CLOSED_TASK_ID" --reason "setup" --json
   ```
5. Seed todo-style tasks using the todo family itself:
   ```bash
   TODO1_JSON="$($SQ_BIN todo add "todo item one" --description "first todo" --json)"
   TODO1_ID="$(printf '%s' "$TODO1_JSON" | jq -r '.id')"

   TODO2_JSON="$($SQ_BIN todo add "todo item two" --priority 1 --description "second todo" --json)"
   TODO2_ID="$(printf '%s' "$TODO2_JSON" | jq -r '.id')"
   ```

## Test Steps

### 1. Help and command surface

1. Run top-level usage via invalid subcommand handling:
   ```bash
   $SQ_BIN todo wat
   ```
2. Confirm the command exits non-zero and reports an unknown subcommand rather than silently ignoring it.
3. Run umbrella help if supported through the generic help system:
   ```bash
   $SQ_BIN help todo
   ```
4. Confirm the command family is discoverable from help output and references `list`, `add`, and `done` behavior.

### 2. Default listing behavior

1. Run the default invocation:
   ```bash
   $SQ_BIN todo
   ```
2. Confirm it behaves like `sq todo list`.
3. Verify open task-type items are returned, including:
   - `plain open task`
   - `todo item one`
   - `todo item two`
4. Verify excluded items are not returned:
   - closed task (`$CLOSED_TASK_ID`)
   - non-task type issue (`$BUG_ID`)

### 3. Explicit list behavior

1. Run explicit list mode:
   ```bash
   $SQ_BIN todo list --json
   ```
2. Confirm the payload is valid JSON.
3. Confirm the result is an array of task objects.
4. Verify every returned object has:
   - `id`
   - `title`
   - `status`
   - `issue_type`
5. Confirm every returned item has:
   - `status == open`
   - `issue_type == task`
6. Confirm the returned ids include `$OPEN_TASK_ID`, `$TODO1_ID`, and `$TODO2_ID`.
7. Confirm the returned ids do not include `$CLOSED_TASK_ID` or `$BUG_ID`.

### 4. Human-readable list behavior

1. Run:
   ```bash
   $SQ_BIN todo list
   ```
2. Confirm output is readable and includes the expected task ids/titles.
3. Verify the command does not emit errors or usage text during normal list execution.

### 5. Lifecycle interaction with `done`

1. Mark one todo item done:
   ```bash
   $SQ_BIN todo done "$TODO1_ID" --reason "finished during test" --json
   ```
2. Re-run:
   ```bash
   $SQ_BIN todo list --json
   ```
3. Confirm `$TODO1_ID` no longer appears.
4. Confirm `$TODO2_ID` and `$OPEN_TASK_ID` still appear.
5. Confirm the closed todo is still visible via `sq show "$TODO1_ID" --json` if a manual cross-check is desired.

### 6. Empty-list behavior

1. Close every remaining open task-type item created during setup:
   ```bash
   $SQ_BIN close "$OPEN_TASK_ID" --reason "cleanup prep" --json
   $SQ_BIN todo done "$TODO2_ID" --reason "cleanup prep" --json
   ```
2. Re-run:
   ```bash
   $SQ_BIN todo list --json
   ```
3. Confirm the output is a valid empty JSON value/empty list shape as implemented.
4. Run:
   ```bash
   $SQ_BIN todo
   ```
5. Confirm the default invocation also reports an empty result cleanly.

### 7. Validation and error handling

Verify each case exits non-zero and returns a useful error message:

1. Unknown subcommand:
   ```bash
   $SQ_BIN todo wat
   ```
2. `done` without id:
   ```bash
   $SQ_BIN todo done
   ```
3. `add` without title:
   ```bash
   $SQ_BIN todo add
   ```
4. Invalid priority for `add`:
   ```bash
   $SQ_BIN todo add "bad priority" --priority nope --json
   ```
5. Unknown flag on list/default path:
   ```bash
   $SQ_BIN todo --wat
   ```
6. Unknown flag on done path:
   ```bash
   $SQ_BIN todo done "$TODO2_ID" --wat
   ```
7. Unknown/nonexistent id for done:
   ```bash
   $SQ_BIN todo done bd-missing --reason "nope" --json
   ```

### 8. Cross-check against generic list/show

1. Run:
   ```bash
   $SQ_BIN list --json
   ```
2. Confirm todo-visible items are a subset of open task-type issues from the general list output.
3. Spot-check one active todo item with:
   ```bash
   $SQ_BIN show "$TODO2_ID" --json
   ```
4. Confirm fields shown there align with what `sq todo list --json` returns.

## Cleanup

1. Leave the temporary workspace:
   ```bash
   cd /
   ```
2. Remove the disposable directory and database:
   ```bash
   rm -rf "$TMP_DIR"
   ```
3. Confirm no test artifacts remain in the repository worktree.

## Expected Outcome

- `sq todo` defaults to listing open task-type items.
- `sq todo list` returns only open tasks, not closed tasks or non-task issue types.
- `sq todo done` removes completed todo items from subsequent todo listings.
- Empty-state behavior is clean and machine-readable in JSON mode.
- Invalid invocations fail explicitly without mutating unrelated issues.
- The todo command family remains consistent with the broader sq task lifecycle.
