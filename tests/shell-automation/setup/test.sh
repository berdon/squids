#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
[[ -x "$SQ_BIN" ]] || (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
TMP_DIR="$(mktemp -d -t sq-setup-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT
fail(){ echo "[setup] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }
export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
run_capture help_cmd "$SQ_BIN" help setup
assert_eq "$RUN_CODE" "0" "help setup"
assert_contains "$RUN_OUT" "setup" "help setup command name"
assert_contains "$RUN_OUT" "Usage:" "help setup usage"
run_capture help_flag "$SQ_BIN" setup --help
assert_eq "$RUN_CODE" "0" "setup --help should succeed"
assert_contains "$RUN_OUT" "setup" "setup --help command name"
assert_contains "$RUN_OUT" "Usage:" "setup --help usage"
run_capture base "$SQ_BIN" setup
assert_eq "$RUN_CODE" "1" "setup baseline runtime"
assert_contains "$RUN_ERR" "setup compatibility surface only" "setup runtime error"
run_capture json_mode "$SQ_BIN" setup --json
assert_eq "$RUN_CODE" "1" "setup --json runtime"
assert_contains "$RUN_ERR" "setup compatibility surface only" "setup json runtime error"
run_capture bad_flag "$SQ_BIN" setup --wat
assert_eq "$RUN_CODE" "2" "setup unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "setup unknown flag error"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "setup should not mutate state"
echo "[setup] PASS"
