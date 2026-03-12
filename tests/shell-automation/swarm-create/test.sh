#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-swarm-create-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[swarm-create] FAIL: $*" >&2
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

assert_empty() {
  local got="$1"
  local context="$2"
  if [[ -n "$got" ]]; then
    fail "$context: expected empty output, got '$got'"
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local context="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    fail "$context: did not expect '$needle' in output"
  fi
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

export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"

echo "[swarm-create] binary=$SQ_BIN"
echo "[swarm-create] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
assert_eq "$(json_eval "$RUN_OUT" 'obj["command"]')" "init" "init command"
assert_eq "$(json_eval "$RUN_OUT" 'obj["ok"]')" "true" "init ok"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before swarm create"
COUNT_BEFORE="$RUN_OUT"
assert_eq "$(json_eval "$COUNT_BEFORE" 'obj["count"]')" "0" "count before swarm create payload"

run_capture help_swarm "$SQ_BIN" help swarm
assert_eq "$RUN_CODE" "0" "sq help swarm"
assert_contains "$RUN_OUT" "Help for command: swarm" "help swarm heading"
assert_contains "$RUN_OUT" "Usage: sq swarm [args]" "help swarm usage"

run_capture swarm_group "$SQ_BIN" swarm
assert_eq "$RUN_CODE" "0" "sq swarm"
assert_eq "$RUN_OUT" "sq swarm [create|list|status|validate]" "sq swarm discovery surface"
assert_empty "$RUN_ERR" "sq swarm stderr"

run_capture create_default "$SQ_BIN" swarm create
assert_eq "$RUN_CODE" "1" "sq swarm create runtime status"
assert_empty "$RUN_OUT" "sq swarm create stdout"
assert_contains "$RUN_ERR" "swarm create not yet supported on sq sqlite backend" "sq swarm create runtime error"

run_capture create_json "$SQ_BIN" swarm create --json
assert_eq "$RUN_CODE" "1" "sq swarm create --json runtime status"
assert_empty "$RUN_OUT" "sq swarm create --json stdout"
assert_contains "$RUN_ERR" "swarm create not yet supported on sq sqlite backend" "sq swarm create --json runtime error"
assert_not_contains "$RUN_ERR" "{" "sq swarm create --json should not claim json output"

run_capture create_help_flag "$SQ_BIN" swarm create --help
assert_eq "$RUN_CODE" "1" "sq swarm create --help runtime status"
assert_empty "$RUN_OUT" "sq swarm create --help stdout"
assert_contains "$RUN_ERR" "swarm create not yet supported on sq sqlite backend" "sq swarm create --help runtime error"

run_capture create_bad_flag "$SQ_BIN" swarm create --wat
assert_eq "$RUN_CODE" "2" "sq swarm create unknown flag"
assert_empty "$RUN_OUT" "sq swarm create unknown flag stdout"
assert_contains "$RUN_ERR" "unknown flag: --wat" "sq swarm create unknown flag error"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after swarm create checks"
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "swarm create failures should not mutate sq state"

run_capture list_after "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list after swarm create checks"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "0" "list after swarm create checks should remain empty"

echo "[swarm-create] PASS"
