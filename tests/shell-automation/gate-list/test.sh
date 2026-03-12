#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-gate-list-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[gate-list] FAIL: $*" >&2
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

echo "[gate-list] workspace=$WORKSPACE"
echo "[gate-list] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"


run_capture help_cmd "$SQ_BIN" help gate list
assert_eq "$RUN_CODE" "0" "sq help gate list"
assert_contains "$RUN_OUT" "List gate issues." "help command description"
assert_contains "$RUN_OUT" "sq gate list [flags]" "help command usage"
assert_contains "$RUN_OUT" "--all" "help command all flag"
assert_contains "$RUN_OUT" "--json" "help command json flag"
HELP_CMD_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" gate list --help
assert_eq "$RUN_CODE" "0" "sq gate list --help"
assert_contains "$RUN_OUT" "List gate issues." "help flag description"
assert_contains "$RUN_OUT" "sq gate list [flags]" "help flag usage"
assert_eq "$RUN_OUT" "$HELP_CMD_OUT" "help command and --help parity"

run_capture empty_default "$SQ_BIN" gate list
assert_eq "$RUN_CODE" "0" "gate list empty default"
assert_contains "$RUN_OUT" "Found 0 gates:" "empty default output"

run_capture empty_json "$SQ_BIN" gate list --json
assert_eq "$RUN_CODE" "0" "gate list empty json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "0" "empty json count"

run_capture create_gate_one "$SQ_BIN" create "Gate One" --type gate --json
GATE_ONE_ID="$(json_id "$RUN_OUT")"
run_capture create_gate_two "$SQ_BIN" create "Gate Two" --type gate --json
GATE_TWO_ID="$(json_id "$RUN_OUT")"
run_capture create_normal "$SQ_BIN" create "Normal Task" --type task --json
NORMAL_ID="$(json_id "$RUN_OUT")"
run_capture close_gate_two "$SQ_BIN" close "$GATE_TWO_ID" --reason done --json
assert_eq "$RUN_CODE" "0" "close second gate"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
COUNT_BEFORE="$RUN_OUT"

run_capture status_before "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status before"
STATUS_BEFORE="$RUN_OUT"

run_capture default_output "$SQ_BIN" gate list
assert_eq "$RUN_CODE" "0" "gate list default output"
assert_contains "$RUN_OUT" "Found 1 gates:" "default count"
assert_contains "$RUN_OUT" "$GATE_ONE_ID" "default includes open gate"
assert_not_contains "$RUN_OUT" "$GATE_TWO_ID" "default excludes closed gate"
assert_not_contains "$RUN_OUT" "$NORMAL_ID" "default excludes non-gate"
assert_not_contains "$RUN_OUT" '"id"' "default should be human-readable"

run_capture json_output "$SQ_BIN" gate list --json
assert_eq "$RUN_CODE" "0" "gate list json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "1" "json open gate count"
assert_contains "$RUN_OUT" "$GATE_ONE_ID" "json includes open gate"
assert_not_contains "$RUN_OUT" "$GATE_TWO_ID" "json excludes closed gate without --all"

run_capture all_json "$SQ_BIN" gate list --all --json
assert_eq "$RUN_CODE" "0" "gate list --all --json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "2" "all json count"
assert_contains "$RUN_OUT" "$GATE_ONE_ID" "all json includes open gate"
assert_contains "$RUN_OUT" "$GATE_TWO_ID" "all json includes closed gate"

run_capture all_human "$SQ_BIN" gate list --all
assert_eq "$RUN_CODE" "0" "gate list --all human"
assert_contains "$RUN_OUT" "$GATE_ONE_ID" "all human includes open gate"
assert_contains "$RUN_OUT" "$GATE_TWO_ID" "all human includes closed gate"
assert_contains "$RUN_OUT" "✓ $GATE_TWO_ID [closed] - Gate Two" "all human closed icon"

run_capture positional "$SQ_BIN" gate list stray
assert_eq "$RUN_CODE" "2" "gate list positional arg should fail"
assert_contains "$RUN_ERR" "gate list does not accept positional arguments" "positional arg error"

run_capture bad_flag "$SQ_BIN" gate list --wat
assert_eq "$RUN_CODE" "2" "gate list unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --wat" "unknown flag error"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after"
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "gate list should not mutate count summary"

run_capture status_after "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status after"
assert_eq "$RUN_OUT" "$STATUS_BEFORE" "gate list should not mutate status summary"

echo "[gate-list] PASS"
