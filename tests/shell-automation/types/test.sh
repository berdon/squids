#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  echo "[types] sq binary not found at $SQ_BIN (run make build first)" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d -t sq-types-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[types] FAIL: $*" >&2
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

echo "[types] workspace=$WORKSPACE"
echo "[types] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
COUNT_BEFORE="$RUN_OUT"

run_capture status_before "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status before"
STATUS_BEFORE="$RUN_OUT"

run_capture help_cmd "$SQ_BIN" help types
assert_eq "$RUN_CODE" "0" "sq help types"
assert_contains "$RUN_OUT" "sq types [flags]" "help command usage"
assert_contains "$RUN_OUT" "--json" "help command json flag"
HELP_CMD_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" types --help
assert_eq "$RUN_CODE" "0" "sq types --help should succeed"
assert_contains "$RUN_OUT" "sq types [flags]" "types --help usage"
assert_contains "$RUN_OUT" "--json" "types --help json flag"
HELP_FLAG_OUT="$RUN_OUT"
assert_eq "$HELP_FLAG_OUT" "$HELP_CMD_OUT" "help command and --help parity"

run_capture default "$SQ_BIN" types
assert_eq "$RUN_CODE" "0" "sq types default"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj["core_types"])')" "6" "core type count"
assert_eq "$(json_eval "$RUN_OUT" 'obj["core_types"][0]["name"]')" "task" "first type name"
assert_eq "$(json_eval "$RUN_OUT" 'obj["core_types"][1]["name"]')" "bug" "second type name"
assert_eq "$(json_eval "$RUN_OUT" 'obj["core_types"][-1]["name"]')" "decision" "last type name"
assert_contains "$RUN_OUT" '"description": "General work item (default)"' "default output description"
DEFAULT_OUT="$RUN_OUT"

run_capture json "$SQ_BIN" types --json
assert_eq "$RUN_CODE" "0" "sq types --json"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "default and --json output parity"
assert_eq "$(json_eval "$RUN_OUT" 'len({item["name"] for item in obj["core_types"]})')" "6" "unique type names"

run_capture bogus "$SQ_BIN" types --bogus-flag
assert_eq "$RUN_CODE" "2" "unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --bogus-flag" "unknown flag error"

run_capture rerun "$SQ_BIN" types --json
assert_eq "$RUN_CODE" "0" "rerun types --json"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "idempotent rerun output"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after"
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "types should not mutate count"

run_capture status_after "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status after"
assert_eq "$RUN_OUT" "$STATUS_BEFORE" "types should not mutate status"

echo "[types] PASS"
