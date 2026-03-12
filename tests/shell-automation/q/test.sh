#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-q-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[q] FAIL: $*" >&2
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

assert_id_like() {
  local id="$1"
  local context="$2"
  [[ "$id" =~ ^bd-[a-z0-9]+$ ]] || fail "$context: expected bd-... id, got '$id'"
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

echo "[q] binary=$SQ_BIN"
echo "[q] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
run_capture list_empty "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list empty"
assert_eq "$(json_eval "$RUN_OUT" 'len(obj)')" "0" "empty database"

run_capture basic "$SQ_BIN" q "Quick capture task"
assert_eq "$RUN_CODE" "0" "basic q"
BASIC_ID="$(printf '%s' "$RUN_OUT" | tr -d '\n\r')"
assert_id_like "$BASIC_ID" "basic q id"
run_capture show_basic "$SQ_BIN" show "$BASIC_ID" --json
assert_eq "$RUN_CODE" "0" "show basic q task"
assert_eq "$(json_eval "$RUN_OUT" 'obj["title"]')" "Quick capture task" "basic title"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "task" "basic default type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "2" "basic default priority"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "open" "basic default status"

run_capture json_mode "$SQ_BIN" q "Quick capture json" --json
assert_eq "$RUN_CODE" "0" "q json mode"
JSON_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_id_like "$JSON_ID" "json q id"
run_capture show_json "$SQ_BIN" show "$JSON_ID" --json
assert_eq "$RUN_CODE" "0" "show json q task"

run_capture flags "$SQ_BIN" q "Typed quick task" --type bug --priority 1 --description "Captured quickly" --json
assert_eq "$RUN_CODE" "0" "q with flags"
FLAGS_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture show_flags "$SQ_BIN" show "$FLAGS_ID" --json
assert_eq "$RUN_CODE" "0" "show flagged q task"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "bug" "flagged type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "1" "flagged priority"
assert_eq "$(json_eval "$RUN_OUT" 'obj["description"]')" "Captured quickly" "flagged description"

run_capture seq1 "$SQ_BIN" q "First quick task"
assert_eq "$RUN_CODE" "0" "q sequence 1"
ID1="$(printf '%s' "$RUN_OUT" | tr -d '\n\r')"
run_capture seq2 "$SQ_BIN" q "Second quick task"
assert_eq "$RUN_CODE" "0" "q sequence 2"
ID2="$(printf '%s' "$RUN_OUT" | tr -d '\n\r')"
run_capture seq3 "$SQ_BIN" q "Third quick task" --json
assert_eq "$RUN_CODE" "0" "q sequence 3"
ID3="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_id_like "$ID1" "seq1 id"
assert_id_like "$ID2" "seq2 id"
assert_id_like "$ID3" "seq3 id"
[[ "$ID1" != "$ID2" && "$ID2" != "$ID3" && "$ID1" != "$ID3" ]] || fail "quick capture ids should be distinct"
run_capture list_after_seq "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list after sequence"
assert_contains "$RUN_OUT" "$ID1" "list includes first"
assert_contains "$RUN_OUT" "$ID2" "list includes second"
assert_contains "$RUN_OUT" "$ID3" "list includes third"

run_capture lifecycle "$SQ_BIN" q "Lifecycle quick task"
assert_eq "$RUN_CODE" "0" "q lifecycle create"
LIFECYCLE_ID="$(printf '%s' "$RUN_OUT" | tr -d '\n\r')"
run_capture lifecycle_update "$SQ_BIN" update "$LIFECYCLE_ID" --status in_progress --json
assert_eq "$RUN_CODE" "0" "update lifecycle task"
run_capture lifecycle_close "$SQ_BIN" close "$LIFECYCLE_ID" --reason "Done" --json
assert_eq "$RUN_CODE" "0" "close lifecycle task"
run_capture lifecycle_show "$SQ_BIN" show "$LIFECYCLE_ID" --json
assert_eq "$RUN_CODE" "0" "show lifecycle task"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "closed" "lifecycle closed state"

export BD_ACTOR=tester
run_capture actor "$SQ_BIN" q "Actor quick task" --json
assert_eq "$RUN_CODE" "0" "q with actor"
ACTOR_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_id_like "$ACTOR_ID" "actor q id"
unset BD_ACTOR

run_capture missing_title "$SQ_BIN" q
assert_eq "$RUN_CODE" "2" "q missing title should fail"
assert_contains "$RUN_ERR" "title is required" "q missing title error"

run_capture bad_priority "$SQ_BIN" q "Bad priority" --priority nope
assert_eq "$RUN_CODE" "2" "q invalid priority should fail"
assert_contains "$RUN_ERR" "invalid --priority" "q invalid priority error"

run_capture bad_flag "$SQ_BIN" q "Bad flag" --wat
assert_eq "$RUN_CODE" "2" "q unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --wat" "q unknown flag error"

COUNT_BEFORE_HELP="$($SQ_BIN count --json)"
run_capture help_flag "$SQ_BIN" q --help
assert_eq "$RUN_CODE" "0" "q --help should succeed"
assert_contains "$RUN_OUT" "q" "q --help output"
assert_contains "$RUN_OUT" "Usage:" "q --help usage"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE_HELP" "q --help should not mutate state"

assert_not_contains "$BASIC_ID" "{" "human q output should be id only"
assert_contains "$($SQ_BIN q "Compact json check" --json)" '"id"' "json q output contains id field"

echo "[q] PASS"
