# Test Plan: `sq init`

## Scope
Validate `sq init` for first-run initialization, idempotent re-initialization, database path selection, environment-driven path overrides, schema reporting, and failure behavior when the database path cannot be created.

## Initialization
1. Create an isolated temporary workspace.
2. Start from a directory with no existing `.sq/` directory.
3. Confirm the working tree is clean and that no pre-existing `SQ_DB_PATH` value is exported.
4. Prepare two database targets for testing:
   - default path via current working directory
   - explicit alternate path via `SQ_DB_PATH`
5. Prepare one invalid path case where the parent path is a file instead of a directory.

## Test Steps

### 1. First-run initialization with default path
1. `cd` into the empty temp workspace.
2. Run:
   - `sq init`
3. Verify the command succeeds.
4. Verify output is valid JSON.
5. Verify the JSON payload includes:
   - `command: "init"`
   - `ok: true`
   - `db_path`
   - `schema_version`
6. Verify the created database path matches the default discovery path under `.sq/tasks.sqlite`.
7. Verify the database file now exists on disk.
8. Verify the `.sq/` directory was created automatically.

### 2. Idempotent re-run on an already initialized workspace
1. Run `sq init` again in the same workspace.
2. Verify the command still succeeds.
3. Verify it reports the same database path.
4. Verify `schema_version` remains stable.
5. Verify the second run does not corrupt or recreate the database unexpectedly.

### 3. Initialization using `SQ_DB_PATH`
1. Export `SQ_DB_PATH` to a fresh alternate path, for example:
   - `export SQ_DB_PATH="$(pwd)/custom/location/tasks.sqlite"`
2. Run `sq init`.
3. Verify the command succeeds.
4. Verify `db_path` in the JSON payload matches the exported path exactly.
5. Verify parent directories were created automatically.
6. Verify the database file exists at the custom location.

### 4. Re-run with the same explicit path
1. Keep `SQ_DB_PATH` set to the same alternate path.
2. Run `sq init` again.
3. Verify the command succeeds and remains idempotent.
4. Verify the reported path and schema version remain unchanged.

### 5. Workspace isolation between default and custom path cases
1. Run one initialization using the default path in workspace A.
2. Run another initialization using `SQ_DB_PATH` in workspace B.
3. Verify each workspace creates or reports only its own database target.
4. Verify there is no accidental leakage between the two path-selection modes.

### 6. Schema reporting sanity check
1. After a successful `sq init`, capture the returned `schema_version`.
2. Verify it is a positive integer.
3. If available, cross-check with a follow-up command that reads the same database and confirms it is usable, for example:
   - `sq count --json`
4. Verify the initialized database can be opened by later commands without additional setup.

### 7. Failure path when the parent path is invalid
1. Create a file where a parent directory would need to exist.
2. Point `SQ_DB_PATH` to a child path under that file.
3. Run `sq init`.
4. Verify the command fails.
5. Verify stderr contains a concrete filesystem/path creation error.
6. Verify no partial database file is created.

### 8. Command-line help / flag surface documentation check
1. Run:
   - `sq help init`
2. Verify help output documents `sq init` usage.
3. Run:
   - `sq init --help`
4. Record observed behavior explicitly:
   - if command-specific help is supported, verify usage/help output
   - if command-specific help is not implemented and init executes instead, document that exact current behavior as a follow-up concern
5. Run:
   - `sq init --json`
6. Record whether the command accepts or ignores the flag while preserving successful initialization semantics.

### 9. Repeatability across fresh temp directories
1. Repeat the default-path initialization in at least two fresh temp directories.
2. Verify both succeed independently.
3. Verify each created database path is local to that temp directory.

### 10. Post-init smoke validation
1. After initialization, run at least one read command against the same database:
   - `sq count --json`
   - or `sq status --json`
2. Verify the command succeeds.
3. Verify the initialized database is ready for normal sq usage.

## Cleanup
1. Unset `SQ_DB_PATH` if it was exported.
2. Remove all temporary workspaces and custom database files created for the test.
3. Confirm no unintended `.sq/` directories or sqlite files remain outside the temp locations.
4. Confirm the main repository working tree remains unchanged except for intended documentation edits.

## Notes
- Current implementation returns structured JSON on success.
- Current implementation chooses the database path from environment or current working directory rather than from explicit command-specific flags.
- A likely regression risk is changing default path creation, breaking idempotency, or altering the success payload fields consumed by tests and automation.
- If `sq init --help` still initializes instead of showing help, track that as an observed behavior gap rather than silently normalizing it in the test plan.