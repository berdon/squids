#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
[[ -x "$SQ_BIN" ]] || (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
TMP_DIR="$(mktemp -d -t sq-gitlab-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT
fail(){ echo "[gitlab] FAIL: $*" >&2; exit 1; }
assert_eq(){ [[ "$1" == "$2" ]] || fail "$3: expected '$2', got '$1'"; }
assert_contains(){ [[ "$1" == *"$2"* ]] || fail "$3: expected '$2' in output"; }
run_capture(){ local n="$1"; shift; local o="$TMP_DIR/${n}.out" e="$TMP_DIR/${n}.err"; set +e; "$@" >"$o" 2>"$e"; RUN_CODE=$?; set -e; RUN_OUT="$(<"$o")"; RUN_ERR="$(<"$e")"; }
export SQ_DB_PATH="$TMP_DIR/tasks.sqlite"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
run_capture help_cmd "$SQ_BIN" help gitlab
assert_eq "$RUN_CODE" "0" "help gitlab"
assert_contains "$RUN_OUT" "gitlab" "help gitlab command name"
assert_contains "$RUN_OUT" "Usage:" "help gitlab usage"
run_capture help_flag "$SQ_BIN" gitlab --help
assert_eq "$RUN_CODE" "0" "gitlab --help should succeed"
assert_contains "$RUN_OUT" "gitlab" "gitlab --help command name"
assert_contains "$RUN_OUT" "Usage:" "gitlab --help usage"
run_capture base "$SQ_BIN" gitlab
assert_eq "$RUN_CODE" "0" "gitlab baseline root usage"
assert_contains "$RUN_OUT" "sq gitlab" "gitlab root usage"
run_capture json_mode "$SQ_BIN" gitlab --json
assert_eq "$RUN_CODE" "0" "gitlab --json root usage"
assert_contains "$RUN_OUT" "sq gitlab" "gitlab json root usage"
run_capture bad_flag "$SQ_BIN" gitlab --wat
assert_eq "$RUN_CODE" "2" "gitlab unknown flag"
assert_contains "$RUN_ERR" "unknown flag" "gitlab unknown flag error"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "gitlab should not mutate state"
echo "[gitlab] PASS"
