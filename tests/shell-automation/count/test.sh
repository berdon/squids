#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_eq() {
  local expected="$1"
  local actual="$2"
  local context="$3"
  if [[ "$expected" != "$actual" ]]; then
    fail "$context: expected '$expected', got '$actual'"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local context="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    fail "$context: expected to find '$needle' in '$haystack'"
  fi
}

assert_json_count() {
  local json_input="$1"
  local expected="$2"
  local context="$3"
  local actual
  actual="$(JSON_INPUT="$json_input" python3 - <<'PY'
import json, os
obj = json.loads(os.environ['JSON_INPUT'])
print(obj['count'])
PY
)"
  assert_eq "$expected" "$actual" "$context"
}

run_cmd() {
  local out_file err_file
  out_file="$(mktemp)"
  err_file="$(mktemp)"
  set +e
  "$@" >"$out_file" 2>"$err_file"
  RUN_CODE=$?
  set -e
  RUN_STDOUT="$(python3 - "$out_file" <<'PY'
import pathlib, sys
print(pathlib.Path(sys.argv[1]).read_text(), end="")
PY
)"
  RUN_STDERR="$(python3 - "$err_file" <<'PY'
import pathlib, sys
print(pathlib.Path(sys.argv[1]).read_text(), end="")
PY
)"
  rm -f "$out_file" "$err_file"
}

json_get_id() {
  local json_input="$1"
  JSON_INPUT="$json_input" python3 - <<'PY'
import json, os
print(json.loads(os.environ['JSON_INPUT'])['id'])
PY
}

workspace="$(mktemp -d)"
empty_workspace=""
trap 'rm -rf "$workspace" "$empty_workspace"' EXIT

SQ_BIN="${SQ_BIN:-}"
REPO_ROOT="${REPO_ROOT:-}"
[[ -n "$SQ_BIN" ]] || fail "SQ_BIN must be set"
[[ -x "$SQ_BIN" ]] || fail "SQ_BIN is not executable: $SQ_BIN"
[[ -n "$REPO_ROOT" ]] || fail "REPO_ROOT must be set"
[[ -d "$REPO_ROOT" ]] || fail "REPO_ROOT does not exist: $REPO_ROOT"

echo "using sq binary: $SQ_BIN"
export SQ_DB_PATH="$workspace/tasks.sqlite"
cd "$workspace"

run_cmd "$SQ_BIN" init --json
assert_eq "0" "$RUN_CODE" "sq init should succeed"

run_cmd "$SQ_BIN" create "open task" --type task --priority 1 --json
assert_eq "0" "$RUN_CODE" "create open task"
open_id="$(json_get_id "$RUN_STDOUT")"
[[ -n "$open_id" ]] || fail "missing open task id"

run_cmd "$SQ_BIN" create "second open task" --type bug --priority 2 --json
assert_eq "0" "$RUN_CODE" "create second open task"
second_open_id="$(json_get_id "$RUN_STDOUT")"
[[ -n "$second_open_id" ]] || fail "missing second open task id"

run_cmd "$SQ_BIN" create "in progress task" --type task --priority 2 --json
assert_eq "0" "$RUN_CODE" "create in progress task"
in_progress_id="$(json_get_id "$RUN_STDOUT")"
[[ -n "$in_progress_id" ]] || fail "missing in progress task id"

run_cmd "$SQ_BIN" create "closed task" --type chore --priority 3 --json
assert_eq "0" "$RUN_CODE" "create closed task"
closed_id="$(json_get_id "$RUN_STDOUT")"
[[ -n "$closed_id" ]] || fail "missing closed task id"

run_cmd "$SQ_BIN" update "$in_progress_id" --status in_progress --json
assert_eq "0" "$RUN_CODE" "set in_progress status"
run_cmd "$SQ_BIN" close "$closed_id" --reason "test setup" --json
assert_eq "0" "$RUN_CODE" "close task for setup"

run_cmd "$SQ_BIN" count --help
assert_eq "0" "$RUN_CODE" "count help should succeed"
assert_contains "$RUN_STDOUT" "Count issues matching the specified filters." "count help description"
assert_contains "$RUN_STDOUT" "sq count [flags]" "count help usage"
assert_contains "$RUN_STDOUT" "-s, --status string" "count help status flag"
assert_contains "$RUN_STDOUT" "Global Flags:" "count help global flags"

run_cmd "$SQ_BIN" help count
assert_eq "0" "$RUN_CODE" "help count should succeed"
assert_contains "$RUN_STDOUT" "Help for command: count" "help count output"

run_cmd "$SQ_BIN" count
assert_eq "0" "$RUN_CODE" "count human output"
assert_eq "4" "$(printf '%s' "$RUN_STDOUT" | tr -d '\n\r')" "count total human"

run_cmd "$SQ_BIN" count --json
assert_eq "0" "$RUN_CODE" "count json output"
assert_json_count "$RUN_STDOUT" "4" "count total json"

run_cmd "$SQ_BIN" count --status open --json
assert_eq "0" "$RUN_CODE" "count open json"
assert_json_count "$RUN_STDOUT" "2" "open count"

run_cmd "$SQ_BIN" count --status in_progress --json
assert_eq "0" "$RUN_CODE" "count in_progress json"
assert_json_count "$RUN_STDOUT" "1" "in_progress count"

run_cmd "$SQ_BIN" count --status closed --json
assert_eq "0" "$RUN_CODE" "count closed json"
assert_json_count "$RUN_STDOUT" "1" "closed count"

run_cmd "$SQ_BIN" count -s open
assert_eq "0" "$RUN_CODE" "count open human"
assert_eq "2" "$(printf '%s' "$RUN_STDOUT" | tr -d '\n\r')" "count open human value"

run_cmd "$SQ_BIN" count --status deferred --json
assert_eq "0" "$RUN_CODE" "count deferred json"
assert_json_count "$RUN_STDOUT" "0" "deferred count"

run_cmd "$SQ_BIN" count --json --quiet
assert_eq "0" "$RUN_CODE" "count --quiet"
assert_json_count "$RUN_STDOUT" "4" "count --quiet result"

run_cmd "$SQ_BIN" count --json --verbose
assert_eq "0" "$RUN_CODE" "count --verbose"
assert_json_count "$RUN_STDOUT" "4" "count --verbose result"

run_cmd "$SQ_BIN" count --json --profile
assert_eq "0" "$RUN_CODE" "count --profile"
assert_json_count "$RUN_STDOUT" "4" "count --profile result"

run_cmd "$SQ_BIN" count --json --readonly
assert_eq "0" "$RUN_CODE" "count --readonly"
assert_json_count "$RUN_STDOUT" "4" "count --readonly result"

run_cmd "$SQ_BIN" count --json --sandbox
assert_eq "0" "$RUN_CODE" "count --sandbox"
assert_json_count "$RUN_STDOUT" "4" "count --sandbox result"

run_cmd "$SQ_BIN" count --json --actor tester
assert_eq "0" "$RUN_CODE" "count --actor"
assert_json_count "$RUN_STDOUT" "4" "count --actor result"

run_cmd "$SQ_BIN" count --json --db "$SQ_DB_PATH"
assert_eq "0" "$RUN_CODE" "count --db"
assert_json_count "$RUN_STDOUT" "4" "count --db result"

run_cmd "$SQ_BIN" count --json --dolt-auto-commit off
assert_eq "0" "$RUN_CODE" "count --dolt-auto-commit"
assert_json_count "$RUN_STDOUT" "4" "count --dolt-auto-commit result"

run_cmd "$SQ_BIN" count --wat
assert_eq "2" "$RUN_CODE" "count unknown flag should fail"
assert_contains "$RUN_STDERR" "unknown flag" "count unknown flag error"

run_cmd "$SQ_BIN" count --status
assert_eq "2" "$RUN_CODE" "count missing --status value should fail"
assert_contains "$RUN_STDERR" "missing value" "count missing --status error"

empty_workspace="$(mktemp -d)"
export SQ_DB_PATH="$empty_workspace/tasks.sqlite"
cd "$empty_workspace"
run_cmd "$SQ_BIN" init --json
assert_eq "0" "$RUN_CODE" "init empty database"
run_cmd "$SQ_BIN" count
assert_eq "0" "$RUN_CODE" "count empty database human"
assert_eq "0" "$(printf '%s' "$RUN_STDOUT" | tr -d '\n\r')" "empty database human count"
run_cmd "$SQ_BIN" count --json
assert_eq "0" "$RUN_CODE" "count empty database json"
assert_json_count "$RUN_STDOUT" "0" "empty database json count"

cd "$REPO_ROOT"
run_cmd env TARGET_BIN="$SQ_BIN" ./scripts/parity/run-parity.sh
assert_eq "0" "$RUN_CODE" "parity suite should pass for count coverage"
assert_contains "$RUN_STDOUT" "[parity] PASS" "parity output"

echo "count shell automation passed"
