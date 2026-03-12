# Test Plan: `sq count`

## Scope
Validate the `sq count` command end to end for human-readable output, JSON output, filtering behavior, help text, accepted global compatibility flags, and error handling.

## Initialization
1. Create an isolated temp workspace.
2. Choose a fresh database path, for example `$(pwd)/.sq/tasks.sqlite` inside the temp workspace.
3. Initialize sq storage:
   - `sq init --json`
4. Seed representative issues so count results can be verified across statuses:
   - `sq create "open task" --type task --priority 1 --json`
   - `sq create "second open task" --type bug --priority 2 --json`
   - `sq create "in progress task" --type task --priority 2 --json`
   - `sq create "closed task" --type chore --priority 3 --json`
5. Transition seeded issues so multiple statuses exist:
   - `sq update <in-progress-id> --status in_progress --json`
   - `sq close <closed-id> --reason "test setup" --json`
6. Record the expected totals for later assertions:
   - total issues = 4
   - open issues = 2
   - in_progress issues = 1
   - closed issues = 1

## Test Steps

### 1. Help and usage surface
1. Run `sq count --help`.
2. Verify output includes:
   - `Count issues matching the specified filters.`
   - `Usage:`
   - `sq count [flags]`
   - `-s, --status string`
   - `Global Flags:`
3. Run `sq help count` and verify it points to the count command.

### 2. Default human-readable output
1. Run `sq count`.
2. Verify stdout is a plain integer with no JSON wrapper.
3. Verify the value matches the total seeded issue count.

### 3. JSON output
1. Run `sq count --json`.
2. Verify output is valid JSON.
3. Verify the payload includes exactly the `count` field needed by parity callers.
4. Verify `count` equals the total seeded issue count.

### 4. Status filtering
1. Run `sq count --status open --json`.
2. Verify `count == 2`.
3. Run `sq count --status in_progress --json`.
4. Verify `count == 1`.
5. Run `sq count --status closed --json`.
6. Verify `count == 1`.
7. Run `sq count -s open`.
8. Verify stdout is the plain integer `2`.
9. Run `sq count --status deferred --json` on a database with no deferred issues and verify `count == 0`.

### 5. Accepted global compatibility flags
1. Run `sq count --json --quiet`.
2. Run `sq count --json --verbose`.
3. Run `sq count --json --profile`.
4. Run `sq count --json --readonly`.
5. Run `sq count --json --sandbox`.
6. Run `sq count --json --actor tester`.
7. Run `sq count --json --db <db-path>`.
8. Run `sq count --json --dolt-auto-commit off`.
9. Verify all accepted compatibility invocations succeed and still return the expected count.

### 6. Unknown flag handling
1. Run `sq count --wat`.
2. Verify the command exits with usage error semantics.
3. Verify stderr includes `unknown flag`.

### 7. Missing-value behavior for status flag
1. Run `sq count --status`.
2. Verify current behavior explicitly:
   - either it should fail with a usage error, or
   - if compatibility intentionally tolerates the missing value, document the observed result.
3. If behavior differs from intended command conventions, file follow-up work.

### 8. Empty database behavior
1. Repeat in a fresh initialized database with zero created issues.
2. Run `sq count` and verify stdout is `0`.
3. Run `sq count --json` and verify output is `{ "count": 0 }` semantically.

### 9. Parity regression check
1. Run the sq parity suite from the repository root:
   - `TARGET_BIN=./bin/sq ./scripts/parity/run-parity.sh`
2. Verify the parity step covering `count` passes.
3. Confirm both of these cases remain green:
   - `count --json`
   - `count --status open --json`

## Cleanup
1. Remove the temp workspace and test database.
2. Clear any exported environment variables used during setup, such as `SQ_DB_PATH` or `SQ_ACTOR`.
3. If the parity suite produced temporary artifacts, remove them.
4. Confirm no unintended repository changes remain outside the intended worktree.

## Notes
- This command currently supports status filtering plus accepted global compatibility flags.
- The primary regression risk is changing default output back to JSON, which breaks bd-style CLI behavior.
- The primary parity risk is removing the `count` JSON field or changing flag acceptance semantics.