#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-list-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[list] FAIL: $*" >&2
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

DB1="$TMP_DIR/tasks.sqlite"
export SQ_DB_PATH="$DB1"

echo "[list] binary=$SQ_BIN"
echo "[list] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
run_capture create_a "$SQ_BIN" create "List smoke A" --type task --priority 1 --json
A_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture create_b "$SQ_BIN" create "List smoke B" --type bug --priority 2 --json
B_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture create_c "$SQ_BIN" create "List smoke C" --type feature --priority 3 --json
C_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture update_b "$SQ_BIN" update "$B_ID" --status in_progress --json
assert_eq "$RUN_CODE" "0" "update B"
run_capture close_c "$SQ_BIN" close "$C_ID" --reason "Done" --json
assert_eq "$RUN_CODE" "0" "close C"
run_capture show_a "$SQ_BIN" show "$A_ID" --json
assert_eq "$RUN_CODE" "0" "show A"

run_capture basic "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list basic"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "3" "list count"
assert_contains "$RUN_OUT" "$A_ID" "list includes A"
assert_contains "$RUN_OUT" "$B_ID" "list includes B"
assert_contains "$RUN_OUT" "$C_ID" "list includes C"
assert_eq "$(json_eval "$RUN_OUT" 'obj[0]["id"]')" "$A_ID" "ordering observation A first"
assert_eq "$(json_eval "$RUN_OUT" 'obj[1]["id"]')" "$B_ID" "ordering observation B second"
assert_eq "$(json_eval "$RUN_OUT" 'obj[2]["id"]')" "$C_ID" "ordering observation C third"

DB2="$TMP_DIR/empty.sqlite"
export SQ_DB_PATH="$DB2"
run_capture init_empty "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "init empty db"
run_capture list_empty "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list empty"
assert_eq "$RUN_OUT" "[]" "empty list"

export SQ_DB_PATH="$DB1"
run_capture compat "$SQ_BIN" list --json --flat --no-pager
assert_eq "$RUN_CODE" "0" "list compat flags"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "3" "compat list count"
assert_contains "$RUN_OUT" "$A_ID" "compat includes A"

run_capture show_from_list "$SQ_BIN" show "$B_ID" --json
assert_eq "$RUN_CODE" "0" "show id from list"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "in_progress" "show B status"
run_capture update_a "$SQ_BIN" update "$A_ID" --status in_progress --json
assert_eq "$RUN_CODE" "0" "update A in progress"
run_capture list_after_update "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list after update"
assert_contains "$RUN_OUT" '"status": "in_progress"' "list reflects update"
run_capture close_a "$SQ_BIN" close "$A_ID" --reason "Done" --json
assert_eq "$RUN_CODE" "0" "close A"
run_capture list_after_close "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list after close"
assert_contains "$RUN_OUT" '"status": "closed"' "list reflects close"

run_capture help_flag "$SQ_BIN" list --help
assert_eq "$RUN_CODE" "2" "list --help should fail cleanly"
assert_contains "$RUN_ERR" "unknown flag: --help" "list --help error"
run_capture bad_flag "$SQ_BIN" list --wat
assert_eq "$RUN_CODE" "2" "list bad flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "list bad flag error"
run_capture unsupported "$SQ_BIN" list --status open
assert_eq "$RUN_CODE" "2" "list unsupported flag"
assert_contains "$RUN_ERR" "unknown flag: --status" "list unsupported flag error"

run_capture before_repeat "$SQ_BIN" list --json
BEFORE_REPEAT="$RUN_OUT"
run_capture repeat1 "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "repeat list 1"
run_capture repeat2 "$SQ_BIN" list --json --flat --no-pager
assert_eq "$RUN_CODE" "0" "repeat list 2"
run_capture after_repeat "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "repeat list after"
assert_eq "$RUN_OUT" "$BEFORE_REPEAT" "list should be read-only"

DB3="$TMP_DIR/alt.sqlite"
export SQ_DB_PATH="$DB3"
run_capture init_alt "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "init alt db"
run_capture create_alt "$SQ_BIN" create "Alt DB task" --json
assert_eq "$RUN_CODE" "0" "create alt task"
ALT_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture list_alt "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list alt db"
assert_contains "$RUN_OUT" "$ALT_ID" "alt db includes alt task"
if [[ "$RUN_OUT" == *"$A_ID"* ]]; then
  fail "alt db list should not include tasks from primary db"
fi

unset SQ_DB_PATH || true

echo "[list] PASS"
