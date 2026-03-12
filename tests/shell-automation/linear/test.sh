#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
[[ -x "$SQ_BIN" ]] || (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
TMP_DIR="$(mktemp -d -t sq-linear-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT
fail(){ echo "[linear] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }
export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
run_capture help_cmd "$SQ_BIN" help linear
assert_eq "$RUN_CODE" "0" "help linear"
assert_contains "$RUN_OUT" "linear" "help linear command name"
assert_contains "$RUN_OUT" "Usage:" "help linear usage"
run_capture help_flag "$SQ_BIN" linear --help
assert_eq "$RUN_CODE" "0" "linear --help should succeed"
assert_contains "$RUN_OUT" "linear" "linear --help command name"
assert_contains "$RUN_OUT" "Usage:" "linear --help usage"
run_capture base "$SQ_BIN" linear
assert_eq "$RUN_CODE" "1" "linear baseline runtime"
assert_contains "$RUN_ERR" "linear integration not yet supported" "linear runtime error"
run_capture json_mode "$SQ_BIN" linear --json
assert_eq "$RUN_CODE" "1" "linear --json runtime"
assert_contains "$RUN_ERR" "linear integration not yet supported" "linear json runtime error"
run_capture bad_flag "$SQ_BIN" linear --wat
assert_eq "$RUN_CODE" "2" "linear unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "linear unknown flag error"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "linear should not mutate state"
echo "[linear] PASS"
