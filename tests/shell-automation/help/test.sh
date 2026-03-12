#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-help-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[help] FAIL: $*" >&2
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

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local context="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    fail "$context: did not expect '$needle' in output"
  fi
}

assert_file_nonempty() {
  local path="$1"
  local context="$2"
  [[ -s "$path" ]] || fail "$context: expected non-empty file at $path"
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

SCRATCH="$TMP_DIR/scratch"
mkdir -p "$SCRATCH"
export SQ_DB_PATH="$SCRATCH/tasks.sqlite"

echo "[help] binary=$SQ_BIN"
echo "[help] scratch=$SCRATCH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init for help test"
run_capture ready_before "$SQ_BIN" ready --json
assert_eq "$RUN_CODE" "0" "ready before help"
READY_BEFORE="$RUN_OUT"

run_capture top_help "$SQ_BIN" help
assert_eq "$RUN_CODE" "0" "sq help"
assert_contains "$RUN_OUT" "Usage:" "top-level help usage"
assert_contains "$RUN_OUT" "sq [flags]" "top-level help flags"
assert_contains "$RUN_OUT" "sq [command]" "top-level help command"
assert_contains "$RUN_OUT" "Working With Issues:" "top-level help command groups"
assert_contains "$RUN_OUT" "Global Flags:" "top-level help global flags"

run_capture help_help "$SQ_BIN" help --help
assert_eq "$RUN_CODE" "0" "sq help --help"
assert_contains "$RUN_OUT" "Help provides help for any command in the application." "help --help description"
assert_contains "$RUN_OUT" "sq help [command] [flags]" "help --help usage"
assert_contains "$RUN_OUT" "--all" "help --help all flag"
assert_contains "$RUN_OUT" "help for help" "help --help text"
assert_contains "$RUN_OUT" "Global Flags:" "help --help global flags"

run_capture help_all "$SQ_BIN" help --all
assert_eq "$RUN_CODE" "0" "sq help --all"
assert_contains "$RUN_OUT" "# sq — Complete Command Reference" "help --all heading"
assert_contains "$RUN_OUT" "## Table of Contents" "help --all toc"
assert_contains "$RUN_OUT" "sq init" "help --all init entry"
assert_contains "$RUN_OUT" "sq create" "help --all create entry"
assert_contains "$RUN_OUT" "sq ready" "help --all ready entry"
printf '%s' "$RUN_OUT" > "$SCRATCH/help-all.md"
assert_file_nonempty "$SCRATCH/help-all.md" "help --all redirect"

run_capture help_create "$SQ_BIN" help create
assert_eq "$RUN_CODE" "0" "sq help create"
assert_contains "$RUN_OUT" "Create a task" "help create description"
assert_contains "$RUN_OUT" "sq create [title] [flags]" "help create usage"
assert_contains "$RUN_OUT" "--description" "help create description flag"
assert_contains "$RUN_OUT" "--priority" "help create priority flag"
assert_contains "$RUN_OUT" "--type" "help create type flag"
assert_contains "$RUN_OUT" "Global Flags:" "help create global flags"
HELP_CREATE_OUT="$RUN_OUT"

run_capture help_ready "$SQ_BIN" help ready
assert_eq "$RUN_CODE" "0" "sq help ready"
assert_contains "$RUN_OUT" "Show ready work" "help ready description"
assert_contains "$RUN_OUT" "--assignee" "help ready assignee flag"
assert_contains "$RUN_OUT" "--label" "help ready label flag"
assert_contains "$RUN_OUT" "--priority" "help ready priority flag"

run_capture help_label "$SQ_BIN" help label
assert_eq "$RUN_CODE" "0" "sq help label"
assert_contains "$RUN_OUT" "sq label add" "help label grouped usage"
run_capture help_query "$SQ_BIN" help query
assert_eq "$RUN_CODE" "0" "sq help query"
assert_contains "$RUN_OUT" "sq query <expression> [flags]" "help query usage"
run_capture help_backup "$SQ_BIN" help backup
assert_eq "$RUN_CODE" "0" "sq help backup"
assert_contains "$RUN_OUT" "sq backup" "help backup usage"
run_capture help_quickstart "$SQ_BIN" help quickstart
assert_eq "$RUN_CODE" "0" "sq help quickstart"
assert_contains "$RUN_OUT" "sq quickstart [flags]" "help quickstart usage"

for flag in --json --quiet --verbose --profile --readonly --sandbox; do
  run_capture "compat_${flag//-/}" "$SQ_BIN" help "$flag"
  assert_eq "$RUN_CODE" "0" "sq help $flag"
  assert_contains "$RUN_OUT" "Usage:" "sq help $flag usage"
done
run_capture compat_actor "$SQ_BIN" help --actor tester
assert_eq "$RUN_CODE" "0" "sq help --actor"
assert_contains "$RUN_OUT" "Usage:" "sq help --actor usage"
run_capture compat_db "$SQ_BIN" help --db "$SCRATCH/tasks.sqlite"
assert_eq "$RUN_CODE" "0" "sq help --db"
assert_contains "$RUN_OUT" "Usage:" "sq help --db usage"
run_capture compat_dolt "$SQ_BIN" help --dolt-auto-commit off
assert_eq "$RUN_CODE" "0" "sq help --dolt-auto-commit"
assert_contains "$RUN_OUT" "Usage:" "sq help --dolt-auto-commit usage"

run_capture too_many "$SQ_BIN" help create list
assert_eq "$RUN_CODE" "2" "sq help too many args"
assert_contains "$RUN_ERR" "help accepts at most one command" "sq help too many args error"

run_capture bad_flag "$SQ_BIN" help --wat
assert_eq "$RUN_CODE" "2" "sq help unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "sq help unknown flag error"

run_capture create_help_flag "$SQ_BIN" create --help
assert_eq "$RUN_CODE" "0" "sq create --help should succeed"
assert_contains "$RUN_OUT" "Create a task" "sq create --help description"
assert_contains "$RUN_OUT" "sq create [title] [flags]" "sq create --help usage"
assert_eq "$RUN_OUT" "$HELP_CREATE_OUT" "sq create --help should align with sq help create"
run_capture ready_help_flag "$SQ_BIN" ready --help
assert_eq "$RUN_CODE" "0" "sq ready --help should succeed"
assert_contains "$RUN_OUT" "Show ready work" "sq ready --help description"

run_capture ready_after "$SQ_BIN" ready --json
assert_eq "$RUN_CODE" "0" "ready after help"
assert_eq "$RUN_OUT" "$READY_BEFORE" "help should not mutate state"

assert_not_contains "$RUN_OUT" "unknown command" "direct help should be coherent"

echo "[help] PASS"
