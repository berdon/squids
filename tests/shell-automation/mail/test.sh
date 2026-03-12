#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-mail-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[mail] FAIL: $*" >&2
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

echo "[mail] binary=$SQ_BIN"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"

run_capture help_cmd "$SQ_BIN" help mail
assert_eq "$RUN_CODE" "0" "help mail"
assert_contains "$RUN_OUT" "mail" "help mail command name"
assert_contains "$RUN_OUT" "Usage:" "help mail usage"

run_capture help_flag "$SQ_BIN" mail --help
assert_eq "$RUN_CODE" "0" "mail --help should succeed"
assert_contains "$RUN_OUT" "mail" "mail --help command name"
assert_contains "$RUN_OUT" "Usage:" "mail --help usage"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "mail --help should not mutate state"

run_capture base "$SQ_BIN" mail
assert_eq "$RUN_CODE" "1" "mail baseline runtime"
assert_contains "$RUN_ERR" "mail compatibility surface only" "mail runtime error"

run_capture json_mode "$SQ_BIN" mail --json
assert_eq "$RUN_CODE" "1" "mail --json runtime"
assert_contains "$RUN_ERR" "mail compatibility surface only" "mail --json runtime error"

run_capture bad_flag "$SQ_BIN" mail --wat
assert_eq "$RUN_CODE" "2" "mail unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "mail unknown flag error"

assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "mail should not mutate state"

echo "[mail] PASS"
