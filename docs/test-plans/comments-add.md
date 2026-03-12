# Test Plan: `sq comments add`

## Scope

Validate the `sq comments add` subcommand for:
- inline comment creation
- file-backed comment creation via `-f` / `--file`
- explicit author assignment via `-a` / `--author`
- JSON and human output modes
- validation failures for missing args, missing flag values, missing files, and unknown flags
- compatibility/global flag handling (`--actor`, `--db`, `--dolt-auto-commit`, `--quiet`, `--verbose`, `--profile`, `--readonly`, `--sandbox`)

## Initialization

1. Create a disposable working directory:
   ```bash
   TMP_DIR="$(mktemp -d -t sq-comments-add-XXXXXX)"
   cd "$TMP_DIR"
   ```
2. Build the CLI under test:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq version
   ```
3. Initialize a fresh database:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq init --json
   ```
4. Create a target issue for comments:
   ```bash
   SQ_BIN="/Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq"
   ISSUE_JSON="$($SQ_BIN create "comment target" --json)"
   ISSUE_ID="$(printf '%s' "$ISSUE_JSON" | jq -r '.id')"
   ```
5. Prepare a file-backed comment fixture:
   ```bash
   printf 'comment from file\nsecond line\n' > notes.txt
   ```

## Test Steps

### 1. Help and command surface

1. Verify subcommand help:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add --help
   ```
2. Confirm the help text documents:
   - positional issue id and text
   - `-a` / `--author`
   - `-f` / `--file`
   - global compatibility flags

### 2. Inline comment creation

1. Add an inline comment in JSON mode:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "hello from inline" --json
   ```
2. Verify response fields:
   - `issue_id` matches `$ISSUE_ID`
   - `body` is `hello from inline`
   - `id` is present
   - `created_at` is present
3. List comments to confirm persistence:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments "$ISSUE_ID" --json
   ```
4. Confirm the new comment is present exactly once.

### 3. File-backed comment creation

1. Add a comment using `-f`:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" -f notes.txt --json
   ```
2. Verify the stored body matches the file contents, including embedded newline content.
3. Re-list comments and confirm both inline and file-backed comments are present.

### 4. Explicit author assignment

1. Add a comment with an explicit author:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "authored comment" --author alice --json
   ```
2. Verify JSON output contains `"author": "alice"`.
3. Re-list comments and confirm the stored author remains `alice`.

### 5. Human output mode

1. Add a comment without `--json`:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "human output comment"
   ```
2. Verify the output contains `Added comment` and the target issue id.
3. Run:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments "$ISSUE_ID"
   ```
4. Confirm human listing shows:
   - numeric comment ids
   - timestamps
   - author names (or fallback such as `unknown` when unset)
   - indented comment bodies

### 6. Compatibility/global flags

1. Confirm accepted flags do not break creation:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "compat comment" --actor tester --json
   ```
2. Run no-op compatibility/global flags individually or in small combinations:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "quiet comment" --quiet --json
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "verbose comment" --verbose --json
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "sandbox comment" --sandbox --json
   ```
3. If testing `--db`, point it at the initialized sqlite database path and confirm the comment is written to that database.
4. If testing `--dolt-auto-commit`, verify it is accepted as a compatibility flag and does not change sqlite behavior.

### 7. Validation and error handling

Run each case and verify non-zero exit status plus a useful error message:

1. Missing issue id and text:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add
   ```
2. Missing text when no file is provided:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID"
   ```
3. Missing value for `--author`:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" --author
   ```
4. Missing value for `--file`:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" -f
   ```
5. Nonexistent file:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" -f missing.txt
   ```
6. Unknown flag:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" "bad flag" --wat
   ```
7. Missing values for compatibility flags:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" --actor
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" --db
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add "$ISSUE_ID" --dolt-auto-commit
   ```
8. Missing issue target:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments add bd-missing "orphan comment"
   ```

### 8. Persistence verification

1. After all successful additions, list comments:
   ```bash
   /Users/auhanson/workspace/hnsn/squids-docs-bd-gb9/bin/sq comments "$ISSUE_ID" --json
   ```
2. Confirm the total count matches the number of successful add operations.
3. Confirm bodies and authors are preserved exactly.

## Cleanup

1. Remove temporary files and directories:
   ```bash
   cd /
   rm -rf "$TMP_DIR"
   ```
2. If a shared database path was used for any `--db` tests, delete that temporary database as well.
3. Verify no test artifacts remain in the repository worktree.

## Expected Outcome

- Successful invocations create comments attached to the target issue.
- `-f` / `--file` loads comment content from disk.
- `-a` / `--author` sets the stored author field.
- JSON mode returns structured comment objects.
- Human mode prints a readable confirmation/listing.
- Invalid invocations fail cleanly without creating comments.
- Compatibility/global flags are either accepted as documented or fail with explicit missing-value/unknown-flag errors.
