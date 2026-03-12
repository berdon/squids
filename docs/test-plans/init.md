# Test Plan: `sq init`

## Goal
Verify that `sq init` correctly initializes sq storage, reports the database path and schema version, behaves safely when run repeatedly, respects explicit database location overrides, and leaves the repository in a predictable initialized state.

## Initialization Steps
1. Open a disposable shell in the repository root.
2. Build the CLI:
   ```bash
   mkdir -p bin
   go build -o ./bin/sq ./cmd/sq
   ```
3. Create a clean temporary workspace for initialization checks:
   ```bash
   TMP_ROOT="$(mktemp -d -t sq-init-test-XXXXXX)"
   mkdir -p "$TMP_ROOT/project"
   cd "$TMP_ROOT/project"
   ```
4. Confirm there is no pre-existing sq database:
   ```bash
   find . -maxdepth 3 -type f | sort
   ```
5. Keep the repository build available if you want to run the binary by absolute path.

## Testing Steps

### 1. Basic initialization
1. Run:
   ```bash
   /path/to/repo/bin/sq init --json
   ```
2. Confirm the command exits successfully.
3. Confirm the JSON response contains:
   - `command`
   - `ok`
   - `db_path`
   - `schema_version`
4. Confirm:
   - `command` is `init`
   - `ok` is `true`
   - `schema_version` is a positive integer
5. Confirm the reported `db_path` now exists on disk.
6. Confirm the expected sq directory has been created:
   ```bash
   find . -maxdepth 3 -type f | sort
   ```

### 2. Default path behavior
1. From a clean project directory, run:
   ```bash
   /path/to/repo/bin/sq init --json
   ```
2. Confirm the database is created under the default sq storage location for the current directory.
3. Confirm subsequent sq commands can discover that database automatically:
   ```bash
   /path/to/repo/bin/sq ready --json
   ```
4. Confirm `ready` succeeds immediately after initialization.

### 3. Idempotency / repeated initialization
1. Run `sq init --json` a second time in the same directory:
   ```bash
   /path/to/repo/bin/sq init --json
   ```
2. Confirm the command still exits successfully.
3. Confirm the returned `db_path` is unchanged.
4. Confirm the schema version remains stable.
5. Confirm no duplicate or extra database files are created.

### 4. Explicit database path via environment
1. Create a separate clean directory:
   ```bash
   mkdir -p "$TMP_ROOT/env-db"
   cd "$TMP_ROOT/env-db"
   ```
2. Set an explicit database path:
   ```bash
   export SQ_DB_PATH="$TMP_ROOT/custom/location/tasks.sqlite"
   ```
3. Run:
   ```bash
   /path/to/repo/bin/sq init --json
   ```
4. Confirm the returned `db_path` matches `$SQ_DB_PATH`.
5. Confirm parent directories are created automatically.
6. Confirm the database file exists exactly at the requested path.

### 5. Repository-root initialization behavior
1. From the squids repository root, run:
   ```bash
   ./bin/sq init --json
   ```
2. Confirm the returned `db_path` is inside the repository’s sq storage location.
3. Confirm a follow-up read command works:
   ```bash
   ./bin/sq status --json
   ```
4. Confirm initialization does not require any pre-existing config file.

### 6. Interaction with follow-up commands
1. After initialization, verify that core commands operate against the created database:
   ```bash
   /path/to/repo/bin/sq ready --json
   /path/to/repo/bin/sq create "post-init smoke" --type task --priority 2 --json
   /path/to/repo/bin/sq list --json
   ```
2. Confirm `create` succeeds and `list` shows the new task.
3. Confirm this validates the initialized database is usable, not only present.

### 7. Help and argument handling
1. Run:
   ```bash
   /path/to/repo/bin/sq init --help
   ```
2. Record the current observed behavior.
3. Confirm whether the command prints help text or executes initialization.
4. If current behavior differs from intended CLI conventions, note that as an observation for future parity work.
5. Verify that unsupported extra positional arguments are rejected if the implementation ever accepts any.

### 8. Failure-path checks
1. Force an invalid database location, for example by pointing `SQ_DB_PATH` under a non-directory file parent.
2. Run:
   ```bash
   export SQ_DB_PATH="$TMP_ROOT/notadir/tasks.sqlite"
   printf 'x' > "$TMP_ROOT/notadir"
   /path/to/repo/bin/sq init --json
   ```
3. Confirm the command exits non-zero.
4. Confirm stderr contains a runtime error explaining the directory creation/open failure.

## Cleanup Steps
1. Remove all temporary databases and directories:
   ```bash
   rm -rf "$TMP_ROOT"
   ```
2. Unset any environment overrides:
   ```bash
   unset SQ_DB_PATH
   ```
3. If you initialized the repository root during testing, remove any disposable sq state only if it was created solely for the test and is safe to delete.
4. Return to the repository root.

## Expected Result
- `sq init --json` succeeds in clean directories and creates a usable database.
- repeated initialization is safe and idempotent.
- explicit database path overrides are honored.
- downstream commands can use the initialized store immediately.
- invalid database paths fail cleanly with a runtime error.
- any observed quirks around `--help` are documented for future follow-up if needed.
