#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-audit-label-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[audit-label] FAIL: $*" >&2
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

echo "[audit-label] workspace=$WORKSPACE"
echo "[audit-label] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
COUNT_BEFORE="$RUN_OUT"

run_capture status_before "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status before"
STATUS_BEFORE="$RUN_OUT"

run_capture help_cmd "$SQ_BIN" help audit label
assert_eq "$RUN_CODE" "0" "sq help audit label"
assert_contains "$RUN_OUT" "sq audit label" "help command should mention tuple"
assert_contains "$RUN_OUT" "--json" "help command should mention json flag"
HELP_CMD_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" audit label --help
assert_eq "$RUN_CODE" "0" "sq audit label --help"
assert_contains "$RUN_OUT" "sq audit label" "help flag should mention tuple"
assert_contains "$RUN_OUT" "--json" "help flag should mention json flag"
assert_eq "$RUN_OUT" "$HELP_CMD_OUT" "help command and --help parity"

run_capture unsupported_default "$SQ_BIN" audit label
assert_eq "$RUN_CODE" "1" "sq audit label unsupported runtime"
assert_contains "$RUN_ERR" "audit logging not yet supported on sq sqlite backend" "default unsupported error"

run_capture unsupported_json "$SQ_BIN" audit label --json
assert_eq "$RUN_CODE" "1" "sq audit label --json unsupported runtime"
assert_contains "$RUN_ERR" "audit logging not yet supported on sq sqlite backend" "json unsupported error"

run_capture bad_flag "$SQ_BIN" audit label --wat
assert_eq "$RUN_CODE" "2" "sq audit label unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "unknown flag error"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after"
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "audit label should not mutate count summary"

run_capture status_after "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status after"
assert_eq "$RUN_OUT" "$STATUS_BEFORE" "audit label should not mutate status summary"

echo "[audit-label] PASS"
