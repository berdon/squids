#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-gate-show-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[gate-show] FAIL: $*" >&2
  exit 1
}

assert_eq() {
  local got="$1" want="$2" context="$3"
  if [[ "$got" != "$want" ]]; then
    fail "$context: expected '$want', got '$got'"
  fi
}

assert_contains() {
  local haystack="$1" needle="$2" context="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    fail "$context: expected '$needle' in '$haystack'"
  fi
}

assert_not_contains() {
  local haystack="$1" needle="$2" context="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    fail "$context: did not expect '$needle' in '$haystack'"
  fi
}

json_eval() {
  local json_input="$1" expr="$2"
  JSON_INPUT="$json_input" "$PYTHON" - "$expr" <<'PY'
import json, os, sys
obj = json.loads(os.environ['JSON_INPUT'])
expr = sys.argv[1]
value = eval(expr, {'__builtins__': {}}, {'obj': obj, 'len': len})
if isinstance(value, (dict, list)):
    print(json.dumps(value))
elif isinstance(value, bool):
    print('true' if value else 'false')
else:
    print(value)
PY
}

json_id() {
  local json_input="$1"
  JSON_INPUT="$json_input" "$PYTHON" <<'PY'
import json, os
print(json.loads(os.environ['JSON_INPUT'])['id'])
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

echo "[gate-show] workspace=$WORKSPACE"
echo "[gate-show] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture help_cmd "$SQ_BIN" help gate show
assert_eq "$RUN_CODE" "0" "sq help gate show"
assert_contains "$RUN_OUT" "Show a single gate issue." "help command description"
assert_contains "$RUN_OUT" "sq gate show <id> [flags]" "help command usage"
assert_contains "$RUN_OUT" "--json" "help command json flag"
HELP_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" gate show --help
assert_eq "$RUN_CODE" "0" "sq gate show --help"
assert_eq "$RUN_OUT" "$HELP_OUT" "help parity"

run_capture missing "$SQ_BIN" gate show
assert_eq "$RUN_CODE" "2" "gate show missing id"
assert_contains "$RUN_ERR" "usage: sq gate show <id> [--json]" "missing id usage"

run_capture create_gate "$SQ_BIN" create "GateAlpha" --type gate --json
GATE_ID="$(json_id "$RUN_OUT")"
run_capture create_task "$SQ_BIN" create "TaskAlpha" --type task --json
TASK_ID="$(json_id "$RUN_OUT")"

run_capture count_before "$SQ_BIN" count --json
COUNT_BEFORE="$RUN_OUT"
run_capture status_before "$SQ_BIN" status --json
STATUS_BEFORE="$RUN_OUT"

run_capture default_output "$SQ_BIN" gate show "$GATE_ID"
assert_eq "$RUN_CODE" "0" "gate show default"
assert_contains "$RUN_OUT" "Gate $GATE_ID" "default gate id"
assert_contains "$RUN_OUT" "status: open" "default status"
assert_contains "$RUN_OUT" "title: GateAlpha" "default title"
assert_not_contains "$RUN_OUT" '"id"' "default should be human-readable"

run_capture json_output "$SQ_BIN" gate show "$GATE_ID" --json
assert_eq "$RUN_CODE" "0" "gate show json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["id"]')" "$GATE_ID" "json id"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "gate" "json type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "open" "json status"

run_capture nongate "$SQ_BIN" gate show "$TASK_ID" --json
assert_eq "$RUN_CODE" "2" "gate show non-gate should fail"
assert_contains "$RUN_ERR" "issue is not a gate: $TASK_ID" "non-gate error"

run_capture missing_id "$SQ_BIN" gate show bd-missing --json
assert_eq "$RUN_CODE" "1" "gate show missing issue runtime failure"
assert_contains "$RUN_ERR" "issue not found: bd-missing" "missing issue error"

run_capture positional "$SQ_BIN" gate show "$GATE_ID" stray
assert_eq "$RUN_CODE" "2" "gate show extra positional should fail"
assert_contains "$RUN_ERR" "gate show accepts exactly one positional argument" "extra positional error"

run_capture bad_flag "$SQ_BIN" gate show "$GATE_ID" --wat
assert_eq "$RUN_CODE" "2" "gate show unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --wat" "unknown flag error"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "gate show should not mutate count"
run_capture status_after "$SQ_BIN" status --json
assert_eq "$RUN_OUT" "$STATUS_BEFORE" "gate show should not mutate status"

echo "[gate-show] PASS"
