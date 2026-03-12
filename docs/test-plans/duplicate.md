# Test Plan: `sq duplicate`

## Scope
Validate the `sq duplicate` command/subcommand tuple end to end, including setup requirements, success cases, error handling, output format checks, and cleanup. This plan is intended to support repeatable manual or automated verification without relying on project-global state.

## Initialization Steps
1. Create an isolated temporary workspace for the test run.
2. `cd` into the temporary workspace.
3. If the command works against sq state, initialize a fresh database with:
   - `sq init --json`
4. Export only the environment variables needed for the scenario under test, and record them so they can be removed during cleanup.
5. Seed any prerequisite issues, labels, dependencies, comments, gates, hooks, or backup files required by the tuple before running the main assertions.
6. Confirm the command under test is available from the target build or PATH and record the exact binary used.

## Testing Steps

### 1. Help / discovery surface
1. Run `sq help duplicate` if applicable.
2. Run `sq duplicate --help`.
3. Verify the command name, synopsis, supported flags, and examples (if any) are documented consistently.
4. Verify help exits successfully and does not mutate repository or database state.

### 2. Baseline success path
1. Execute the primary tuple in a minimal valid scenario.
2. Verify it exits successfully.
3. Verify stdout/stderr match the expected mode for the tuple:
   - human-readable text for default interactive output
   - JSON when `--json` is supported
4. Verify any created or updated sq state matches the observed output.

### 3. Flag and option coverage
1. Enumerate the tuple-specific flags shown in help.
2. Exercise each supported flag at least once in a valid combination.
3. For commands that accept `--json`, verify the JSON is valid and semantically complete.
4. For commands with aliases, verify the alias behavior matches the canonical form.
5. For command groups, verify behavior both with and without required subcommand arguments.

### 4. Input validation and failure behavior
1. Run the tuple with missing required arguments.
2. Run the tuple with an unknown flag.
3. Run the tuple with malformed or non-existent identifiers/paths where applicable.
4. Verify each invalid invocation fails with a usage/runtime error that is specific enough to diagnose the problem.
5. Verify failed invocations do not partially mutate sq state.

### 5. State verification
1. After each successful mutation, inspect the resulting state using a follow-up sq command.
2. Confirm that any issue/task metadata, status, dependency, label, or file-system side effect is actually persisted.
3. If the tuple is intended to be read-only, verify that no task state changes occur.
4. If the tuple is intended to be idempotent, run it twice and confirm the second invocation is safe.

### 6. Concrete command checks for `sq duplicate`
Use the following command-focused checks as a starting point:
1. Run `sq duplicate --help` and record the observed output/exit code.
2. Run `sq duplicate` and record the observed output/exit code.
3. Run `sq duplicate --json` and record the observed output/exit code.


### 7. Cross-command follow-up checks
1. If `sq duplicate` creates or mutates sq data, validate the result with a secondary command such as `sq show`, `sq list`, `sq count --json`, or `sq status --json`.
2. If `sq duplicate` is documentation- or reporting-oriented, compare default output and help output for consistency.
3. If the tuple is part of a command family, run the adjacent subcommands needed to confirm the family remains coherent.

## Cleanup Steps
1. Remove any temporary workspace created for the test.
2. Remove any temporary database files, backup files, completion scripts, or hook fixtures created during setup.
3. Unset environment variables exported specifically for the test run.
4. If the tuple mutated sq state in a shared database during exploratory testing, delete or reset that state before ending the test.
5. Confirm there are no unintended repository changes outside the planned test artifacts.

## Notes
- File name for this plan follows the required convention: `docs/test-plans/duplicate.md`.
- Target tuple: `sq duplicate`.
- If observed runtime behavior differs from help text or from neighboring command-family behavior, capture that mismatch as follow-up work instead of silently normalizing it.
