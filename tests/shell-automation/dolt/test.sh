#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
[[ -x "$SQ_BIN" ]] || (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
TMP_DIR="$(mktemp -d -t sq-dolt-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT
fail(){ echo "[dolt] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }
export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
run_capture help_cmd "$SQ_BIN" help dolt
assert_eq "$RUN_CODE" "0" "help dolt"
assert_contains "$RUN_OUT" "dolt" "help dolt command name"
assert_contains "$RUN_OUT" "Usage:" "help dolt usage"
run_capture help_flag "$SQ_BIN" dolt --help
assert_eq "$RUN_CODE" "0" "dolt --help should succeed"
assert_contains "$RUN_OUT" "dolt" "dolt --help command name"
assert_contains "$RUN_OUT" "Usage:" "dolt --help usage"
run_capture base "$SQ_BIN" dolt
assert_eq "$RUN_CODE" "1" "dolt baseline runtime"
assert_contains "$RUN_ERR" "dolt integration not yet supported" "dolt runtime error"
run_capture json_mode "$SQ_BIN" dolt --json
assert_eq "$RUN_CODE" "1" "dolt --json runtime"
assert_contains "$RUN_ERR" "dolt integration not yet supported" "dolt json runtime error"
run_capture bad_flag "$SQ_BIN" dolt --wat
assert_eq "$RUN_CODE" "2" "dolt unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "dolt unknown flag error"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "dolt should not mutate state"
echo "[dolt] PASS"
