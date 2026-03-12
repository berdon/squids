#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-create-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[create] FAIL: $*" >&2
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

echo "[create] binary=$SQ_BIN"
echo "[create] db=$SQ_DB_PATH"

run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"

run_capture seed_parent "$SQ_BIN" create "parent epic" --type epic --json
assert_eq "$RUN_CODE" "0" "seed parent"
PARENT_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture seed_blocker "$SQ_BIN" create "blocking task" --type task --priority 1 --json
assert_eq "$RUN_CODE" "0" "seed blocker"
BLOCKER_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"

run_capture help_create "$SQ_BIN" help create
assert_eq "$RUN_CODE" "0" "help create"
assert_contains "$RUN_OUT" "Create a task" "help create description"
assert_contains "$RUN_OUT" "--description" "help create description flag"
assert_contains "$RUN_OUT" "--type" "help create type flag"
assert_contains "$RUN_OUT" "--priority" "help create priority flag"
assert_contains "$RUN_OUT" "--deps" "help create deps flag"
assert_contains "$RUN_OUT" "--json" "help create json flag"
assert_contains "$RUN_OUT" "Global Flags:" "help create global flags"

run_capture basic "$SQ_BIN" create "basic created task" --json
assert_eq "$RUN_CODE" "0" "basic create"
BASIC_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_id_like "$BASIC_ID" "basic create id"
assert_eq "$(json_eval "$RUN_OUT" 'obj["title"]')" "basic created task" "basic title"
assert_eq "$(json_eval "$RUN_OUT" 'obj["status"]')" "open" "basic status"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "task" "basic type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "0" "basic default priority"

run_capture full "$SQ_BIN" create "full create task" --description "detailed description" --type feature --priority 0 --json
assert_eq "$RUN_CODE" "0" "full create"
FULL_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_eq "$(json_eval "$RUN_OUT" 'obj["description"]')" "detailed description" "full description"
assert_eq "$(json_eval "$RUN_OUT" 'obj["issue_type"]')" "feature" "full type"
assert_eq "$(json_eval "$RUN_OUT" 'obj["priority"]')" "0" "full priority"
run_capture full_show "$SQ_BIN" show "$FULL_ID" --json
assert_eq "$RUN_CODE" "0" "show full create"
assert_eq "$(json_eval "$RUN_OUT" 'obj["description"]')" "detailed description" "persisted description"

run_capture deps "$SQ_BIN" create "task with deps" --deps "parent-child:$PARENT_ID,blocks:$BLOCKER_ID" --json
assert_eq "$RUN_CODE" "0" "create with comma-separated deps"
DEPS_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture deps_show "$SQ_BIN" show "$DEPS_ID" --json
assert_eq "$RUN_CODE" "0" "show deps task"
run_capture children "$SQ_BIN" children "$PARENT_ID" --json
assert_eq "$RUN_CODE" "0" "children after deps create"
assert_contains "$RUN_OUT" "$DEPS_ID" "children includes deps task"
run_capture blocked "$SQ_BIN" blocked --json
assert_eq "$RUN_CODE" "0" "blocked after deps create"
assert_contains "$RUN_OUT" "$DEPS_ID" "blocked includes deps task"
run_capture dep_list "$SQ_BIN" dep list "$DEPS_ID" --json
assert_eq "$RUN_CODE" "0" "dep list after create"
assert_contains "$RUN_OUT" "$PARENT_ID" "dep list parent ref"
assert_contains "$RUN_OUT" "$BLOCKER_ID" "dep list blocker ref"

run_capture actor_env env BD_ACTOR=alice "$SQ_BIN" create "actor created task" --json
assert_eq "$RUN_CODE" "0" "create with actor env"
ACTOR_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_id_like "$ACTOR_ID" "actor create id"
run_capture compat_actor "$SQ_BIN" create "compat task" --actor tester --json
assert_eq "$RUN_CODE" "0" "create with --actor compat flag"
run_capture compat_db "$SQ_BIN" create "compat task db" --db "$SQ_DB_PATH" --json
assert_eq "$RUN_CODE" "0" "create with --db compat flag"
run_capture compat_dolt "$SQ_BIN" create "compat task auto commit" --dolt-auto-commit off --json
assert_eq "$RUN_CODE" "0" "create with --dolt-auto-commit compat flag"

COUNT_BEFORE_HUMAN="$($SQ_BIN count --json)"
COUNT_BEFORE_HUMAN_VALUE="$(json_eval "$COUNT_BEFORE_HUMAN" 'obj["count"]')"
run_capture human "$SQ_BIN" create "human mode task"
assert_eq "$RUN_CODE" "0" "human create"
assert_not_contains "$RUN_OUT" '"id"' "human create should not be raw JSON"
assert_contains "$RUN_OUT" "bd-" "human create should include created id"
HUMAN_ID="$(printf '%s' "$RUN_OUT" | grep -o 'bd-[a-z0-9]\+' | head -1)"
assert_id_like "$HUMAN_ID" "human create id"
run_capture human_show "$SQ_BIN" show "$HUMAN_ID" --json
assert_eq "$RUN_CODE" "0" "show human-created task"
COUNT_AFTER_HUMAN="$($SQ_BIN count --json)"
assert_eq "$(json_eval "$COUNT_AFTER_HUMAN" 'obj["count"]')" "$((COUNT_BEFORE_HUMAN_VALUE + 1))" "human create should change count by exactly one"

run_capture missing_title "$SQ_BIN" create
assert_eq "$RUN_CODE" "2" "missing title"
assert_contains "$RUN_ERR" "title is required" "missing title error"

run_capture bad_priority "$SQ_BIN" create "bad priority" --priority nope --json
assert_eq "$RUN_CODE" "2" "bad priority"
assert_contains "$RUN_ERR" "invalid --priority" "bad priority error"

run_capture bad_deps "$SQ_BIN" create "bad deps" --deps : --json
assert_eq "$RUN_CODE" "2" "bad deps"
assert_contains "$RUN_ERR" "invalid --deps value" "bad deps error"

run_capture bad_flag "$SQ_BIN" create "bad flag" --wat
assert_eq "$RUN_CODE" "2" "bad flag"
assert_contains "$RUN_ERR" "unknown flag: --wat" "bad flag error"

run_capture missing_description "$SQ_BIN" create "missing description value" --description
assert_eq "$RUN_CODE" "2" "missing description value"
assert_contains "$RUN_ERR" "missing value" "missing description error"

run_capture missing_type "$SQ_BIN" create "missing type value" --type
assert_eq "$RUN_CODE" "2" "missing type value"
assert_contains "$RUN_ERR" "missing value" "missing type error"

run_capture missing_priority "$SQ_BIN" create "missing priority value" --priority
assert_eq "$RUN_CODE" "2" "missing priority value"
assert_contains "$RUN_ERR" "missing value" "missing priority error"

run_capture missing_actor "$SQ_BIN" create "missing actor value" --actor
assert_eq "$RUN_CODE" "2" "missing actor value"
assert_contains "$RUN_ERR" "missing value" "missing actor error"
run_capture missing_db "$SQ_BIN" create "missing db value" --db
assert_eq "$RUN_CODE" "2" "missing db value"
assert_contains "$RUN_ERR" "missing value" "missing db error"
run_capture missing_auto "$SQ_BIN" create "missing auto commit value" --dolt-auto-commit
assert_eq "$RUN_CODE" "2" "missing auto commit value"
assert_contains "$RUN_ERR" "missing value" "missing auto commit error"

run_capture list_all "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list all after creates"
for id in "$BASIC_ID" "$FULL_ID" "$DEPS_ID" "$ACTOR_ID" "$HUMAN_ID"; do
  assert_contains "$RUN_OUT" "$id" "list should include created id $id"
done

echo "[create] PASS"
