#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-todo-add-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[todo-add] FAIL: $*" >&2
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

echo "[todo-add] workspace=$WORKSPACE"
echo "[todo-add] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
assert_eq "$(json_eval "$RUN_OUT" 'obj["count"]')" "0" "initial count"

run_capture help_cmd "$SQ_BIN" help todo add
assert_eq "$RUN_CODE" "0" "sq help todo add"
assert_contains "$RUN_OUT" "sq todo add <title>" "help command usage"
assert_contains "$RUN_OUT" "--priority" "help command priority flag"
assert_contains "$RUN_OUT" "--description" "help command description flag"
HELP_CMD_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" todo add --help
assert_eq "$RUN_CODE" "0" "sq todo add --help"
assert_contains "$RUN_OUT" "sq todo add <title>" "help flag usage"
assert_contains "$RUN_OUT" "--priority" "help flag priority flag"
assert_contains "$RUN_OUT" "--description" "help flag description flag"
assert_eq "$RUN_OUT" "$HELP_CMD_OUT" "help command and --help parity"

run_capture noarg "$SQ_BIN" todo add
assert_eq "$RUN_CODE" "2" "todo add missing title should fail"
assert_contains "$RUN_ERR" "usage: sq todo add <title>" "missing title usage"

run_capture bogus "$SQ_BIN" todo add --bogus
assert_eq "$RUN_CODE" "2" "todo add unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --bogus" "unknown flag error"

run_capture default_add "$SQ_BIN" todo add "First todo"
assert_eq "$RUN_CODE" "0" "todo add default"
assert_contains "$RUN_OUT" "Added todo" "default output should be human readable"
assert_contains "$RUN_OUT" "First todo" "default output should mention title"
assert_not_contains "$RUN_OUT" '"id"' "default output should not be raw JSON"

run_capture list_after_default "$SQ_BIN" todo --json
assert_eq "$RUN_CODE" "0" "todo list after default add"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "1" "todo list count after default add"
FIRST_ID="$(json_eval "$RUN_OUT" 'obj[0]["id"]')"
assert_eq "$(json_eval "$RUN_OUT" 'obj[0]["title"]')" "First todo" "default add title persisted"
assert_eq "$(json_eval "$RUN_OUT" 'obj[0]["priority"]')" "2" "default add priority persisted"

run_capture json_add "$SQ_BIN" todo add "Second todo" --priority 1 --description "important follow-up" --json
assert_eq "$RUN_CODE" "0" "todo add json"
SECOND_ID="$(json_id "$RUN_OUT")"
assert_eq "$(json_eval "$RUN_OUT" 'obj["title"]')" "Second todo" "json add title"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "1" "json add priority"
assert_eq "$(json_eval "$RUN_OUT" 'obj["description"]')" "important follow-up" "json add description"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "task" "json add type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "open" "json add status"

run_capture show_second "$SQ_BIN" show "$SECOND_ID" --json
assert_eq "$RUN_CODE" "0" "show second todo"
assert_eq "$(json_eval "$RUN_OUT" 'obj["title"]')" "Second todo" "show second title"
assert_eq "$(json_eval "$RUN_OUT" 'obj["description"]')" "important follow-up" "show second description"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "1" "show second priority"

run_capture todo_list "$SQ_BIN" todo --json
assert_eq "$RUN_CODE" "0" "todo list json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "2" "todo list total"
assert_contains "$RUN_OUT" "$FIRST_ID" "todo list contains first id"
assert_contains "$RUN_OUT" "$SECOND_ID" "todo list contains second id"

run_capture count_after_success "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after success paths"
assert_eq "$(json_eval "$RUN_OUT" 'obj["count"]')" "2" "count after success paths"

run_capture actor_json "$SQ_BIN" todo add "Actor todo" --json --actor tester
assert_eq "$RUN_CODE" "0" "todo add --actor"
THIRD_ID="$(json_id "$RUN_OUT")"

run_capture readonly_json "$SQ_BIN" todo add "Readonly todo" --json --readonly
assert_eq "$RUN_CODE" "0" "todo add --readonly"
FOURTH_ID="$(json_id "$RUN_OUT")"

run_capture sandbox_json "$SQ_BIN" todo add "Sandbox todo" --json --sandbox
assert_eq "$RUN_CODE" "0" "todo add --sandbox"
FIFTH_ID="$(json_id "$RUN_OUT")"

run_capture profile_json "$SQ_BIN" todo add "Profile todo" --json --profile
assert_eq "$RUN_CODE" "0" "todo add --profile"
SIXTH_ID="$(json_id "$RUN_OUT")"

run_capture quiet_json "$SQ_BIN" todo add "Quiet todo" --json --quiet
assert_eq "$RUN_CODE" "0" "todo add --quiet"
SEVENTH_ID="$(json_id "$RUN_OUT")"

run_capture verbose_json "$SQ_BIN" todo add "Verbose todo" --json --verbose
assert_eq "$RUN_CODE" "0" "todo add --verbose"
EIGHTH_ID="$(json_id "$RUN_OUT")"

run_capture dolt_json "$SQ_BIN" todo add "Dolt todo" --json --dolt-auto-commit off
assert_eq "$RUN_CODE" "0" "todo add --dolt-auto-commit"
NINTH_ID="$(json_id "$RUN_OUT")"

run_capture db_json "$SQ_BIN" todo add "DB todo" --json --db "$SQ_DB_PATH"
assert_eq "$RUN_CODE" "0" "todo add --db"
TENTH_ID="$(json_id "$RUN_OUT")"

run_capture count_after_flags "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after flag coverage"
assert_eq "$(json_eval "$RUN_OUT" 'obj["count"]')" "10" "count after all successful adds"

run_capture list_final "$SQ_BIN" todo --json
assert_eq "$RUN_CODE" "0" "final todo list"
for id in "$FIRST_ID" "$SECOND_ID" "$THIRD_ID" "$FOURTH_ID" "$FIFTH_ID" "$SIXTH_ID" "$SEVENTH_ID" "$EIGHTH_ID" "$NINTH_ID" "$TENTH_ID"; do
  assert_contains "$RUN_OUT" "$id" "final list should contain $id"
done

echo "[todo-add] PASS"
