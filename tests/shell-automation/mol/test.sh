#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
[[ -x "$SQ_BIN" ]] || (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
TMP_DIR="$(mktemp -d -t sq-mol-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT
fail(){ echo "[mol] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }
export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
run_capture help_cmd "$SQ_BIN" help mol
assert_eq "$RUN_CODE" "0" "help mol"
assert_contains "$RUN_OUT" "mol" "help mol command name"
assert_contains "$RUN_OUT" "Usage:" "help mol usage"
run_capture help_flag "$SQ_BIN" mol --help
assert_eq "$RUN_CODE" "0" "mol --help should succeed"
assert_contains "$RUN_OUT" "mol" "mol --help command name"
assert_contains "$RUN_OUT" "Usage:" "mol --help usage"
run_capture base "$SQ_BIN" mol
assert_eq "$RUN_CODE" "1" "mol baseline runtime"
assert_contains "$RUN_ERR" "mol compatibility surface only" "mol runtime error"
run_capture json_mode "$SQ_BIN" mol --json
assert_eq "$RUN_CODE" "1" "mol --json runtime"
assert_contains "$RUN_ERR" "mol compatibility surface only" "mol json runtime error"
run_capture bad_flag "$SQ_BIN" mol --wat
assert_eq "$RUN_CODE" "2" "mol unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "mol unknown flag error"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "mol should not mutate state"
echo "[mol] PASS"
