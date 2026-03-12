#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-label-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[label] FAIL: $*" >&2
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

json_eval() {
  local payload="$1"
  local expr="$2"
  JSON_INPUT="$payload" "$PYTHON" - "$expr" <<'PY'
import json, os, sys
obj = json.loads(os.environ["JSON_INPUT"])
expr = sys.argv[1]
value = eval(expr, {"__builtins__": {}}, {"obj": obj, "len": len})
if isinstance(value, (dict, list)):
    print(json.dumps(value))
elif isinstance(value, bool):
    print("true" if value else "false")
elif value is None:
    print("")
else:
    print(value)
PY
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

echo "[label] binary=$SQ_BIN"
echo "[label] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
STATUS_BEFORE="$($SQ_BIN status --json)"

run_capture help_label "$SQ_BIN" help label
assert_eq "$RUN_CODE" "0" "help label"
assert_contains "$RUN_OUT" "Manage labels on tasks" "help label description"
assert_contains "$RUN_OUT" "sq label add <id> <label> [--json]" "help label add usage"
assert_contains "$RUN_OUT" "sq label remove <id> <label> [--json]" "help label remove usage"
assert_contains "$RUN_OUT" "sq label list <id> [--json]" "help label list usage"
assert_contains "$RUN_OUT" "sq label list-all [--json]" "help label list-all usage"

run_capture label_help "$SQ_BIN" label --help
assert_eq "$RUN_CODE" "0" "label --help"
assert_eq "$RUN_OUT" "$($SQ_BIN help label)" "label --help parity"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "label --help should not mutate state"
assert_eq "$($SQ_BIN status --json)" "$STATUS_BEFORE" "label --help should not mutate status"

run_capture label_missing_subcommand "$SQ_BIN" label
assert_eq "$RUN_CODE" "2" "label missing subcommand"
assert_contains "$RUN_ERR" "label subcommand required" "label missing subcommand error"

run_capture label_json_without_subcommand "$SQ_BIN" label --json
assert_eq "$RUN_CODE" "2" "label json without subcommand"
assert_contains "$RUN_ERR" "unknown label subcommand: --json" "label json without subcommand error"

run_capture create_task "$SQ_BIN" create "label shell automation task" --json
assert_eq "$RUN_CODE" "0" "create label task"
TASK_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
COUNT_AFTER_CREATE="$($SQ_BIN count --json)"
STATUS_AFTER_CREATE="$($SQ_BIN status --json)"

run_capture add_json "$SQ_BIN" label add "$TASK_ID" triage --json
assert_eq "$RUN_CODE" "0" "label add json"
assert_eq "$(json_eval "$RUN_OUT" 'obj["id"]')" "$TASK_ID" "label add json id"
assert_contains "$RUN_OUT" "triage" "label add json labels"

run_capture add_human "$SQ_BIN" label add "$TASK_ID" human
assert_eq "$RUN_CODE" "0" "label add human"
assert_contains "$RUN_OUT" "Added label 'human' to $TASK_ID" "label add human output"
assert_not_contains "$RUN_OUT" '"id"' "label add human should not be raw JSON"

run_capture list_json "$SQ_BIN" label list "$TASK_ID" --json
assert_eq "$RUN_CODE" "0" "label list json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "2" "label list json count"
assert_contains "$RUN_OUT" "triage" "label list json triage"
assert_contains "$RUN_OUT" "human" "label list json human"

run_capture show_task "$SQ_BIN" show "$TASK_ID" --json
assert_eq "$RUN_CODE" "0" "show labeled task"
assert_contains "$RUN_OUT" "triage" "show labeled task triage"
assert_contains "$RUN_OUT" "human" "show labeled task human"

run_capture list_human "$SQ_BIN" label list "$TASK_ID"
assert_eq "$RUN_CODE" "0" "label list human"
assert_contains "$RUN_OUT" "Labels for $TASK_ID" "label list human heading"
assert_contains "$RUN_OUT" "- triage" "label list human triage"
assert_contains "$RUN_OUT" "- human" "label list human human"

run_capture list_all_json "$SQ_BIN" label list-all --json
assert_eq "$RUN_CODE" "0" "label list-all json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "2" "label list-all json count"
assert_contains "$RUN_OUT" "triage" "label list-all json triage"
assert_contains "$RUN_OUT" "human" "label list-all json human"

run_capture list_all_human "$SQ_BIN" label list-all
assert_eq "$RUN_CODE" "0" "label list-all human"
assert_contains "$RUN_OUT" "All labels (2 unique):" "label list-all human heading"
assert_contains "$RUN_OUT" "human" "label list-all human includes human"
assert_contains "$RUN_OUT" "triage" "label list-all human includes triage"
assert_contains "$RUN_OUT" "(1 issues)" "label list-all human counts"

run_capture remove_json "$SQ_BIN" label remove "$TASK_ID" triage --json
assert_eq "$RUN_CODE" "0" "label remove json"
assert_not_contains "$RUN_OUT" "triage" "label remove json removed triage"
assert_contains "$RUN_OUT" "human" "label remove json kept human"

run_capture remove_human "$SQ_BIN" label remove "$TASK_ID" human
assert_eq "$RUN_CODE" "0" "label remove human"
assert_contains "$RUN_OUT" "Removed label 'human' from $TASK_ID" "label remove human output"

run_capture list_empty "$SQ_BIN" label list "$TASK_ID" --json
assert_eq "$RUN_CODE" "0" "label list empty json"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "0" "label list empty count"

run_capture list_empty_human "$SQ_BIN" label list "$TASK_ID"
assert_eq "$RUN_CODE" "0" "label list empty human"
assert_contains "$RUN_OUT" "(none)" "label list empty human marker"

run_capture add_missing_args "$SQ_BIN" label add
assert_eq "$RUN_CODE" "2" "label add missing args"
assert_contains "$RUN_ERR" "usage: sq label add <id> <label> [--json]" "label add missing args error"

run_capture list_missing_id "$SQ_BIN" label list
assert_eq "$RUN_CODE" "2" "label list missing id"
assert_contains "$RUN_ERR" "usage: sq label list <id> [--json]" "label list missing id error"

run_capture remove_missing_args "$SQ_BIN" label remove
assert_eq "$RUN_CODE" "2" "label remove missing args"
assert_contains "$RUN_ERR" "usage: sq label remove <id> <label> [--json]" "label remove missing args error"

run_capture bad_subcommand "$SQ_BIN" label wat
assert_eq "$RUN_CODE" "2" "label bad subcommand"
assert_contains "$RUN_ERR" "unknown label subcommand: wat" "label bad subcommand error"

assert_eq "$($SQ_BIN count --json)" "$COUNT_AFTER_CREATE" "label should not change issue count"
assert_eq "$($SQ_BIN status --json)" "$STATUS_AFTER_CREATE" "label should not change status counts"

echo "[label] PASS"
