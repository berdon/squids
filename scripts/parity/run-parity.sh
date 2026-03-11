#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_BIN="${TARGET_BIN:-bd}"
TARGET_ARGS="${TARGET_ARGS:-}"

if [[ "$TARGET_BIN" == */* ]]; then
  TARGET_BIN="$(cd "$ROOT_DIR" && realpath "$TARGET_BIN")"
fi

TMP_DIR="$(mktemp -d -t squids-parity-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

WORK_DIR="$TMP_DIR/work"
mkdir -p "$WORK_DIR"

run_target() {
  local cmd="$1"
  if [[ -n "$TARGET_ARGS" ]]; then
    bash -lc "$TARGET_BIN $TARGET_ARGS $cmd"
  else
    bash -lc "$TARGET_BIN $cmd"
  fi
}

json_field() {
  local json="$1"
  local field="$2"
  python3 - <<PY
import json
obj=json.loads('''$json''')
if isinstance(obj, list):
    obj = obj[0] if obj else {}
val=obj
for part in "$field".split('.'):
    if isinstance(val, list) and part.isdigit():
        val = val[int(part)]
    else:
        val=val[part]
print(val if not isinstance(val, (dict,list)) else json.dumps(val))
PY
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "ASSERT FAILED: expected output to contain: $needle"
    echo "----- output -----"
    echo "$haystack"
    exit 1
  fi
}

assert_eq() {
  local got="$1"
  local want="$2"
  if [[ "$got" != "$want" ]]; then
    echo "ASSERT FAILED: expected '$want' got '$got'"
    exit 1
  fi
}

assert_json_status() {
  local json="$1"
  local want="$2"
  local got
  got="$(json_field "$json" "status")"
  assert_eq "$got" "$want"
}

echo "[parity] target=$TARGET_BIN"
cd "$WORK_DIR"

# 1) init + ready
run_target "init --prefix bd --json" >/dev/null
READY_JSON="$(run_target "ready --json")"
assert_contains "$READY_JSON" "["

# 2) create
CREATE_JSON="$(run_target "create 'Parity lifecycle task' --type task --priority 1 --description 'shell parity' --json")"
TASK_ID="$(json_field "$CREATE_JSON" "id")"
assert_contains "$TASK_ID" "bd-"

# 3) show
SHOW_JSON="$(run_target "show $TASK_ID --json")"
assert_eq "$(json_field "$SHOW_JSON" "id")" "$TASK_ID"
assert_eq "$(json_field "$SHOW_JSON" "title")" "Parity lifecycle task"
assert_json_status "$SHOW_JSON" "open"

# 4) update status + assignee
UPDATE_JSON="$(run_target "update $TASK_ID --status in_progress --assignee alice --json")"
assert_json_status "$UPDATE_JSON" "in_progress"
assert_eq "$(json_field "$UPDATE_JSON" "assignee")" "alice"

# 5) list contract
LIST_JSON="$(run_target "list --json --flat --no-pager")"
assert_contains "$LIST_JSON" "$TASK_ID"

# 6) labels + deps
META_JSON="$(run_target "update $TASK_ID --add-label shop:forge --add-label type:mail --json")"
assert_contains "$META_JSON" "shop:forge"

# 6a) label command family parity
LABEL_ADD_JSON="$(run_target "label add $TASK_ID triage --json")"
assert_contains "$LABEL_ADD_JSON" "triage"
LABEL_LIST_JSON="$(run_target "label list $TASK_ID --json")"
assert_contains "$LABEL_LIST_JSON" "triage"
LABEL_REMOVE_JSON="$(run_target "label remove $TASK_ID triage --json")"
LABEL_LIST_AFTER_REMOVE="$(run_target "label list $TASK_ID --json")"
if [[ "$LABEL_LIST_AFTER_REMOVE" == *"triage"* ]]; then
  echo "ASSERT FAILED: expected triage to be removed"
  exit 1
fi
LABEL_ALL_JSON="$(run_target "label list-all --json")"
assert_contains "$LABEL_ALL_JSON" "shop:forge"

SECOND_JSON="$(run_target "create 'Dependency task' --type task --priority 2 --json")"
SECOND_ID="$(json_field "$SECOND_JSON" "id")"
DEP_JSON="$(run_target "update $TASK_ID --set-metadata upstream=$SECOND_ID --json")"
assert_contains "$DEP_JSON" "$SECOND_ID"

# 6b) dep command family parity
DEP_ADD_JSON="$(run_target "dep add $TASK_ID $SECOND_ID --json")"
assert_contains "$DEP_ADD_JSON" "$SECOND_ID"
DEP_LIST_JSON="$(run_target "dep list $TASK_ID --json")"
assert_contains "$DEP_LIST_JSON" "$SECOND_ID"
DEP_REMOVE_JSON="$(run_target "dep remove $TASK_ID $SECOND_ID --json")"
assert_contains "$DEP_REMOVE_JSON" "removed"
DEP_LIST_AFTER_REMOVE="$(run_target "dep list $TASK_ID --json")"
if [[ "$DEP_LIST_AFTER_REMOVE" == *"$SECOND_ID"* ]]; then
  echo "ASSERT FAILED: expected dependency to be removed"
  exit 1
fi

# 6c) comments command family parity (add + list)
COMMENT_ADD_JSON="$(run_target "comments add $TASK_ID 'hello comment' --json")"
assert_contains "$COMMENT_ADD_JSON" "hello comment"
COMMENT_LIST_JSON="$(run_target "comments $TASK_ID --json")"
assert_contains "$COMMENT_LIST_JSON" "hello comment"

# 7) assignee clear path (empty assignee should be accepted)
CLEAR_JSON="$(run_target "update $TASK_ID --assignee '' --status open --json")"
assert_json_status "$CLEAR_JSON" "open"

# 8) close + verify terminal state
CLOSE_JSON="$(run_target "close $TASK_ID --reason 'Completed parity test' --json")"
assert_json_status "$CLOSE_JSON" "closed"
SHOW_CLOSED="$(run_target "show $TASK_ID --json")"
assert_json_status "$SHOW_CLOSED" "closed"

# 9) reopen path
REOPEN_JSON="$(run_target "reopen $TASK_ID --json")"
assert_json_status "$REOPEN_JSON" "open"
SHOW_REOPENED="$(run_target "show $TASK_ID --json")"
assert_json_status "$SHOW_REOPENED" "open"

# 10) delete path
DELETE_JSON="$(run_target "delete $SECOND_ID --force --json")"
assert_contains "$DELETE_JSON" "deleted"
set +e
SHOW_DELETED_OUT="$(run_target "show $SECOND_ID --json" 2>&1)"
SHOW_DELETED_CODE=$?
set -e
if [[ $SHOW_DELETED_CODE -eq 0 ]]; then
  echo "ASSERT FAILED: expected deleted issue show to fail"
  exit 1
fi

# 11) claim path
THIRD_JSON="$(run_target "create 'Claim me' --type task --priority 2 --json")"
THIRD_ID="$(json_field "$THIRD_JSON" "id")"
CLAIM_JSON="$(run_target "update $THIRD_ID --claim --json")"
assert_contains "$CLAIM_JSON" "in_progress"

# 11a) todo command parity
TODO_ADD_JSON="$(run_target "todo add 'Parity todo item' --priority 2 --json")"
TODO_ID="$(json_field "$TODO_ADD_JSON" "id")"
assert_eq "$(json_field "$TODO_ADD_JSON" "issue_type")" "task"
TODO_LIST_JSON="$(run_target "todo --json")"
assert_contains "$TODO_LIST_JSON" "$TODO_ID"
TODO_DONE_JSON="$(run_target "todo done $TODO_ID --json")"
assert_contains "$TODO_DONE_JSON" "closed"

# 12) query command parity
QUERY_JSON="$(run_target "query \"status=open AND priority<=2\" --json")"
assert_contains "$QUERY_JSON" "$TASK_ID"
set +e
BAD_QUERY_OUT="$(run_target "query \"madeupfield=abc\" --json" 2>&1)"
BAD_QUERY_CODE=$?
set -e
if [[ $BAD_QUERY_CODE -eq 0 ]]; then
  echo "ASSERT FAILED: expected bad query to fail"
  exit 1
fi
assert_contains "$BAD_QUERY_OUT" "unknown"

# 13) search command parity
SEARCH_JSON="$(run_target "search \"Parity lifecycle\" --json -n 5")"
assert_contains "$SEARCH_JSON" "$TASK_ID"
SEARCH_EMPTY_JSON="$(run_target "search \"no-match-xyz-123\" --json")"
assert_contains "$SEARCH_EMPTY_JSON" "["

# 14) count/status parity
COUNT_JSON="$(run_target "count --json")"
assert_contains "$COUNT_JSON" "count"
COUNT_OPEN_JSON="$(run_target "count --status open --json")"
assert_contains "$COUNT_OPEN_JSON" "count"
STATUS_JSON="$(run_target "status --json")"
assert_contains "$STATUS_JSON" "open"

# 15) negative path: missing issue show should fail
set +e
MISSING_OUT="$(run_target "show bd-does-not-exist --json" 2>&1)"
MISSING_CODE=$?
set -e
if [[ $MISSING_CODE -eq 0 ]]; then
  echo "ASSERT FAILED: expected missing issue show to fail"
  exit 1
fi
assert_contains "$MISSING_OUT" "not"

# 14) unknown flag should fail with usage-like error
set +e
BAD_FLAG_OUT="$(run_target "create 'x' --bogus-flag --json" 2>&1)"
BAD_FLAG_CODE=$?
set -e
if [[ $BAD_FLAG_CODE -eq 0 ]]; then
  echo "ASSERT FAILED: expected unknown flag to fail"
  exit 1
fi
assert_contains "$BAD_FLAG_OUT" "unknown flag"

echo "[parity] PASS"
