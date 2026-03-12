#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-todo-done-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[todo-done] FAIL: $*" >&2
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
    fail "$context: expected '$needle' in '$haystack'"
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local context="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    fail "$context: did not expect '$needle' in '$haystack'"
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

json_id() {
  local payload="$1"
  JSON_INPUT="$payload" "$PYTHON" <<'PY'
import json, os
print(json.loads(os.environ["JSON_INPUT"])["id"])
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

WORKSPACE="$TMP_DIR/workspace"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"
export SQ_DB_PATH="$WORKSPACE/tasks.sqlite"

echo "[todo-done] workspace=$WORKSPACE"
echo "[todo-done] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture help_cmd "$SQ_BIN" help todo done
assert_eq "$RUN_CODE" "0" "sq help todo done"
assert_contains "$RUN_OUT" "sq todo done <id>" "help command usage"
assert_contains "$RUN_OUT" "--reason" "help command reason flag"
assert_contains "$RUN_OUT" "--json" "help command json flag"
HELP_CMD_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" todo done --help
assert_eq "$RUN_CODE" "0" "sq todo done --help"
assert_contains "$RUN_OUT" "sq todo done <id>" "help flag usage"
assert_contains "$RUN_OUT" "--reason" "help flag reason flag"
assert_contains "$RUN_OUT" "--json" "help flag json flag"
assert_eq "$RUN_OUT" "$HELP_CMD_OUT" "help command and --help parity"

run_capture noarg "$SQ_BIN" todo done
assert_eq "$RUN_CODE" "2" "todo done missing id should fail"
assert_contains "$RUN_ERR" "usage: sq todo done <id>" "missing id usage"

run_capture bogus "$SQ_BIN" todo done --bogus
assert_eq "$RUN_CODE" "2" "todo done unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --bogus" "unknown flag error"

run_capture missing_id "$SQ_BIN" todo done bd-does-not-exist
assert_eq "$RUN_CODE" "1" "todo done missing issue id should fail"
assert_contains "$RUN_ERR" "issue not found: bd-does-not-exist" "missing issue error"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
assert_eq "$(json_eval "$RUN_OUT" 'obj["count"]')" "0" "initial count"

run_capture create_default "$SQ_BIN" todo add "Todo default" --json
assert_eq "$RUN_CODE" "0" "create default todo"
DEFAULT_ID="$(json_id "$RUN_OUT")"

run_capture done_default "$SQ_BIN" todo done "$DEFAULT_ID"
assert_eq "$RUN_CODE" "0" "todo done default"
assert_contains "$RUN_OUT" "Completed todo" "default output should be human readable"
assert_contains "$RUN_OUT" "$DEFAULT_ID" "default output should mention id"
assert_not_contains "$RUN_OUT" '"id"' "default output should not be raw JSON"

run_capture show_default "$SQ_BIN" show "$DEFAULT_ID" --json
assert_eq "$RUN_CODE" "0" "show default closed todo"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "closed" "default todo status closed"
assert_eq "$(json_eval "$RUN_OUT" 'obj["close_reason"]')" "Completed" "default todo close reason"

run_capture create_json "$SQ_BIN" todo add "Todo json" --json
assert_eq "$RUN_CODE" "0" "create json todo"
JSON_ID="$(json_id "$RUN_OUT")"

run_capture done_json "$SQ_BIN" todo done "$JSON_ID" --reason "finished in test" --json
assert_eq "$RUN_CODE" "0" "todo done json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["id"]')" "$JSON_ID" "json closed id"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "closed" "json closed status"
assert_eq "$(json_eval "$RUN_OUT" 'obj["close_reason"]')" "finished in test" "json close reason"

run_capture show_json "$SQ_BIN" show "$JSON_ID" --json
assert_eq "$RUN_CODE" "0" "show json closed todo"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "closed" "show json status"
assert_eq "$(json_eval "$RUN_OUT" 'obj["close_reason"]')" "finished in test" "show json reason"

run_capture actor_json "$SQ_BIN" todo add "Actor todo" --json
THIRD_ID="$(json_id "$RUN_OUT")"
run_capture done_actor "$SQ_BIN" todo done "$THIRD_ID" --json --actor tester
assert_eq "$RUN_CODE" "0" "todo done --actor"

run_capture readonly_create "$SQ_BIN" todo add "Readonly todo" --json
FOURTH_ID="$(json_id "$RUN_OUT")"
run_capture done_readonly "$SQ_BIN" todo done "$FOURTH_ID" --json --readonly
assert_eq "$RUN_CODE" "0" "todo done --readonly"

run_capture sandbox_create "$SQ_BIN" todo add "Sandbox todo" --json
FIFTH_ID="$(json_id "$RUN_OUT")"
run_capture done_sandbox "$SQ_BIN" todo done "$FIFTH_ID" --json --sandbox
assert_eq "$RUN_CODE" "0" "todo done --sandbox"

run_capture profile_create "$SQ_BIN" todo add "Profile todo" --json
SIXTH_ID="$(json_id "$RUN_OUT")"
run_capture done_profile "$SQ_BIN" todo done "$SIXTH_ID" --json --profile
assert_eq "$RUN_CODE" "0" "todo done --profile"

run_capture quiet_create "$SQ_BIN" todo add "Quiet todo" --json
SEVENTH_ID="$(json_id "$RUN_OUT")"
run_capture done_quiet "$SQ_BIN" todo done "$SEVENTH_ID" --json --quiet
assert_eq "$RUN_CODE" "0" "todo done --quiet"

run_capture verbose_create "$SQ_BIN" todo add "Verbose todo" --json
EIGHTH_ID="$(json_id "$RUN_OUT")"
run_capture done_verbose "$SQ_BIN" todo done "$EIGHTH_ID" --json --verbose
assert_eq "$RUN_CODE" "0" "todo done --verbose"

run_capture dolt_create "$SQ_BIN" todo add "Dolt todo" --json
NINTH_ID="$(json_id "$RUN_OUT")"
run_capture done_dolt "$SQ_BIN" todo done "$NINTH_ID" --json --dolt-auto-commit off
assert_eq "$RUN_CODE" "0" "todo done --dolt-auto-commit"

run_capture db_create "$SQ_BIN" todo add "DB todo" --json
TENTH_ID="$(json_id "$RUN_OUT")"
run_capture done_db "$SQ_BIN" todo done "$TENTH_ID" --json --db "$SQ_DB_PATH"
assert_eq "$RUN_CODE" "0" "todo done --db"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after"
assert_eq "$(json_eval "$RUN_OUT" 'obj["count"]')" "10" "count should remain 10 after closing todos"

run_capture todo_list "$SQ_BIN" todo --json
assert_eq "$RUN_CODE" "0" "todo list final"
assert_eq "$(json_eval "$RUN_OUT" 'len([item for item in obj if item["status"] == "closed"])')" "10" "all todos should be closed"

echo "[todo-done] PASS"
