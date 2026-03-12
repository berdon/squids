#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
[[ -x "$SQ_BIN" ]] || (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
TMP_DIR="$(mktemp -d -t sq-history-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT
fail(){ echo "[history] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }
export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
run_capture help_cmd "$SQ_BIN" help history
assert_eq "$RUN_CODE" "0" "help history"
assert_contains "$RUN_OUT" "history" "help history command name"
assert_contains "$RUN_OUT" "Usage:" "help history usage"
run_capture help_flag "$SQ_BIN" history --help
assert_eq "$RUN_CODE" "0" "history --help should succeed"
assert_contains "$RUN_OUT" "history" "history --help command name"
assert_contains "$RUN_OUT" "Usage:" "history --help usage"
run_capture missing_arg "$SQ_BIN" history
assert_eq "$RUN_CODE" "2" "history missing arg"
assert_contains "$RUN_ERR" "usage: sq history <id> [--limit N] [--json]" "history missing arg usage"
run_capture runtime "$SQ_BIN" history bd-missing
assert_eq "$RUN_CODE" "1" "history runtime backend failure"
assert_contains "$RUN_ERR" "history requires Dolt backend" "history runtime error"
run_capture json_mode "$SQ_BIN" history bd-missing --json
assert_eq "$RUN_CODE" "1" "history json runtime"
assert_contains "$RUN_ERR" "history requires Dolt backend" "history json runtime error"
run_capture bad_flag "$SQ_BIN" history --wat
assert_eq "$RUN_CODE" "2" "history unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "history unknown flag error"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "history should not mutate state"
echo "[history] PASS"
