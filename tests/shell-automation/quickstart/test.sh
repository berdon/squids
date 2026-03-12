#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  echo "[quickstart] sq binary not found at $SQ_BIN (run make build first)" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d -t sq-quickstart-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[quickstart] FAIL: $*" >&2
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

WORKSPACE="$TMP_DIR/workspace"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"
export SQ_DB_PATH="$WORKSPACE/tasks.sqlite"

echo "[quickstart] workspace=$WORKSPACE"
echo "[quickstart] binary=$SQ_BIN"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture count_before "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count before"
COUNT_BEFORE="$RUN_OUT"

run_capture status_before "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status before"
STATUS_BEFORE="$RUN_OUT"

run_capture help_cmd "$SQ_BIN" help quickstart
assert_eq "$RUN_CODE" "0" "sq help quickstart"
assert_contains "$RUN_OUT" "Display a quick start guide showing common sq workflows." "help command description"
assert_contains "$RUN_OUT" "sq quickstart [flags]" "help command usage"
assert_contains "$RUN_OUT" "--json" "help command json flag"

run_capture help_flag "$SQ_BIN" quickstart --help
assert_eq "$RUN_CODE" "0" "sq quickstart --help"
assert_contains "$RUN_OUT" "Display a quick start guide showing common sq workflows." "help flag description"
assert_contains "$RUN_OUT" "--readonly" "help flag readonly"
HELP_FLAG_OUT="$RUN_OUT"

run_capture help_cmd_again "$SQ_BIN" help quickstart
assert_eq "$RUN_CODE" "0" "sq help quickstart rerun"
assert_eq "$RUN_OUT" "$HELP_FLAG_OUT" "help command and --help parity"

run_capture default "$SQ_BIN" quickstart
assert_eq "$RUN_CODE" "0" "sq quickstart default"
assert_contains "$RUN_OUT" "sq quickstart" "default banner"
assert_contains "$RUN_OUT" "GETTING STARTED" "default getting started section"
assert_contains "$RUN_OUT" "sq create \"Fix login bug\"" "default create example"
assert_contains "$RUN_OUT" "sq ready" "default ready example"
DEFAULT_OUT="$RUN_OUT"

run_capture json "$SQ_BIN" quickstart --json
assert_eq "$RUN_CODE" "0" "sq quickstart --json"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "--json currently matches default output"

run_capture actor "$SQ_BIN" quickstart --actor tester
assert_eq "$RUN_CODE" "0" "sq quickstart --actor"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "--actor output parity"

run_capture readonly "$SQ_BIN" quickstart --readonly
assert_eq "$RUN_CODE" "0" "sq quickstart --readonly"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "--readonly output parity"

run_capture sandbox "$SQ_BIN" quickstart --sandbox
assert_eq "$RUN_CODE" "0" "sq quickstart --sandbox"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "--sandbox output parity"

run_capture dbflag "$SQ_BIN" quickstart --db "$WORKSPACE/other.sqlite"
assert_eq "$RUN_CODE" "0" "sq quickstart --db"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "--db output parity"

run_capture dolt "$SQ_BIN" quickstart --dolt-auto-commit off
assert_eq "$RUN_CODE" "0" "sq quickstart --dolt-auto-commit"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "--dolt-auto-commit output parity"

run_capture bogus "$SQ_BIN" quickstart --bogus-flag
assert_eq "$RUN_CODE" "2" "unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --bogus-flag" "unknown flag error"

run_capture rerun "$SQ_BIN" quickstart
assert_eq "$RUN_CODE" "0" "rerun quickstart"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "idempotent rerun output"

run_capture count_after "$SQ_BIN" count --json
assert_eq "$RUN_CODE" "0" "count after"
assert_eq "$RUN_OUT" "$COUNT_BEFORE" "quickstart should not mutate count"

run_capture status_after "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "status after"
assert_eq "$RUN_OUT" "$STATUS_BEFORE" "quickstart should not mutate status"

echo "[quickstart] PASS"
