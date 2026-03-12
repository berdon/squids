#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-where-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[where] FAIL: $*" >&2
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

echo "[where] binary=$SQ_BIN"
echo "[where] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
EXPECTED_DIR="$TMP_DIR"
EXPECTED_DB="$SQ_DB_PATH"
COUNT_BEFORE="$($SQ_BIN count --json)"
STATUS_BEFORE="$($SQ_BIN status --json)"

run_capture help_cmd "$SQ_BIN" help where
assert_eq "$RUN_CODE" "0" "help where"
assert_contains "$RUN_OUT" "where" "help where command name"
assert_contains "$RUN_OUT" "Usage:" "help where usage"

run_capture help_flag "$SQ_BIN" where --help
assert_eq "$RUN_CODE" "0" "where --help"
assert_contains "$RUN_OUT" "Show active sq storage location" "where --help description"
assert_contains "$RUN_OUT" "Usage:" "where --help usage"
assert_contains "$RUN_OUT" "--json" "where --help json flag"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "where --help should not mutate count"

run_capture baseline "$SQ_BIN" where
assert_eq "$RUN_CODE" "0" "where baseline"
assert_contains "$RUN_OUT" "$EXPECTED_DIR" "where path"
assert_contains "$RUN_OUT" "prefix: bd" "where prefix"
assert_contains "$RUN_OUT" "$EXPECTED_DB" "where database path"

run_capture json_mode "$SQ_BIN" where --json
assert_eq "$RUN_CODE" "0" "where json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["path"]')" "$EXPECTED_DIR" "where json path"
assert_eq "$(json_eval "$RUN_OUT" 'obj["database_path"]')" "$EXPECTED_DB" "where json db path"
assert_eq "$(json_eval "$RUN_OUT" 'obj["prefix"]')" "bd" "where json prefix"

run_capture bad_flag "$SQ_BIN" where --wat
assert_eq "$RUN_CODE" "2" "where bad flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "where bad flag error"

run_capture extra "$SQ_BIN" where extra
assert_eq "$RUN_CODE" "2" "where extra positional"
assert_contains "$RUN_ERR" "unexpected positional" "where extra positional error"

assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "where should not mutate count"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "where should not mutate status"

echo "[where] PASS"
