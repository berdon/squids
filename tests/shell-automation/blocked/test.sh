#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-blocked-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[blocked] FAIL: $*" >&2
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
  local json_input="$1"
  local expr="$2"
  JSON_INPUT="$json_input" "$PYTHON" - "$expr" <<'PY'
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
  local json_input="$1"
  JSON_INPUT="$json_input" "$PYTHON" <<'PY'
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

echo "[blocked] workspace=$WORKSPACE"
echo "[blocked] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"


run_capture help_cmd "$SQ_BIN" help blocked
assert_eq "$RUN_CODE" "0" "sq help blocked"
assert_contains "$RUN_OUT" "Show blocked issues" "help command description"
assert_contains "$RUN_OUT" "sq blocked [flags]" "help command usage"
assert_contains "$RUN_OUT" "--parent string" "help command parent flag"

run_capture help_flag "$SQ_BIN" blocked --help
assert_eq "$RUN_CODE" "0" "sq blocked --help"
assert_contains "$RUN_OUT" "Show blocked issues" "help flag description"
assert_contains "$RUN_OUT" "sq blocked [flags]" "help flag usage"
assert_contains "$RUN_OUT" "--parent string" "help flag parent flag"

run_capture empty_default "$SQ_BIN" blocked
assert_eq "$RUN_CODE" "0" "blocked empty default"
assert_contains "$RUN_OUT" "No blocked issues found" "empty default output"

run_capture empty_json "$SQ_BIN" blocked --json
assert_eq "$RUN_CODE" "0" "blocked empty json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "0" "empty json list"

run_capture create_parent "$SQ_BIN" create "Parent" --type epic --priority 1 --json
PARENT_ID="$(json_id "$RUN_OUT")"
run_capture create_child "$SQ_BIN" create "Child" --type task --priority 2 --deps "parent-child:$PARENT_ID" --json
CHILD_ID="$(json_id "$RUN_OUT")"
run_capture create_grandchild "$SQ_BIN" create "Grandchild" --type task --priority 2 --deps "parent-child:$CHILD_ID" --json
GRANDCHILD_ID="$(json_id "$RUN_OUT")"
run_capture create_blocker "$SQ_BIN" create "Blocker" --type bug --priority 1 --json
BLOCKER_ID="$(json_id "$RUN_OUT")"
run_capture create_other "$SQ_BIN" create "Other blocked" --type task --priority 2 --json
OTHER_ID="$(json_id "$RUN_OUT")"
run_capture create_other_blocker "$SQ_BIN" create "Other blocker" --type bug --priority 1 --json
OTHER_BLOCKER_ID="$(json_id "$RUN_OUT")"

run_capture dep_child "$SQ_BIN" dep add "$BLOCKER_ID" "$CHILD_ID" --json
assert_eq "$RUN_CODE" "0" "dep add child"
run_capture dep_grandchild "$SQ_BIN" dep add "$BLOCKER_ID" "$GRANDCHILD_ID" --json
assert_eq "$RUN_CODE" "0" "dep add grandchild"
run_capture dep_other "$SQ_BIN" dep add "$OTHER_BLOCKER_ID" "$OTHER_ID" --json
assert_eq "$RUN_CODE" "0" "dep add other"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
COUNT_BEFORE="$RUN_OUT"

run_capture status_before "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status before"
STATUS_BEFORE="$RUN_OUT"

run_capture default_output "$SQ_BIN" blocked
assert_eq "$RUN_CODE" "0" "blocked default output"
assert_contains "$RUN_OUT" "Found 3 blocked issue(s):" "default blocked header"
assert_contains "$RUN_OUT" "$CHILD_ID" "default output child id"
assert_contains "$RUN_OUT" "$GRANDCHILD_ID" "default output grandchild id"
assert_contains "$RUN_OUT" "$OTHER_ID" "default output other id"
assert_contains "$RUN_OUT" "$BLOCKER_ID" "default output blocker id"
assert_not_contains "$RUN_OUT" '"id"' "default output should be human-readable"

run_capture json_output "$SQ_BIN" blocked --json
assert_eq "$RUN_CODE" "0" "blocked json output"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "3" "blocked json count"
assert_eq "$(json_eval "$RUN_OUT" 'len([item for item in obj if item["id"]])')" "3" "blocked json ids present"
assert_eq "$(json_eval "$RUN_OUT" 'len([item for item in obj if item["blocked_by_count"] == 1])')" "3" "blocked_by_count present"
assert_contains "$RUN_OUT" "$CHILD_ID" "json child id"
assert_contains "$RUN_OUT" "$GRANDCHILD_ID" "json grandchild id"
assert_contains "$RUN_OUT" "$OTHER_ID" "json other id"

run_capture parent_filter "$SQ_BIN" blocked --parent "$PARENT_ID" --json
assert_eq "$RUN_CODE" "0" "blocked parent filter"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "2" "parent filter count"
assert_contains "$RUN_OUT" "$CHILD_ID" "parent filter child"
assert_contains "$RUN_OUT" "$GRANDCHILD_ID" "parent filter grandchild"
assert_not_contains "$RUN_OUT" "$OTHER_ID" "parent filter excludes non-descendant"

run_capture bad_flag "$SQ_BIN" blocked --wat
assert_eq "$RUN_CODE" "2" "blocked unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "unknown flag error"

run_capture positional "$SQ_BIN" blocked stray
assert_eq "$RUN_CODE" "2" "blocked positional arg should fail"
assert_contains "$RUN_ERR" "blocked does not accept positional arguments" "positional arg error"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after"
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "blocked should not mutate count summary"

run_capture status_after "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status after"
assert_eq "$RUN_OUT" "$STATUS_BEFORE" "blocked should not mutate status summary"

echo "[blocked] PASS"
