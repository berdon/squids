#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-memories-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail(){ echo "[memories] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }

export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"

run_capture help_cmd "$SQ_BIN" help memories
assert_eq "$RUN_CODE" "0" "help memories"
assert_contains "$RUN_OUT" "memories" "help memories command name"
assert_contains "$RUN_OUT" "Usage:" "help memories usage"

run_capture help_flag "$SQ_BIN" memories --help
assert_eq "$RUN_CODE" "0" "memories --help should succeed"
assert_contains "$RUN_OUT" "memories" "memories --help command name"
assert_contains "$RUN_OUT" "Usage:" "memories --help usage"

run_capture base "$SQ_BIN" memories
assert_eq "$RUN_CODE" "1" "memories baseline runtime"
assert_contains "$RUN_ERR" "memories compatibility surface only" "memories runtime error"

run_capture json_mode "$SQ_BIN" memories --json
assert_eq "$RUN_CODE" "1" "memories --json runtime"
assert_contains "$RUN_ERR" "memories compatibility surface only" "memories json runtime error"

run_capture bad_flag "$SQ_BIN" memories --wat
assert_eq "$RUN_CODE" "2" "memories unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "memories unknown flag error"

assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "memories should not mutate state"
echo "[memories] PASS"
