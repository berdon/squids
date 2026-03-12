#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-show-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[show] FAIL: $*" >&2
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

echo "[show] binary=$SQ_BIN"
echo "[show] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
run_capture create_open "$SQ_BIN" create "show open task" --type task --priority 2 --json
OPEN_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture create_bug "$SQ_BIN" create "show bug" --type bug --priority 1 --description "bug description" --json
BUG_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture create_closed "$SQ_BIN" create "show closed task" --type task --json
CLOSED_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture close_closed "$SQ_BIN" close "$CLOSED_ID" --reason "setup" --json
assert_eq "$RUN_CODE" "0" "close closed task"
run_capture update_open "$SQ_BIN" update "$OPEN_ID" --assignee alice --add-label important --json
assert_eq "$RUN_CODE" "0" "update open task"

run_capture help_show "$SQ_BIN" help show
assert_eq "$RUN_CODE" "0" "help show"
assert_contains "$RUN_OUT" "show" "help show command name"
assert_contains "$RUN_OUT" "--json" "help show json flag"
assert_contains "$RUN_OUT" "Global Flags:" "help show global flags"

run_capture show_open_json "$SQ_BIN" show "$OPEN_ID" --json
assert_eq "$RUN_CODE" "0" "show open json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["id"]')" "$OPEN_ID" "show open id"
assert_eq "$(json_eval "$RUN_OUT" 'obj["title"]')" "show open task" "show open title"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "open" "show open status"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "task" "show open type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "2" "show open priority"
assert_contains "$RUN_OUT" '"created_at"' "show open created_at"
assert_contains "$RUN_OUT" '"updated_at"' "show open updated_at"

run_capture show_open_human "$SQ_BIN" show "$OPEN_ID"
assert_eq "$RUN_CODE" "0" "show open human"
assert_contains "$RUN_OUT" "$OPEN_ID" "human output id"
assert_contains "$RUN_OUT" "task" "human output type"
assert_contains "$RUN_OUT" "P2" "human output priority"
assert_contains "$RUN_OUT" "show open task" "human output title"

run_capture show_open_updated "$SQ_BIN" show "$OPEN_ID" --json
assert_eq "$RUN_CODE" "0" "show open updated"
assert_eq "$(json_eval "$RUN_OUT" 'obj["assignee"]')" "alice" "updated assignee"
assert_contains "$RUN_OUT" "important" "updated label"

run_capture show_bug "$SQ_BIN" show "$BUG_ID" --json
assert_eq "$RUN_CODE" "0" "show bug"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "bug" "show bug type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["description"]')" "bug description" "show bug description"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "1" "show bug priority"

run_capture show_closed "$SQ_BIN" show "$CLOSED_ID" --json
assert_eq "$RUN_CODE" "0" "show closed"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "closed" "show closed status"
assert_eq "$(json_eval "$RUN_OUT" 'obj["close_reason"]')" "setup" "show closed reason"

for flag in --quiet --verbose --sandbox --readonly --profile; do
  run_capture "compat_${flag//-/}" "$SQ_BIN" show "$OPEN_ID" --json "$flag"
  assert_eq "$RUN_CODE" "0" "show $flag"
  assert_eq "$(json_eval "$RUN_OUT" 'obj["id"]')" "$OPEN_ID" "show $flag id"
done
run_capture compat_actor "$SQ_BIN" show "$OPEN_ID" --actor tester --json
assert_eq "$RUN_CODE" "0" "show --actor compat"
run_capture compat_db "$SQ_BIN" show "$OPEN_ID" --db "$SQ_DB_PATH" --json
assert_eq "$RUN_CODE" "0" "show --db compat"
run_capture compat_dolt "$SQ_BIN" show "$OPEN_ID" --dolt-auto-commit off --json
assert_eq "$RUN_CODE" "0" "show --dolt-auto-commit compat"

run_capture missing_id "$SQ_BIN" show
assert_eq "$RUN_CODE" "2" "show missing id"
assert_contains "$RUN_ERR" "id is required" "show missing id error"
run_capture missing_issue "$SQ_BIN" show bd-missing --json
assert_eq "$RUN_CODE" "1" "show missing issue"
assert_contains "$RUN_ERR" "issue not found" "show missing issue error"
run_capture bad_flag "$SQ_BIN" show "$OPEN_ID" --wat
assert_eq "$RUN_CODE" "2" "show unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "show unknown flag error"
run_capture missing_actor "$SQ_BIN" show "$OPEN_ID" --actor
assert_eq "$RUN_CODE" "2" "show missing actor value"
assert_contains "$RUN_ERR" "missing value" "show missing actor error"
run_capture missing_db "$SQ_BIN" show "$OPEN_ID" --db
assert_eq "$RUN_CODE" "2" "show missing db value"
assert_contains "$RUN_ERR" "missing value" "show missing db error"
run_capture missing_dolt "$SQ_BIN" show "$OPEN_ID" --dolt-auto-commit
assert_eq "$RUN_CODE" "2" "show missing dolt value"
assert_contains "$RUN_ERR" "missing value" "show missing dolt error"
run_capture extra_positional "$SQ_BIN" show "$OPEN_ID" extra
assert_eq "$RUN_CODE" "2" "show extra positional"
assert_contains "$RUN_ERR" "unexpected positional" "show extra positional error"

run_capture list_all "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list all"
assert_contains "$RUN_OUT" "$OPEN_ID" "list includes open"
assert_contains "$RUN_OUT" "$BUG_ID" "list includes bug"
assert_contains "$RUN_OUT" "$CLOSED_ID" "list includes closed"

echo "[show] PASS"
