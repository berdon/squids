#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-import-beads-shell-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[import-beads] FAIL: $*" >&2
  exit 1
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    fail "expected output to contain '$needle' but got: $haystack"
  fi
}

assert_eq() {
  local got="$1"
  local want="$2"
  if [[ "$got" != "$want" ]]; then
    fail "expected '$want' but got '$got'"
  fi
}

json_query() {
  local json="$1"
  local expr="$2"
  JSON_INPUT="$json" "$PYTHON" - "$expr" <<'PY'
import json, os, sys
obj = json.loads(os.environ["JSON_INPUT"])
expr = sys.argv[1]
value = eval(expr, {"__builtins__": {}}, {"obj": obj, "len": len})
if isinstance(value, (dict, list)):
    print(json.dumps(value))
elif isinstance(value, bool):
    print("true" if value else "false")
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
  local code=$?
  set -e
  RUN_CODE=$code
  RUN_OUT="$(<"$out_file")"
  RUN_ERR="$(<"$err_file")"
}

seed_source_db() {
  local db_path="$1"
  SQ_DB_PATH="$db_path" "$SQ_BIN" init --json >/dev/null
  SOURCE_DB="$db_path" "$PYTHON" <<'PY'
import os, sqlite3

db_path = os.environ["SOURCE_DB"]
conn = sqlite3.connect(db_path)
cur = conn.cursor()
cur.execute(
    """
    INSERT INTO tasks(
      id,title,description,status,priority,issue_type,assignee,owner,
      labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at
    ) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
    """,
    (
      "bd-1", "Imported task", "imported description", "open", 1, "task", "alice", "alice",
      '["triage","ops"]', '[]', '{"k":"v"}', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z', ''
    ),
)
cur.execute(
    """
    INSERT INTO tasks(
      id,title,description,status,priority,issue_type,assignee,owner,
      labels_json,deps_json,metadata_json,close_reason,created_at,updated_at,closed_at
    ) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
    """,
    (
      "bd-2", "Blocking task", "secondary task", "open", 2, "bug", "", "",
      '[]', '[]', '{}', '', '2026-01-02T00:00:00Z', '2026-01-02T00:00:00Z', ''
    ),
)
cur.execute(
    "INSERT INTO dependencies(issue_id, depends_on_id, dep_type) VALUES(?,?,?)",
    ("bd-1", "bd-2", "blocks"),
)
cur.execute(
    "INSERT INTO comments(issue_id, author, body, created_at) VALUES(?,?,?,?)",
    ("bd-1", "alice", "hello from beads", "2026-01-01T00:00:01Z"),
)
conn.commit()
conn.close()
PY
}

make_invalid_source_db() {
  local db_path="$1"
  BAD_DB="$db_path" "$PYTHON" <<'PY'
import os, sqlite3
conn = sqlite3.connect(os.environ["BAD_DB"])
conn.execute("CREATE TABLE nope (id TEXT)")
conn.commit()
conn.close()
PY
}

list_count() {
  local db_path="$1"
  local json
  json="$(SQ_DB_PATH="$db_path" "$SQ_BIN" list --json --flat --no-pager)"
  json_query "$json" 'len(obj)'
}

echo "[import-beads] tmp=$TMP_DIR"
echo "[import-beads] sq_bin=$SQ_BIN"

SOURCE_DB="$TMP_DIR/source.sqlite"
seed_source_db "$SOURCE_DB"

# 1) Help / discovery surface
HELP_OUT="$($SQ_BIN help import-beads)"
assert_contains "$HELP_OUT" "Import tasks, dependencies, and comments from a beads sqlite database."
assert_contains "$HELP_OUT" "sq import-beads [flags]"
assert_contains "$HELP_OUT" "--source string"

run_capture help_flag "$SQ_BIN" import-beads --help
assert_eq "$RUN_CODE" "0"
assert_contains "$RUN_OUT" "Import tasks, dependencies, and comments from a beads sqlite database."
assert_contains "$RUN_OUT" "sq import-beads [flags]"
assert_contains "$RUN_OUT" "--dry-run"
assert_contains "$RUN_OUT" "--no-comments"

# 2) Baseline success path with explicit source
TARGET_DB="$TMP_DIR/target.sqlite"
SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" init --json >/dev/null
run_capture explicit_human env SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" import-beads --source "$SOURCE_DB"
assert_eq "$RUN_CODE" "0"
assert_contains "$RUN_OUT" "Imported from: $SOURCE_DB"
assert_contains "$RUN_OUT" "Tasks: created=2 updated=0 unchanged=0"
assert_contains "$RUN_OUT" "Deps: created=1 unchanged=0"
assert_contains "$RUN_OUT" "Comments: created=1 unchanged=0"
assert_contains "$RUN_OUT" "Dry-run: false"

SHOW_JSON="$(SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" show bd-1 --json)"
assert_eq "$(json_query "$SHOW_JSON" 'obj["id"]')" "bd-1"
assert_eq "$(json_query "$SHOW_JSON" 'obj["title"]')" "Imported task"
assert_eq "$(json_query "$SHOW_JSON" 'obj["assignee"]')" "alice"
COMMENTS_JSON="$(SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" comments bd-1 --json)"
assert_eq "$(json_query "$COMMENTS_JSON" 'len(obj)')" "1"
assert_eq "$(json_query "$COMMENTS_JSON" 'obj[0]["body"]')" "hello from beads"
DEP_JSON="$(SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" dep list bd-1 --json)"
assert_eq "$(json_query "$DEP_JSON" 'obj[0]')" "bd-2"
assert_eq "$(list_count "$TARGET_DB")" "2"

# 3) JSON path + idempotency
REPORT_JSON="$(SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" import-beads --source "$SOURCE_DB" --json)"
assert_eq "$(json_query "$REPORT_JSON" 'obj["tasks"]["updated"]')" "2"
assert_eq "$(json_query "$REPORT_JSON" 'obj["deps"]["unchanged"]')" "1"
assert_eq "$(json_query "$REPORT_JSON" 'obj["comments"]["unchanged"]')" "1"
assert_eq "$(list_count "$TARGET_DB")" "2"
COMMENTS_JSON="$(SQ_DB_PATH="$TARGET_DB" "$SQ_BIN" comments bd-1 --json)"
assert_eq "$(json_query "$COMMENTS_JSON" 'len(obj)')" "1"

# 4) Dry-run path should not write
DRY_TARGET_DB="$TMP_DIR/dry-target.sqlite"
SQ_DB_PATH="$DRY_TARGET_DB" "$SQ_BIN" init --json >/dev/null
DRY_JSON="$(SQ_DB_PATH="$DRY_TARGET_DB" "$SQ_BIN" import-beads --source "$SOURCE_DB" --dry-run --json)"
assert_eq "$(json_query "$DRY_JSON" 'obj["dry_run"]')" "true"
assert_eq "$(json_query "$DRY_JSON" 'obj["tasks"]["created"]')" "2"
assert_eq "$(list_count "$DRY_TARGET_DB")" "0"

# 5) Tuple-specific flags: --no-comments and --no-events
NO_COMMENTS_DB="$TMP_DIR/no-comments.sqlite"
SQ_DB_PATH="$NO_COMMENTS_DB" "$SQ_BIN" init --json >/dev/null
NO_COMMENTS_JSON="$(SQ_DB_PATH="$NO_COMMENTS_DB" "$SQ_BIN" import-beads --source "$SOURCE_DB" --no-comments --no-events --json)"
assert_eq "$(json_query "$NO_COMMENTS_JSON" 'obj["comments"]["created"]')" "0"
COMMENTS_JSON="$(SQ_DB_PATH="$NO_COMMENTS_DB" "$SQ_BIN" comments bd-1 --json)"
assert_eq "$(json_query "$COMMENTS_JSON" 'len(obj)')" "0"
assert_eq "$(list_count "$NO_COMMENTS_DB")" "2"

# 6) Source discovery via env and .beads directory
ENV_TARGET_DB="$TMP_DIR/env-target.sqlite"
SQ_DB_PATH="$ENV_TARGET_DB" "$SQ_BIN" init --json >/dev/null
ENV_JSON="$(SQ_DB_PATH="$ENV_TARGET_DB" BEADS_DATABASE="$SOURCE_DB" "$SQ_BIN" import-beads --json)"
assert_eq "$(json_query "$ENV_JSON" 'obj["source"]')" "$SOURCE_DB"
assert_eq "$(list_count "$ENV_TARGET_DB")" "2"

DISCOVERY_WORKDIR="$TMP_DIR/discovery-workdir"
mkdir -p "$DISCOVERY_WORKDIR/.beads"
cp -f "$SOURCE_DB" "$DISCOVERY_WORKDIR/.beads/tasks.sqlite"
DISCOVERY_TARGET_DB="$DISCOVERY_WORKDIR/target.sqlite"
SQ_DB_PATH="$DISCOVERY_TARGET_DB" "$SQ_BIN" init --json >/dev/null
DISCOVERY_JSON="$(cd "$DISCOVERY_WORKDIR" && SQ_DB_PATH="$DISCOVERY_TARGET_DB" "$SQ_BIN" import-beads --json)"
assert_contains "$DISCOVERY_JSON" "$DISCOVERY_WORKDIR/.beads/tasks.sqlite"
assert_eq "$(list_count "$DISCOVERY_TARGET_DB")" "2"

# 7) Input validation / failure behavior
MISSING_TARGET_DB="$TMP_DIR/missing-source-target.sqlite"
SQ_DB_PATH="$MISSING_TARGET_DB" "$SQ_BIN" init --json >/dev/null
run_capture missing_source env -u BEADS_DATABASE -u BEADS_DIR "$SQ_BIN" import-beads --json
assert_eq "$RUN_CODE" "2"
assert_contains "$RUN_ERR" "unable to discover beads source database"
assert_eq "$(list_count "$MISSING_TARGET_DB")" "0"

run_capture unknown_flag "$SQ_BIN" import-beads --wat
assert_eq "$RUN_CODE" "2"
assert_contains "$RUN_ERR" "unknown flag: --wat"

BAD_PATH_TARGET_DB="$TMP_DIR/bad-path-target.sqlite"
SQ_DB_PATH="$BAD_PATH_TARGET_DB" "$SQ_BIN" init --json >/dev/null
run_capture bad_path env SQ_DB_PATH="$BAD_PATH_TARGET_DB" "$SQ_BIN" import-beads --source "$TMP_DIR/does-not-exist.sqlite" --json
assert_eq "$RUN_CODE" "2"
assert_contains "$RUN_ERR" "does-not-exist.sqlite"
assert_eq "$(list_count "$BAD_PATH_TARGET_DB")" "0"

INVALID_SOURCE_DB="$TMP_DIR/invalid-source.sqlite"
make_invalid_source_db "$INVALID_SOURCE_DB"
INVALID_TARGET_DB="$TMP_DIR/invalid-target.sqlite"
SQ_DB_PATH="$INVALID_TARGET_DB" "$SQ_BIN" init --json >/dev/null
run_capture invalid_schema env SQ_DB_PATH="$INVALID_TARGET_DB" "$SQ_BIN" import-beads --source "$INVALID_SOURCE_DB" --json
assert_eq "$RUN_CODE" "2"
assert_contains "$RUN_ERR" "source validation failed"
assert_eq "$(list_count "$INVALID_TARGET_DB")" "0"

echo "[import-beads] PASS"
