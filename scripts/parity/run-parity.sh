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

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "ASSERT FAILED: expected output to NOT contain: $needle"
    echo "----- output -----"
    echo "$haystack"
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

# 0) help parity
HELP_TXT="$(run_target "help")"
assert_contains "$HELP_TXT" "Usage"
HELP_SUB_TXT="$(run_target "help create")"
assert_contains "$HELP_SUB_TXT" "create"
run_target "help --all" >/dev/null
run_target "help --actor tester" >/dev/null

# 1) init + ready
run_target "init --prefix bd --json" >/dev/null
READY_JSON="$(run_target "ready --json")"
assert_contains "$READY_JSON" "["

# 2) create
CREATE_JSON="$(run_target "create 'Parity lifecycle task' --type task --priority 1 --description 'shell parity' --json")"
TASK_ID="$(json_field "$CREATE_JSON" "id")"
assert_contains "$TASK_ID" "bd-"

# 2a) quick capture command parity
Q_ID_RAW="$(run_target "q QuickCapture --type task --priority 2")"
Q_ID="$(echo "$Q_ID_RAW" | tr -d '\r\n')"
assert_contains "$Q_ID" "bd-"

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

# 6c) children/blocked parity
PARENT_JSON="$(run_target "create 'Parent issue' --type epic --priority 1 --json")"
PARENT_ID="$(json_field "$PARENT_JSON" "id")"
CHILD_JSON="$(run_target "create 'Child issue' --type task --priority 2 --deps parent-child:$PARENT_ID --json")"
CHILD_ID="$(json_field "$CHILD_JSON" "id")"
BLOCKER_JSON="$(run_target "create 'Blocker issue' --type task --priority 1 --json")"
BLOCKER_ID="$(json_field "$BLOCKER_JSON" "id")"
run_target "dep add $BLOCKER_ID $CHILD_ID --json" >/dev/null
CHILDREN_JSON="$(run_target "children $PARENT_ID --json")"
assert_contains "$CHILDREN_JSON" "$CHILD_ID"
BLOCKED_JSON="$(run_target "blocked --json")"
assert_contains "$BLOCKED_JSON" "$CHILD_ID"
assert_contains "$BLOCKED_JSON" "$BLOCKER_ID"
READY_AFTER_BLOCK_JSON="$(run_target "ready --json")"
assert_contains "$READY_AFTER_BLOCK_JSON" "\"id\": \"$CHILD_ID\""
assert_not_contains "$READY_AFTER_BLOCK_JSON" "\"id\": \"$BLOCKER_ID\""

# 6d) comments command family parity (add + list)
COMMENT_ADD_JSON="$(run_target "comments add $TASK_ID 'hello comment' --json")"
assert_contains "$COMMENT_ADD_JSON" "hello comment"
COMMENT_LIST_JSON="$(run_target "comments $TASK_ID --json")"
assert_contains "$COMMENT_LIST_JSON" "hello comment"

# 6e) defer/undefer parity
DEFER_JSON="$(run_target "defer $TASK_ID --json")"
assert_contains "$DEFER_JSON" "deferred"
UNDEFER_JSON="$(run_target "undefer $TASK_ID --json")"
assert_contains "$UNDEFER_JSON" "open"

# 6f) rename/rename-prefix parity
REN_OLD_JSON="$(run_target "create 'Rename target' --type task --priority 2 --json")"
REN_OLD_ID="$(json_field "$REN_OLD_JSON" "id")"
REN_NEW_ID="bd-renamed-target"
REN_JSON="$(run_target "rename $REN_OLD_ID $REN_NEW_ID --json")"
assert_contains "$REN_JSON" "$REN_NEW_ID"
REN_SHOW_JSON="$(run_target "show $REN_NEW_ID --json")"
assert_eq "$(json_field "$REN_SHOW_JSON" "id")" "$REN_NEW_ID"
# 6g) duplicate/supersede parity
ORIG_JSON="$(run_target "create 'Original issue' --type bug --priority 1 --json")"
ORIG_ID="$(json_field "$ORIG_JSON" "id")"
DUP_JSON="$(run_target "create 'Duplicate issue' --type bug --priority 2 --json")"
DUP_ID="$(json_field "$DUP_JSON" "id")"
NEW_JSON="$(run_target "create 'Replacement issue' --type bug --priority 1 --json")"
NEW_ID="$(json_field "$NEW_JSON" "id")"
DUP_OP_JSON="$(run_target "duplicate $DUP_ID --of $ORIG_ID --json")"
assert_contains "$DUP_OP_JSON" "$DUP_ID"
assert_contains "$DUP_OP_JSON" "$ORIG_ID"
DUP_SHOW_JSON="$(run_target "show $DUP_ID --json")"
assert_json_status "$DUP_SHOW_JSON" "closed"
SUP_OP_JSON="$(run_target "supersede $ORIG_ID --with $NEW_ID --json")"
assert_contains "$SUP_OP_JSON" "$ORIG_ID"
assert_contains "$SUP_OP_JSON" "$NEW_ID"
ORIG_SHOW_JSON="$(run_target "show $ORIG_ID --json")"
assert_json_status "$ORIG_SHOW_JSON" "closed"

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

# 13) stale/orphans command parity
run_target "stale --days 1 --json" >/dev/null
run_target "orphans --json" >/dev/null

# 14) search command parity
SEARCH_JSON="$(run_target "search \"Parity lifecycle\" --json -n 5")"
assert_contains "$SEARCH_JSON" "$TASK_ID"
SEARCH_EMPTY_JSON="$(run_target "search \"no-match-xyz-123\" --json")"
assert_contains "$SEARCH_EMPTY_JSON" "["

# 13a) types command parity
TYPES_JSON="$(run_target "types --json")"
assert_contains "$TYPES_JSON" "core_types"
assert_contains "$TYPES_JSON" "task"

# 14) count/status parity
COUNT_JSON="$(run_target "count --json")"
assert_contains "$COUNT_JSON" "count"
COUNT_OPEN_JSON="$(run_target "count --status open --json")"
assert_contains "$COUNT_OPEN_JSON" "count"
STATUS_JSON="$(run_target "status --json")"
assert_contains "$STATUS_JSON" "open"

# 14b) version parity
VERSION_TXT="$(run_target "version")"
assert_contains "$VERSION_TXT" "version"
VERSION_JSON="$(run_target "version --json")"
assert_contains "$VERSION_JSON" "version"
VERSION_HELP="$(run_target "version --help")"
assert_contains "$VERSION_HELP" "Print version information"
run_target "version --quiet" >/dev/null
run_target "version --verbose" >/dev/null
run_target "version --profile" >/dev/null
run_target "version --readonly" >/dev/null
run_target "version --sandbox" >/dev/null
run_target "version --actor tester" >/dev/null
run_target "version --db /tmp/sq.db" >/dev/null
run_target "version --dolt-auto-commit off" >/dev/null
run_target "-V" >/dev/null
run_target "--version" >/dev/null

# 14c) where parity
WHERE_TXT="$(run_target "where")"
assert_contains "$WHERE_TXT" ".sq"
WHERE_JSON="$(run_target "where --json")"
assert_contains "$WHERE_JSON" "database_path"
run_target "where --actor tester" >/dev/null

# 14d) info parity
INFO_JSON="$(run_target "info --json")"
assert_contains "$INFO_JSON" "database_path"
INFO_SCHEMA_JSON="$(run_target "info --schema --json")"
assert_contains "$INFO_SCHEMA_JSON" "schema"
run_target "info --whats-new" >/dev/null
run_target "info --whats-new --json" >/dev/null
run_target "info --thanks" >/dev/null

# 14e) human parity
HUMAN_TASK_JSON="$(run_target "create 'Human-needed' --type task --priority 2 --json")"
HUMAN_TASK_ID="$(json_field "$HUMAN_TASK_JSON" "id")"
run_target "label add $HUMAN_TASK_ID human --json" >/dev/null
HUMAN_LIST_JSON="$(run_target "human list --json")"
assert_contains "$HUMAN_LIST_JSON" "$HUMAN_TASK_ID"
run_target "human stats" >/dev/null
run_target "human respond $HUMAN_TASK_ID --response acknowledged --json" >/dev/null

# 14f) quickstart parity
QUICKSTART_TXT="$(run_target "quickstart")"
assert_contains "$QUICKSTART_TXT" "quickstart"
run_target "quickstart --actor tester" >/dev/null

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
