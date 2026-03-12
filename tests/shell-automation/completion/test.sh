#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-completion-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[completion] FAIL: $*" >&2
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

echo "[completion] binary=$SQ_BIN"
echo "[completion] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
STATUS_BEFORE="$($SQ_BIN status --json)"

run_capture help_cmd "$SQ_BIN" help completion
assert_eq "$RUN_CODE" "0" "help completion"
assert_contains "$RUN_OUT" "Help for command: completion" "help completion command name"
assert_contains "$RUN_OUT" "Usage: sq completion [args]" "help completion usage"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "help completion should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "help completion should not mutate status"

run_capture default "$SQ_BIN" completion
assert_eq "$RUN_CODE" "0" "completion default"
assert_eq "$RUN_ERR" "" "completion default stderr"
assert_contains "$RUN_OUT" "Generate the autocompletion script for sq for the specified shell." "completion default description"
assert_contains "$RUN_OUT" "sq completion [command]" "completion default usage"
assert_contains "$RUN_OUT" "Available Commands:" "completion default available commands"
assert_contains "$RUN_OUT" "bash" "completion default bash subcommand"
assert_contains "$RUN_OUT" "fish" "completion default fish subcommand"
assert_contains "$RUN_OUT" "powershell" "completion default powershell subcommand"
assert_contains "$RUN_OUT" "zsh" "completion default zsh subcommand"
assert_contains "$RUN_OUT" "Use \"sq completion [command] --help\"" "completion default guidance"
DEFAULT_OUT="$RUN_OUT"

run_capture help_flag "$SQ_BIN" completion --help
assert_eq "$RUN_CODE" "0" "completion --help"
assert_eq "$RUN_ERR" "" "completion --help stderr"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "completion --help output parity"

run_capture json_mode "$SQ_BIN" completion --json
assert_eq "$RUN_CODE" "2" "completion --json usage failure"
assert_eq "$RUN_OUT" "" "completion --json stdout"
assert_contains "$RUN_ERR" "usage: sq completion <bash|zsh|fish|powershell>" "completion --json error"

run_capture bad_shell "$SQ_BIN" completion wat
assert_eq "$RUN_CODE" "2" "completion unknown shell"
assert_eq "$RUN_OUT" "" "completion unknown shell stdout"
assert_contains "$RUN_ERR" "unknown shell: wat" "completion unknown shell error"

run_capture bad_flag "$SQ_BIN" completion --wat
assert_eq "$RUN_CODE" "2" "completion unknown flag"
assert_eq "$RUN_OUT" "" "completion unknown flag stdout"
assert_contains "$RUN_ERR" "unknown flag: --wat" "completion unknown flag error"

run_capture bash_help "$SQ_BIN" completion bash --help
assert_eq "$RUN_CODE" "0" "completion bash --help"
assert_contains "$RUN_OUT" "Generate the autocompletion script for the bash shell." "completion bash help description"

run_capture zsh_help "$SQ_BIN" completion zsh --help
assert_eq "$RUN_CODE" "0" "completion zsh --help"
assert_contains "$RUN_OUT" "Generate the autocompletion script for the zsh shell." "completion zsh help description"

run_capture repeat "$SQ_BIN" completion
assert_eq "$RUN_CODE" "0" "completion repeat"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "completion repeat output parity"

assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "completion should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "completion should not mutate status"

echo "[completion] PASS"
