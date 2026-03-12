#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-completion-powershell-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[completion-powershell] FAIL: $*" >&2
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

echo "[completion-powershell] binary=$SQ_BIN"
echo "[completion-powershell] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
STATUS_BEFORE="$($SQ_BIN status --json)"

run_capture completion_group "$SQ_BIN" completion
assert_eq "$RUN_CODE" "0" "completion group help"
assert_contains "$RUN_OUT" "powershell" "completion group powershell listing"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "completion group should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "completion group should not mutate status"

run_capture help_flag "$SQ_BIN" completion powershell --help
assert_eq "$RUN_CODE" "0" "completion powershell --help"
assert_eq "$RUN_ERR" "" "completion powershell --help stderr"
assert_contains "$RUN_OUT" "Generate the autocompletion script for powershell." "completion powershell help description"
assert_contains "$RUN_OUT" "sq completion powershell" "completion powershell help usage"
assert_contains "$RUN_OUT" "--no-descriptions" "completion powershell help flag"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "completion powershell --help should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "completion powershell --help should not mutate status"

run_capture default "$SQ_BIN" completion powershell
assert_eq "$RUN_CODE" "0" "completion powershell default"
assert_eq "$RUN_ERR" "" "completion powershell default stderr"
assert_contains "$RUN_OUT" "# powershell completion for sq" "completion powershell header"
assert_contains "$RUN_OUT" "Register-ArgumentCompleter -Native -CommandName 'sq'" "completion powershell registration"
assert_contains "$RUN_OUT" "[System.Management.Automation.CompletionResult]::new" "completion powershell completion result"
assert_contains "$RUN_OUT" "@('bash','zsh','fish','powershell')" "completion powershell shell list"
DEFAULT_OUT="$RUN_OUT"

run_capture no_descriptions "$SQ_BIN" completion powershell --no-descriptions
assert_eq "$RUN_CODE" "0" "completion powershell --no-descriptions"
assert_eq "$RUN_ERR" "" "completion powershell --no-descriptions stderr"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "completion powershell --no-descriptions output parity"

run_capture repeat "$SQ_BIN" completion powershell
assert_eq "$RUN_CODE" "0" "completion powershell repeat"
assert_eq "$RUN_OUT" "$DEFAULT_OUT" "completion powershell repeat output parity"

run_capture bad_flag "$SQ_BIN" completion powershell --wat
assert_eq "$RUN_CODE" "2" "completion powershell unknown flag"
assert_eq "$RUN_OUT" "" "completion powershell unknown flag stdout"
assert_contains "$RUN_ERR" "unknown flag: --wat" "completion powershell unknown flag error"

assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "completion powershell should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "completion powershell should not mutate status"

echo "[completion-powershell] PASS"
