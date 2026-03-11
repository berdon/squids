#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_BIN="${TARGET_BIN:-bd}"
TARGET_ARGS="${TARGET_ARGS:-}"

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
SECOND_JSON="$(run_target "create 'Dependency task' --type task --priority 2 --json")"
SECOND_ID="$(json_field "$SECOND_JSON" "id")"
DEP_JSON="$(run_target "update $TASK_ID --set-metadata upstream=$SECOND_ID --json")"
assert_contains "$DEP_JSON" "$SECOND_ID"

# 7) close + verify terminal state
CLOSE_JSON="$(run_target "close $TASK_ID --reason 'Completed parity test' --json")"
assert_json_status "$CLOSE_JSON" "closed"
SHOW_CLOSED="$(run_target "show $TASK_ID --json")"
assert_json_status "$SHOW_CLOSED" "closed"

# 8) claim path
THIRD_JSON="$(run_target "create 'Claim me' --type task --priority 2 --json")"
THIRD_ID="$(json_field "$THIRD_JSON" "id")"
CLAIM_JSON="$(run_target "update $THIRD_ID --claim --json")"
assert_contains "$CLAIM_JSON" "in_progress"

# 9) negative path: missing issue show should fail
set +e
MISSING_OUT="$(run_target "show bd-does-not-exist --json" 2>&1)"
MISSING_CODE=$?
set -e
if [[ $MISSING_CODE -eq 0 ]]; then
  echo "ASSERT FAILED: expected missing issue show to fail"
  exit 1
fi
assert_contains "$MISSING_OUT" "not"

echo "[parity] PASS"
