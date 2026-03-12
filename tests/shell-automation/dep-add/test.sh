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

json_get_field() {
  local json_input="$1"
  local field="$2"
  JSON_INPUT="$json_input" python3 - "$field" <<'PY'
import json, os, sys
payload = json.loads(os.environ["JSON_INPUT"])
value = payload[sys.argv[1]]
if isinstance(value, bool):
    print(str(value).lower())
elif value is None:
    print("")
else:
    print(value)
PY
}

json_list_len() {
  local json_input="$1"
  JSON_INPUT="$json_input" python3 - <<'PY'
import json, os
print(len(json.loads(os.environ["JSON_INPUT"])))
PY
}

json_list_first() {
  local json_input="$1"
  JSON_INPUT="$json_input" python3 - <<'PY'
import json, os
items = json.loads(os.environ["JSON_INPUT"])
print(items[0] if items else "")
PY
}

workspace="$(mktemp -d)"
trap 'rm -rf "$workspace"' EXIT

SQ_BIN="${SQ_BIN:-}"
[[ -n "$SQ_BIN" ]] || fail "SQ_BIN must be set"
[[ -x "$SQ_BIN" ]] || fail "SQ_BIN is not executable: $SQ_BIN"

echo "using sq binary: $SQ_BIN"
export SQ_DB_PATH="$workspace/tasks.sqlite"
cd "$workspace"

run_cmd "$SQ_BIN" help dep
assert_eq "0" "$RUN_CODE" "sq help dep should succeed"
assert_contains "$RUN_STDOUT" "Help for command: dep" "sq help dep output"

run_cmd "$SQ_BIN" dep add
assert_eq "2" "$RUN_CODE" "sq dep add without args should fail with usage"
assert_contains "$RUN_STDERR" "usage: sq dep add <issue-id> <depends-on-id> [--json]" "sq dep add usage"

run_cmd "$SQ_BIN" dep add --json
assert_eq "2" "$RUN_CODE" "sq dep add --json without ids should fail with usage"
assert_contains "$RUN_STDERR" "usage: sq dep add <issue-id> <depends-on-id> [--json]" "sq dep add --json usage"

before_help_count="$("$SQ_BIN" count --json)"
run_cmd "$SQ_BIN" dep add --help
assert_eq "0" "$RUN_CODE" "sq dep add --help should succeed"
assert_contains "$RUN_STDOUT" "dep add" "sq dep add --help output"
assert_eq "$before_help_count" "$("$SQ_BIN" count --json)" "sq dep add --help should not mutate state"

run_cmd "$SQ_BIN" init --json
assert_eq "0" "$RUN_CODE" "sq init should succeed"
assert_contains "$RUN_STDOUT" '"command": "init"' "sq init json output"

run_cmd "$SQ_BIN" create "task A" --json
assert_eq "0" "$RUN_CODE" "create task A"
issue_a="$(json_get_field "$RUN_STDOUT" id)"

run_cmd "$SQ_BIN" create "task B" --json
assert_eq "0" "$RUN_CODE" "create task B"
issue_b="$(json_get_field "$RUN_STDOUT" id)"

run_cmd "$SQ_BIN" dep add "$issue_a" "$issue_b" --json
assert_eq "0" "$RUN_CODE" "sq dep add success path"
assert_eq "$issue_a" "$(json_get_field "$RUN_STDOUT" issue_id)" "dep add issue_id"
assert_eq "$issue_b" "$(json_get_field "$RUN_STDOUT" depends_on_id)" "dep add depends_on_id"
assert_eq "blocks" "$(json_get_field "$RUN_STDOUT" type)" "dep add type"

run_cmd "$SQ_BIN" dep list "$issue_a" --json
assert_eq "0" "$RUN_CODE" "dep list after add"
assert_eq "1" "$(json_list_len "$RUN_STDOUT")" "dep list should contain one dependency"
assert_eq "$issue_b" "$(json_list_first "$RUN_STDOUT")" "dep list should contain added dependency"

run_cmd "$SQ_BIN" show "$issue_a" --json
assert_eq "0" "$RUN_CODE" "show issue after dep add"
assert_contains "$RUN_STDOUT" "$issue_b" "show should reflect synced deps"

run_cmd "$SQ_BIN" dep add "$issue_a" "$issue_b" --json
assert_eq "0" "$RUN_CODE" "re-adding identical dependency should be safe"
run_cmd "$SQ_BIN" dep list "$issue_a" --json
assert_eq "1" "$(json_list_len "$RUN_STDOUT")" "duplicate dep add should remain idempotent"

run_cmd "$SQ_BIN" count --json
assert_eq "0" "$RUN_CODE" "count before failure checks"
assert_eq "2" "$(json_get_field "$RUN_STDOUT" count)" "expected exactly two tasks before failure checks"

run_cmd "$SQ_BIN" dep add "$issue_a" "bd-missing" --json
assert_eq "1" "$RUN_CODE" "dep add with missing dependency should fail"
assert_contains "$RUN_STDERR" "issue not found: bd-missing" "missing dependency error"
run_cmd "$SQ_BIN" dep list "$issue_a" --json
assert_eq "1" "$(json_list_len "$RUN_STDOUT")" "failed dep add should not partially mutate dependencies"

run_cmd "$SQ_BIN" dep add "bd-missing" "$issue_b" --json
assert_eq "1" "$RUN_CODE" "dep add with missing issue should fail"
assert_contains "$RUN_STDERR" "issue not found: bd-missing" "missing issue error"
run_cmd "$SQ_BIN" count --json
assert_eq "2" "$(json_get_field "$RUN_STDOUT" count)" "failed dep add should not create extra tasks"

run_cmd "$SQ_BIN" dep add "$issue_a" "$issue_b" --wat
assert_eq "2" "$RUN_CODE" "dep add with unknown trailing flag should fail"
assert_contains "$RUN_STDERR" "unknown flag" "unknown trailing flag error"
run_cmd "$SQ_BIN" dep list "$issue_a" --json
assert_eq "1" "$(json_list_len "$RUN_STDOUT")" "unknown flag should not partially mutate dependencies"

echo "dep add shell automation passed"
