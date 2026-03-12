# Test Plan: `sq create`

## Scope

Validate `sq create` for:
- required title handling
- default field population
- explicit field population (`--description`, `--type`, `--priority`)
- JSON output structure
- actor/creator behavior via environment and compatibility flags
- dependency creation via `--deps`
- validation failures for malformed flags and invalid values
- persistence verification through follow-up `show`/`list` commands

This plan targets the top-level `sq create` command only.

## Initialization

1. Create a disposable workspace:
   ```bash
   TMP_DIR="$(mktemp -d -t sq-create-XXXXXX)"
   cd "$TMP_DIR"
   ```
2. Set the CLI path:
   ```bash
   SQ_BIN="/Users/auhanson/workspace/hnsn/squids-docs-bd-mlb/bin/sq"
   $SQ_BIN version
   ```
3. Initialize a fresh database:
   ```bash
   $SQ_BIN init --json
   ```
4. Seed dependency targets used by later tests:
   ```bash
   PARENT_JSON="$($SQ_BIN create "parent epic" --type epic --json)"
   PARENT_ID="$(printf '%s' "$PARENT_JSON" | jq -r '.id')"

   BLOCKER_JSON="$($SQ_BIN create "blocking task" --type task --priority 1 --json)"
   BLOCKER_ID="$(printf '%s' "$BLOCKER_JSON" | jq -r '.id')"
   ```

## Test Steps

### 1. Help and discoverability

1. Run:
   ```bash
   $SQ_BIN help create
   ```
2. Confirm help output documents:
   - positional title
   - `--description`
   - `--type`
   - `--priority`
   - `--deps`
   - `--json`
   - global flags

### 2. Minimal create flow

1. Create a task with only a title:
   ```bash
   BASIC_JSON="$($SQ_BIN create "basic created task" --json)"
   ```
2. Verify JSON output includes:
   - `id`
   - `title`
   - `status`
   - `created_at`
   - `updated_at`
3. Confirm defaults:
   - `title == "basic created task"`
   - status is open
   - default issue type is the expected default for sq create
   - default priority matches current implementation expectations
4. Save the id:
   ```bash
   BASIC_ID="$(printf '%s' "$BASIC_JSON" | jq -r '.id')"
   ```

### 3. Explicit field population

1. Create with full common flags:
   ```bash
   FULL_JSON="$($SQ_BIN create "full create task" \
     --description "detailed description" \
     --type feature \
     --priority 0 \
     --json)"
   ```
2. Verify the JSON output contains the supplied description, type, and priority.
3. Save the id as `FULL_ID`.
4. Cross-check persistence:
   ```bash
   $SQ_BIN show "$FULL_ID" --json
   ```
5. Confirm the persisted object matches the create response.

### 4. Dependency creation via `--deps`

1. Create a task with dependency flags:
   ```bash
   DEPS_JSON="$($SQ_BIN create "task with deps" \
     --deps "parent-child:$PARENT_ID,blocks:$BLOCKER_ID" \
     --json)"
   ```
2. Save the new id as `DEPS_ID`.
3. Verify the task exists:
   ```bash
   $SQ_BIN show "$DEPS_ID" --json
   ```
4. Cross-check dependencies using command-family readers:
   ```bash
   $SQ_BIN children "$PARENT_ID" --json
   $SQ_BIN blocked --json
   $SQ_BIN dep list "$DEPS_ID" --json
   ```
5. Confirm:
   - the new task appears under the specified parent
   - the new task is blocked by the blocker task
   - dependency references are stored as expected

### 5. Creator/actor behavior

1. Create with an explicit actor environment value:
   ```bash
   BD_ACTOR="alice" $SQ_BIN create "actor created task" --json
   ```
2. If creator metadata is persisted or surfaced, confirm it matches `alice`.
3. Also verify accepted compatibility flags do not break creation:
   ```bash
   $SQ_BIN create "compat task" --actor tester --json
   $SQ_BIN create "compat task db" --db "$TMP_DIR/.sq/tasks.sqlite" --json
   $SQ_BIN create "compat task auto commit" --dolt-auto-commit off --json
   ```
4. Confirm these invocations succeed and create valid issues.

### 6. Human-readable mode

1. Create without `--json`:
   ```bash
   HUMAN_OUT="$($SQ_BIN create "human mode task")"
   ```
2. Confirm output is human-readable.
3. Confirm it includes a created issue id that can be used with `sq show`.
4. Verify the issue actually exists by looking it up after extracting the id.

### 7. Validation and error handling

Verify each case exits non-zero with a clear error:

1. Missing title:
   ```bash
   $SQ_BIN create
   ```
2. Invalid priority:
   ```bash
   $SQ_BIN create "bad priority" --priority nope --json
   ```
3. Invalid dependency encoding:
   ```bash
   $SQ_BIN create "bad deps" --deps ":" --json
   ```
4. Unknown flag:
   ```bash
   $SQ_BIN create "bad flag" --wat
   ```
5. Missing value for `--description` if supported as a required-value flag:
   ```bash
   $SQ_BIN create "missing description value" --description
   ```
6. Missing value for `--type`:
   ```bash
   $SQ_BIN create "missing type value" --type
   ```
7. Missing value for `--priority`:
   ```bash
   $SQ_BIN create "missing priority value" --priority
   ```
8. Missing value for compatibility flags:
   ```bash
   $SQ_BIN create "missing actor value" --actor
   $SQ_BIN create "missing db value" --db
   $SQ_BIN create "missing auto commit value" --dolt-auto-commit
   ```

### 8. Persistence and listing cross-checks

1. Run:
   ```bash
   $SQ_BIN list --json
   ```
2. Confirm all successfully created issue ids appear in the list output.
3. Spot-check a subset with `sq show <id> --json` and verify field persistence.
4. Confirm failed create attempts did not produce extra issues.

## Cleanup

1. Leave the temporary workspace:
   ```bash
   cd /
   ```
2. Remove the disposable directory and database:
   ```bash
   rm -rf "$TMP_DIR"
   ```
3. Confirm no repository worktree artifacts remain.

## Expected Outcome

- `sq create` creates new issues reliably in both JSON and human modes.
- Required fields and default values are applied correctly.
- Explicit description, type, priority, and dependency options persist as expected.
- Invalid inputs fail cleanly without partial or hidden issue creation.
- Follow-up `show`, `list`, `children`, `blocked`, and `dep list` checks confirm the created issue state is durable and queryable.
