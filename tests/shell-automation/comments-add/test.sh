#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-comments-add-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[comments-add] FAIL: $*" >&2
  exit 1
}

assert_eq() {
  local got="$1"
  local want="$2"
  local context="$3"
  if [[ "$got" != "$want" ]]; then
    fail "$context: expected '$want', got '$got'"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local context="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    fail "$context: expected '$needle' in output"
  fi
}

json_eval() {
  local payload="$1"
  local expr="$2"
  JSON_INPUT="$payload" "$PYTHON" - "$expr" <<'PY'
import json, os, sys
obj = json.loads(os.environ["JSON_INPUT"])
expr = sys.argv[1]
value = eval(expr, {"__builtins__": {}}, {"obj": obj, "len": len})
if isinstance(value, (dict, list)):
    print(json.dumps(value))
elif isinstance(value, bool):
    print("true" if value else "false")
elif value is None:
    print("")
else:
    print(value)
PY
}

json_b64() {
  local payload="$1"
  local expr="$2"
  JSON_INPUT="$payload" "$PYTHON" - "$expr" <<'PY'
import base64, json, os, sys
obj = json.loads(os.environ["JSON_INPUT"])
expr = sys.argv[1]
value = eval(expr, {"__builtins__": {}}, {"obj": obj})
print(base64.b64encode(value.encode()).decode())
PY
}

run_capture() {
  local name="$1"
  shift
  local out_file="$TMP_DIR/${name}.out"
  local err_file="$TMP_DIR/${name}.err"
  set +e
  "$@" >"$out_file" 2>"$err_file"
  RUN_CODE=$?
  set -e
  RUN_OUT="$(<"$out_file")"
  RUN_ERR="$(<"$err_file")"
}

export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
NOTES_FILE="$TMP_DIR/notes.txt"
MISSING_FILE="$TMP_DIR/missing.txt"

printf 'comment from file\nsecond line\n' > "$NOTES_FILE"

echo "[comments-add] binary=$SQ_BIN"
echo "[comments-add] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture create_issue "$SQ_BIN" create "comment target" --json
assert_eq "$RUN_CODE" "0" "create comment target"
ISSUE_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"

run_capture help_add "$SQ_BIN" comments add --help
assert_eq "$RUN_CODE" "0" "comments add --help"
assert_contains "$RUN_OUT" "sq comments add [issue-id] [text] [flags]" "comments add help usage"
assert_contains "$RUN_OUT" "--author string" "comments add help author flag"
assert_contains "$RUN_OUT" "--file string" "comments add help file flag"
assert_contains "$RUN_OUT" "--actor string" "comments add help global actor flag"
assert_contains "$RUN_OUT" "--db string" "comments add help global db flag"
assert_contains "$RUN_OUT" "--dolt-auto-commit string" "comments add help dolt flag"
assert_contains "$RUN_OUT" "--quiet" "comments add help quiet flag"
assert_contains "$RUN_OUT" "--verbose" "comments add help verbose flag"
assert_contains "$RUN_OUT" "--sandbox" "comments add help sandbox flag"
assert_contains "$RUN_OUT" "--readonly" "comments add help readonly flag"
assert_contains "$RUN_OUT" "--profile" "comments add help profile flag"

run_capture inline "$SQ_BIN" comments add "$ISSUE_ID" "hello from inline" --json
assert_eq "$RUN_CODE" "0" "inline comments add"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_id"]')" "$ISSUE_ID" "inline issue id"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "hello from inline" "inline body"
assert_eq "$(json_eval "$RUN_OUT" 'obj["id"] > 0')" "true" "inline id present"
assert_contains "$RUN_OUT" '"created_at"' "inline created_at present"

run_capture list_after_inline "$SQ_BIN" comments "$ISSUE_ID" --json
assert_eq "$RUN_CODE" "0" "list after inline"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "1" "inline list count"
assert_eq "$(json_eval "$RUN_OUT" 'obj[0]["body"]')" "hello from inline" "inline persisted body"

run_capture file_add "$SQ_BIN" comments add "$ISSUE_ID" -f "$NOTES_FILE" --json
assert_eq "$RUN_CODE" "0" "file-backed comments add"
assert_eq "$(json_b64 "$RUN_OUT" 'obj["body"]')" "Y29tbWVudCBmcm9tIGZpbGUKc2Vjb25kIGxpbmUK" "file-backed body"

run_capture list_after_file "$SQ_BIN" comments "$ISSUE_ID" --json
assert_eq "$RUN_CODE" "0" "list after file add"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "2" "file-backed list count"
assert_eq "$(json_b64 "$RUN_OUT" 'obj[1]["body"]')" "Y29tbWVudCBmcm9tIGZpbGUKc2Vjb25kIGxpbmUK" "file-backed persisted body"

run_capture author_add "$SQ_BIN" comments add "$ISSUE_ID" "authored comment" --author alice --json
assert_eq "$RUN_CODE" "0" "author comments add"
assert_eq "$(json_eval "$RUN_OUT" 'obj["author"]')" "alice" "author in add response"

run_capture list_after_author "$SQ_BIN" comments "$ISSUE_ID" --json
assert_eq "$RUN_CODE" "0" "list after author add"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "3" "author list count"
assert_eq "$(json_eval "$RUN_OUT" 'obj[2]["author"]')" "alice" "author persisted"

run_capture human_add "$SQ_BIN" comments add "$ISSUE_ID" "human output comment"
assert_eq "$RUN_CODE" "0" "human comments add"
assert_contains "$RUN_OUT" "Added comment" "human add confirmation"
assert_contains "$RUN_OUT" "$ISSUE_ID" "human add includes issue id"

run_capture human_list "$SQ_BIN" comments "$ISSUE_ID"
assert_eq "$RUN_CODE" "0" "human comments list"
assert_contains "$RUN_OUT" "unknown" "human list fallback author"
assert_contains "$RUN_OUT" "alice" "human list explicit author"
assert_contains "$RUN_OUT" "    hello from inline" "human list indented inline body"
assert_contains "$RUN_OUT" "    comment from file" "human list indented file body"
assert_contains "$RUN_OUT" "    second line" "human list multiline file body"
assert_contains "$RUN_OUT" "    authored comment" "human list authored comment"
assert_contains "$RUN_OUT" "    human output comment" "human list human body"

run_capture compat_actor "$SQ_BIN" comments add "$ISSUE_ID" "compat comment" --actor tester --json
assert_eq "$RUN_CODE" "0" "comments add with --actor"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "compat comment" "compat actor body"

run_capture compat_quiet "$SQ_BIN" comments add "$ISSUE_ID" "quiet comment" --quiet --json
assert_eq "$RUN_CODE" "0" "comments add with --quiet"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "quiet comment" "quiet body"

run_capture compat_verbose "$SQ_BIN" comments add "$ISSUE_ID" "verbose comment" --verbose --json
assert_eq "$RUN_CODE" "0" "comments add with --verbose"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "verbose comment" "verbose body"

run_capture compat_sandbox "$SQ_BIN" comments add "$ISSUE_ID" "sandbox comment" --sandbox --json
assert_eq "$RUN_CODE" "0" "comments add with --sandbox"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "sandbox comment" "sandbox body"

run_capture compat_db "$SQ_BIN" comments add "$ISSUE_ID" "db comment" --db "$SQ_DB_PATH" --json
assert_eq "$RUN_CODE" "0" "comments add with --db"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "db comment" "db body"

run_capture compat_dolt "$SQ_BIN" comments add "$ISSUE_ID" "dolt comment" --dolt-auto-commit off --json
assert_eq "$RUN_CODE" "0" "comments add with --dolt-auto-commit"
assert_eq "$(json_eval "$RUN_OUT" 'obj["body"]')" "dolt comment" "dolt body"

run_capture missing_all "$SQ_BIN" comments add
assert_eq "$RUN_CODE" "2" "missing issue id and text"
assert_contains "$RUN_ERR" "usage: sq comments add [issue-id] [text] [flags]" "missing issue id and text usage"

run_capture missing_text "$SQ_BIN" comments add "$ISSUE_ID"
assert_eq "$RUN_CODE" "2" "missing text"
assert_contains "$RUN_ERR" "usage: sq comments add [issue-id] [text] [flags]" "missing text usage"

run_capture missing_author "$SQ_BIN" comments add "$ISSUE_ID" --author
assert_eq "$RUN_CODE" "2" "missing author value"
assert_contains "$RUN_ERR" "missing value for --author" "missing author error"

run_capture missing_file_flag "$SQ_BIN" comments add "$ISSUE_ID" -f
assert_eq "$RUN_CODE" "2" "missing file value"
assert_contains "$RUN_ERR" "missing value for -f" "missing file error"

run_capture missing_file_path "$SQ_BIN" comments add "$ISSUE_ID" -f "$MISSING_FILE"
assert_eq "$RUN_CODE" "1" "missing file path should fail"
assert_contains "$RUN_ERR" "no such file or directory" "missing file path error"

run_capture bad_flag "$SQ_BIN" comments add "$ISSUE_ID" "bad flag" --wat
assert_eq "$RUN_CODE" "2" "unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "unknown flag error"

run_capture missing_actor "$SQ_BIN" comments add "$ISSUE_ID" --actor
assert_eq "$RUN_CODE" "2" "missing actor value"
assert_contains "$RUN_ERR" "missing value for --actor" "missing actor error"

run_capture missing_db "$SQ_BIN" comments add "$ISSUE_ID" --db
assert_eq "$RUN_CODE" "2" "missing db value"
assert_contains "$RUN_ERR" "missing value for --db" "missing db error"

run_capture missing_dolt "$SQ_BIN" comments add "$ISSUE_ID" --dolt-auto-commit
assert_eq "$RUN_CODE" "2" "missing dolt value"
assert_contains "$RUN_ERR" "missing value for --dolt-auto-commit" "missing dolt error"

run_capture missing_issue "$SQ_BIN" comments add bd-missing "orphan comment"
assert_eq "$RUN_CODE" "1" "missing issue target"
assert_contains "$RUN_ERR" "issue not found: bd-missing" "missing issue error"

run_capture final_list "$SQ_BIN" comments "$ISSUE_ID" --json
assert_eq "$RUN_CODE" "0" "final comments list"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "10" "final successful comment count"
assert_eq "$(json_eval "$RUN_OUT" 'obj[0]["body"]')" "hello from inline" "final inline body"
assert_eq "$(json_b64 "$RUN_OUT" 'obj[1]["body"]')" "Y29tbWVudCBmcm9tIGZpbGUKc2Vjb25kIGxpbmUK" "final file body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[2]["author"]')" "alice" "final author persisted"
assert_eq "$(json_eval "$RUN_OUT" 'obj[2]["body"]')" "authored comment" "final authored body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[3]["body"]')" "human output comment" "final human body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[4]["body"]')" "compat comment" "final compat actor body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[5]["body"]')" "quiet comment" "final quiet body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[6]["body"]')" "verbose comment" "final verbose body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[7]["body"]')" "sandbox comment" "final sandbox body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[8]["body"]')" "db comment" "final db body"
assert_eq "$(json_eval "$RUN_OUT" 'obj[9]["body"]')" "dolt comment" "final dolt body"

echo "[comments-add] PASS"
