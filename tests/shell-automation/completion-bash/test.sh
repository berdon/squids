#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-completion-bash-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[completion-bash] FAIL: $*" >&2
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

echo "[completion-bash] binary=$SQ_BIN"
echo "[completion-bash] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
STATUS_BEFORE="$($SQ_BIN status --json)"

run_capture help_completion "$SQ_BIN" help completion
assert_eq "$RUN_CODE" "0" "help completion"
assert_contains "$RUN_OUT" "Help for command: completion" "help completion command name"
assert_contains "$RUN_OUT" "Usage: sq completion [args]" "help completion usage"

run_capture completion_group "$SQ_BIN" completion
assert_eq "$RUN_CODE" "0" "completion group help"
assert_contains "$RUN_OUT" "Generate the autocompletion script for sq for the specified shell." "completion group description"
assert_contains "$RUN_OUT" "sq completion [command]" "completion group usage"
assert_contains "$RUN_OUT" "bash" "completion group subcommand list"
assert_contains "$RUN_OUT" "Use \"sq completion [command] --help\"" "completion group guidance"

run_capture help_flag "$SQ_BIN" completion bash --help
assert_eq "$RUN_CODE" "0" "completion bash --help"
assert_contains "$RUN_OUT" "Generate the autocompletion script for the bash shell." "completion bash help description"
assert_contains "$RUN_OUT" "source <(sq completion bash)" "completion bash help example"
assert_contains "$RUN_OUT" "--no-descriptions" "completion bash help flag"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "completion bash --help should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "completion bash --help should not mutate status"

run_capture default "$SQ_BIN" completion bash
assert_eq "$RUN_CODE" "0" "completion bash default"
assert_eq "$RUN_ERR" "" "completion bash default stderr"
assert_contains "$RUN_OUT" "# bash completion for sq" "completion bash header"
assert_contains "$RUN_OUT" "__start_sq()" "completion bash function"
assert_contains "$RUN_OUT" "complete -o default -F __start_sq sq" "completion bash registration"
assert_contains "$RUN_OUT" "compgen -W \"bash zsh fish powershell\"" "completion bash completion subcommands"
DEFAULT_OUT="$RUN_OUT"

run_capture no_descriptions "$SQ_BIN" completion bash --no-descriptions
assert_eq "$RUN_CODE" "0" "completion bash --no-descriptions"
assert_eq "$RUN_ERR" "" "completion bash --no-descriptions stderr"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "completion bash --no-descriptions output parity"

run_capture repeat "$SQ_BIN" completion bash
assert_eq "$RUN_CODE" "0" "completion bash repeat"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "completion bash repeat output parity"

run_capture bad_flag "$SQ_BIN" completion bash --wat
assert_eq "$RUN_CODE" "2" "completion bash unknown flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "completion bash unknown flag error"

assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "completion bash should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "completion bash should not mutate status"

echo "[completion-bash] PASS"
