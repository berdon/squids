#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-init-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[init] FAIL: $*" >&2
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

assert_file_exists() {
  local path="$1"
  local context="$2"
  [[ -f "$path" ]] || fail "$context: expected file at $path"
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

WORKSPACE="$TMP_DIR/workspace"
mkdir -p "$WORKSPACE/project"
cd "$WORKSPACE/project"
unset SQ_DB_PATH || true

echo "[init] workspace=$WORKSPACE/project"
echo "[init] binary=$SQ_BIN"

find . -maxdepth 3 -type f | sort > "$TMP_DIR/files-before.txt"
assert_eq "$(wc -l < "$TMP_DIR/files-before.txt" | tr -d ' ')" "0" "clean workspace should start without files"

run_capture basic_init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "basic init"
assert_eq "$(json_eval "$RUN_OUT" 'obj["command"]')" "init" "basic init command"
assert_eq "$(json_eval "$RUN_OUT" 'obj["ok"]')" "true" "basic init ok"
DB_PATH="$(json_eval "$RUN_OUT" 'obj["db_path"]')"
SCHEMA_VERSION="$(json_eval "$RUN_OUT" 'obj["schema_version"]')"
assert_eq "$DB_PATH" "$WORKSPACE/project/.sq/tasks.sqlite" "default db path"
assert_file_exists "$DB_PATH" "basic init db created"
[[ "$SCHEMA_VERSION" =~ ^[0-9]+$ ]] || fail "schema version should be numeric"
[[ "$SCHEMA_VERSION" -gt 0 ]] || fail "schema version should be positive"
assert_file_exists "$WORKSPACE/project/.sq/tasks.sqlite" "default sq dir created"

run_capture ready_after_init "$SQ_BIN" ready --json
assert_eq "$RUN_CODE" "0" "ready after init"
assert_eq "$RUN_OUT" "[]" "ready should be empty after init"

run_capture init_repeat "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "repeated init"
assert_eq "$(json_eval "$RUN_OUT" 'obj["db_path"]')" "$DB_PATH" "repeated init db path stable"
assert_eq "$(json_eval "$RUN_OUT" 'obj["schema_version"]')" "$SCHEMA_VERSION" "repeated init schema stable"

run_capture create_after_init "$SQ_BIN" create "post-init smoke" --type task --priority 2 --json
assert_eq "$RUN_CODE" "0" "create after init"
TASK_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
run_capture list_after_init "$SQ_BIN" list --json
assert_eq "$RUN_CODE" "0" "list after init"
assert_contains "$RUN_OUT" "$TASK_ID" "list should show created task"

mkdir -p "$WORKSPACE/env-db"
cd "$WORKSPACE/env-db"
export SQ_DB_PATH="$WORKSPACE/custom/location/tasks.sqlite"
run_capture env_init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "env db init"
assert_eq "$(json_eval "$RUN_OUT" 'obj["db_path"]')" "$SQ_DB_PATH" "env db path honored"
assert_file_exists "$SQ_DB_PATH" "env db created"
run_capture env_ready "$SQ_BIN" ready --json
assert_eq "$RUN_CODE" "0" "ready with env db"
assert_eq "$RUN_OUT" "[]" "ready after env init"

REPO_ROOT_STATE_DIR="$ROOT_DIR/.sq"
REPO_ROOT_STATE_CREATED="false"
if [[ ! -d "$REPO_ROOT_STATE_DIR" ]]; then
  REPO_ROOT_STATE_CREATED="true"
fi
unset SQ_DB_PATH || true
cd "$ROOT_DIR"
run_capture repo_root_init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "repo root init"
REPO_DB_PATH="$(json_eval "$RUN_OUT" 'obj["db_path"]')"
assert_contains "$REPO_DB_PATH" "$ROOT_DIR/.sq/" "repo root db path"
assert_file_exists "$REPO_DB_PATH" "repo root db exists"
run_capture repo_root_status "$SQ_BIN" status --json
assert_eq "$RUN_CODE" "0" "repo root status after init"
assert_contains "$RUN_OUT" '"summary"' "repo root status json"
if [[ "$REPO_ROOT_STATE_CREATED" == "true" ]]; then
  rm -rf "$REPO_ROOT_STATE_DIR"
fi

cd "$WORKSPACE/project"
unset SQ_DB_PATH || true
COUNT_BEFORE_HELP="$($SQ_BIN count --json)"
run_capture init_help_flag "$SQ_BIN" init --help
assert_eq "$RUN_CODE" "0" "init --help should succeed"
assert_contains "$RUN_OUT" "init" "init --help should describe command"
assert_contains "$RUN_OUT" "Usage:" "init --help should include usage"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE_HELP" "init --help should not mutate state"

run_capture help_init "$SQ_BIN" help init
assert_eq "$RUN_CODE" "0" "help init should succeed"
assert_contains "$RUN_OUT" "init" "help init output"

run_capture extra_args "$SQ_BIN" init extra --json
assert_eq "$RUN_CODE" "2" "init extra positional args should fail"
assert_contains "$RUN_ERR" "does not accept positional arguments" "init extra arg error"

BAD_ROOT="$TMP_DIR/bad-root"
mkdir -p "$BAD_ROOT"
printf 'x' > "$BAD_ROOT/notadir"
export SQ_DB_PATH="$BAD_ROOT/notadir/tasks.sqlite"
run_capture bad_path "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "1" "bad path init should fail"
assert_contains "$RUN_ERR" "create db directory" "bad path error message"

unset SQ_DB_PATH || true

echo "[init] PASS"
