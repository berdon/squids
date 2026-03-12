# Test Plan: `sq help`

## Goal
Verify that `sq help` correctly documents the CLI, handles its supported flags, and renders command-specific help for top-level commands and subcommands without mutating any task state.

## Initialization Steps
1. Enter the repository root in a disposable shell session.
2. Build the CLI binary:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Confirm the binary is executable:
   ```bash
   ./bin/sq version
   ```
4. Prepare a temporary output scratch directory if capturing help snapshots:
   ```bash
   mkdir -p /tmp/sq-help-test
   ```
5. If testing command-specific help that touches initialized state later, optionally create a disposable database location:
   ```bash
   export SQ_DB_PATH=/tmp/sq-help-test/tasks.sqlite
   ```

## Testing Steps

### 1. Top-level help output
1. Run:
   ```bash
   ./bin/sq help
   ```
2. Confirm the output includes:
   - the general application help text
   - `Usage:`
   - `sq [flags]`
   - `sq [command]`
   - the major command groups
   - `Global Flags:`
3. Confirm the command exits successfully.

### 2. `--help` handling for the help command itself
1. Run:
   ```bash
   ./bin/sq help --help
   ```
2. Confirm the output includes:
   - `Help provides help for any command in the application.`
   - `sq help [command] [flags]`
   - `--all`
   - `help for help`
   - `Global Flags:`
3. Confirm the command exits successfully.

### 3. Full-document help mode
1. Run:
   ```bash
   ./bin/sq help --all
   ```
2. Confirm the output includes:
   - `# sq — Complete Command Reference`
   - `## Table of Contents`
   - core command entries such as `sq init`, `sq create`, and `sq ready`
3. Confirm the output can be redirected to a file:
   ```bash
   ./bin/sq help --all > /tmp/sq-help-test/help-all.md
   ```
4. Confirm the file is non-empty.

### 4. Command-specific help for a top-level command
1. Run:
   ```bash
   ./bin/sq help create
   ```
2. Confirm the output includes:
   - `Create a task`
   - `sq create [title] [flags]`
   - command-specific flags like `--description`, `--priority`, and `--type`
   - `Global Flags:`
3. Repeat with another implemented command such as:
   ```bash
   ./bin/sq help ready
   ```
4. Confirm the output includes ready-specific filters and usage text.

### 5. Command-specific help for grouped commands and compatibility surfaces
1. Run:
   ```bash
   ./bin/sq help label
   ./bin/sq help query
   ./bin/sq help backup
   ./bin/sq help quickstart
   ```
2. Confirm each command exits successfully and shows command-appropriate usage text.
3. Verify at least one grouped command help includes subcommand-oriented guidance.

### 6. Pass-through compatibility/global flags
1. Run the help command with accepted compatibility flags:
   ```bash
   ./bin/sq help --json
   ./bin/sq help --quiet
   ./bin/sq help --verbose
   ./bin/sq help --profile
   ./bin/sq help --readonly
   ./bin/sq help --sandbox
   ./bin/sq help --actor tester
   ./bin/sq help --db /tmp/sq-help-test/tasks.sqlite
   ./bin/sq help --dolt-auto-commit off
   ```
2. Confirm each invocation exits successfully.
3. Confirm these flags do not corrupt the help text or suppress required usage output unexpectedly.

### 7. Error handling
1. Verify too many positional arguments fail:
   ```bash
   ./bin/sq help create list
   ```
2. Confirm the command exits non-zero and prints a usage-style error.
3. Verify unknown flags fail:
   ```bash
   ./bin/sq help --wat
   ```
4. Confirm the command exits non-zero and reports `unknown flag`.

### 8. Relationship with direct command help
1. Compare:
   ```bash
   ./bin/sq help create
   ./bin/sq create --help
   ```
2. Confirm both expose coherent command help for `create`.
3. Repeat for another command such as `ready`, `query`, or `label`.

### 9. Stability / no side effects
1. Capture issue state before and after running help commands if desired:
   ```bash
   ./bin/sq ready --json > /tmp/sq-help-test/before.json
   ./bin/sq help
   ./bin/sq help --all
   ./bin/sq help create
   ./bin/sq ready --json > /tmp/sq-help-test/after.json
   ```
2. Confirm no state-changing side effects occurred as a result of invoking help.

## Cleanup Steps
1. Remove temporary output files:
   ```bash
   rm -rf /tmp/sq-help-test
   ```
2. Unset any temporary environment variables:
   ```bash
   unset SQ_DB_PATH
   ```
3. Delete any disposable database or scratch directories created solely for this test.

## Expected Result
- `sq help` and its supported forms render successfully.
- `--all` produces a full reference document.
- command-specific help works for implemented commands and grouped surfaces.
- accepted compatibility/global flags do not break help behavior.
- invalid invocations fail with clear usage errors.
- no task data is mutated by running help commands.
