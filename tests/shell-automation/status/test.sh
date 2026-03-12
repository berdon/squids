#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  echo "[status] sq binary not found at $SQ_BIN (run make build first)" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d -t sq-status-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[status] FAIL: $*" >&2
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

WORKSPACE="$TMP_DIR/workspace"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"
export SQ_DB_PATH="$WORKSPACE/tasks.sqlite"

echo "[status] workspace=$WORKSPACE"
echo "[status] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture help_cmd "$SQ_BIN" help status
assert_eq "$RUN_CODE" "0" "sq help status"
assert_contains "$RUN_OUT" "Show a quick snapshot of the issue database state and statistics." "help command description"
assert_contains "$RUN_OUT" "sq status [flags]" "help command usage"
assert_contains "$RUN_OUT" "--assigned" "help command assigned flag"
HELP_CMD_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" status --help
assert_eq "$RUN_CODE" "0" "sq status --help"
assert_contains "$RUN_OUT" "Show a quick snapshot of the issue database state and statistics." "help flag description"
assert_contains "$RUN_OUT" "sq status [flags]" "help flag usage"
assert_contains "$RUN_OUT" "--assigned" "help flag assigned flag"
assert_eq "$RUN_OUT" "$HELP_CMD_OUT" "help command and --help parity"

run_capture empty_default "$SQ_BIN" status
assert_eq "$RUN_CODE" "0" "empty status default"
assert_contains "$RUN_OUT" "Issue Database Status" "empty default heading"
assert_contains "$RUN_OUT" "Total Issues:           0" "empty default total"
assert_contains "$RUN_OUT" "Ready to Work:          0" "empty default ready"

run_capture empty_json "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "empty status json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["summary"]["total_issues"]')" "0" "empty total issues"
assert_eq "$(json_eval "$RUN_OUT" 'obj["open"]')" "0" "empty open count"
assert_eq "$(json_eval "$RUN_OUT" 'obj["ready"]')" "0" "empty ready count"

run_capture create_open "$SQ_BIN" create "Open task" --type task --priority 1 --json
assert_eq "$RUN_CODE" "0" "create open task"
OPEN_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"

run_capture create_assigned_open "$SQ_BIN" create "Assigned open task" --type bug --priority 2 --json
assert_eq "$RUN_CODE" "0" "create assigned open task"
ASSIGNED_OPEN_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture assign_open "$SQ_BIN" update "$ASSIGNED_OPEN_ID" --assignee tester --json
assert_eq "$RUN_CODE" "0" "assign open task"

run_capture create_assigned_wip "$SQ_BIN" create "Assigned in progress task" --type feature --priority 1 --json
assert_eq "$RUN_CODE" "0" "create assigned wip task"
ASSIGNED_WIP_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture assign_wip "$SQ_BIN" update "$ASSIGNED_WIP_ID" --assignee tester --status in_progress --json
assert_eq "$RUN_CODE" "0" "assign wip task"

run_capture create_closed "$SQ_BIN" create "Closed task" --type chore --priority 3 --json
assert_eq "$RUN_CODE" "0" "create closed task"
CLOSED_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture close_closed "$SQ_BIN" close "$CLOSED_ID" --reason "test setup" --json
assert_eq "$RUN_CODE" "0" "close task"

run_capture seeded_default "$SQ_BIN" status
assert_eq "$RUN_CODE" "0" "seeded status default"
assert_contains "$RUN_OUT" "Total Issues:           4" "seeded default total"
assert_contains "$RUN_OUT" "Open:                   2" "seeded default open"
assert_contains "$RUN_OUT" "In Progress:            1" "seeded default in progress"
assert_contains "$RUN_OUT" "Closed:                 1" "seeded default closed"
assert_contains "$RUN_OUT" "Ready to Work:          2" "seeded default ready"

run_capture seeded_json "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "seeded status json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["summary"]["total_issues"]')" "4" "seeded total issues"
assert_eq "$(json_eval "$RUN_OUT" 'obj["open"]')" "2" "seeded open"
assert_eq "$(json_eval "$RUN_OUT" 'obj["in_progress"]')" "1" "seeded in progress"
assert_eq "$(json_eval "$RUN_OUT" 'obj["closed"]')" "1" "seeded closed"
assert_eq "$(json_eval "$RUN_OUT" 'obj["ready"]')" "2" "seeded ready"
BASE_JSON="$RUN_OUT"

run_capture assigned_json env SQ_ACTOR=tester SQ_DB_PATH="$SQ_DB_PATH" "$SQ_BIN" status --assigned --json
assert_eq "$RUN_CODE" "0" "status --assigned --json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["summary"]["total_issues"]')" "2" "assigned total issues"
assert_eq "$(json_eval "$RUN_OUT" 'obj["open"]')" "1" "assigned open"
assert_eq "$(json_eval "$RUN_OUT" 'obj["in_progress"]')" "1" "assigned in progress"
assert_eq "$(json_eval "$RUN_OUT" 'obj["closed"]')" "0" "assigned closed"
assert_eq "$(json_eval "$RUN_OUT" 'obj["ready"]')" "1" "assigned ready"

run_capture all_json "$SQ_BIN" status --all --json
assert_eq "$RUN_CODE" "0" "status --all --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--all should match default summary"

run_capture no_activity_json "$SQ_BIN" status --no-activity --json
assert_eq "$RUN_CODE" "0" "status --no-activity --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--no-activity should preserve summary"

run_capture actor_json "$SQ_BIN" status --json --actor tester
assert_eq "$RUN_CODE" "0" "status --actor --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--actor should not change unfiltered summary"

run_capture readonly_json "$SQ_BIN" status --json --readonly
assert_eq "$RUN_CODE" "0" "status --readonly --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--readonly should not change summary"

run_capture sandbox_json "$SQ_BIN" status --json --sandbox
assert_eq "$RUN_CODE" "0" "status --sandbox --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--sandbox should not change summary"

run_capture profile_json "$SQ_BIN" status --json --profile
assert_eq "$RUN_CODE" "0" "status --profile --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--profile should not change summary"

run_capture quiet_json "$SQ_BIN" status --json --quiet
assert_eq "$RUN_CODE" "0" "status --quiet --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--quiet should not change summary"

run_capture verbose_json "$SQ_BIN" status --json --verbose
assert_eq "$RUN_CODE" "0" "status --verbose --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--verbose should not change summary"

run_capture dolt_json "$SQ_BIN" status --json --dolt-auto-commit off
assert_eq "$RUN_CODE" "0" "status --dolt-auto-commit --json"
assert_eq "$RUN_OUT" "$BASE_JSON" "--dolt-auto-commit should not change summary"

run_capture bogus "$SQ_BIN" status --bogus-flag
assert_eq "$RUN_CODE" "2" "unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --bogus-flag" "unknown flag error"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after status checks"
assert_eq "$(json_eval "$RUN_OUT" 'obj["count"]')" "4" "status should not mutate tasks"

echo "[status] PASS"
