#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
PYTHON="${PYTHON:-python3}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi

TMP_DIR="$(mktemp -d -t sq-audit-label-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

fail() {
  echo "[audit-label] FAIL: $*" >&2
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

assert_ne() {
  local got="$1"
  local want="$2"
  local context="$3"
  if [[ "$got" == "$want" ]]; then
    fail "$context: did not expect '$want'"
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

assert_file_lines() {
  local path="$1"
  local want="$2"
  local context="$3"
  local got
  got="$(wc -l < "$path" | tr -d ' ')"
  if [[ "$got" != "$want" ]]; then
    fail "$context: expected $want line(s), got $got"
  fi
}

assert_int_id_like() {
  local id="$1"
  local context="$2"
  [[ "$id" =~ ^int-[a-z0-9]+$ ]] || fail "$context: expected int-... id, got '$id'"
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

jsonl_last_field() {
  local path="$1"
  local field="$2"
  "$PYTHON" - "$path" "$field" <<'PY'
import json, pathlib, sys
lines = [line for line in pathlib.Path(sys.argv[1]).read_text().splitlines() if line.strip()]
obj = json.loads(lines[-1])
value = obj.get(sys.argv[2], "")
if isinstance(value, bool):
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

WORK_DIR="$TMP_DIR/work"
mkdir -p "$WORK_DIR/.beads"
cd "$WORK_DIR"

export SQ_DB_PATH="$WORK_DIR/tasks.sqlite"
INTERACTIONS_FILE="$WORK_DIR/.beads/interactions.jsonl"
printf '{"id":"int-seed","kind":"llm_call","created_at":"2026-01-01T00:00:00Z","actor":"seed","prompt":"hello","response":"world"}\n' > "$INTERACTIONS_FILE"

echo "[audit-label] binary=$SQ_BIN"
echo "[audit-label] db=$SQ_DB_PATH"

echo "[audit-label] workspace=$WORK_DIR"
run_capture init "$SQ_BIN" init --json
assert_eq "$RUN_CODE" "0" "sq init"
COUNT_BEFORE="$($SQ_BIN count --json)"
assert_file_lines "$INTERACTIONS_FILE" "1" "seed interactions log"

run_capture help_nested "$SQ_BIN" help audit label
assert_eq "$RUN_CODE" "0" "sq help audit label"
assert_contains "$RUN_OUT" "Append a label entry referencing an existing interaction" "nested help description"
assert_contains "$RUN_OUT" "Usage:" "nested help usage heading"
assert_contains "$RUN_OUT" "sq audit label <entry-id>" "nested help usage"
assert_contains "$RUN_OUT" "--label string" "nested help label flag"
assert_contains "$RUN_OUT" "--reason string" "nested help reason flag"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "nested help should not mutate sq state"
assert_file_lines "$INTERACTIONS_FILE" "1" "nested help should not mutate interactions log"

run_capture help_flag "$SQ_BIN" audit label --help
assert_eq "$RUN_CODE" "0" "sq audit label --help"
assert_contains "$RUN_OUT" "Append a label entry referencing an existing interaction" "flag help description"
assert_contains "$RUN_OUT" "Usage:" "flag help usage heading"
assert_contains "$RUN_OUT" "sq audit label <entry-id>" "flag help usage"
assert_contains "$RUN_OUT" "--label string" "flag help label flag"
assert_contains "$RUN_OUT" "--reason string" "flag help reason flag"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "--help should not mutate sq state"
assert_file_lines "$INTERACTIONS_FILE" "1" "--help should not mutate interactions log"

run_capture json_success "$SQ_BIN" audit label int-seed --label good --reason "verified by shell automation" --json
assert_eq "$RUN_CODE" "0" "sq audit label --json success"
JSON_ID="$(json_eval "$RUN_OUT" 'obj["id"]')"
assert_int_id_like "$JSON_ID" "json success id"
assert_eq "$(json_eval "$RUN_OUT" 'obj["label"]')" "good" "json success label"
assert_eq "$(json_eval "$RUN_OUT" 'obj["parent_id"]')" "int-seed" "json success parent id"
assert_file_lines "$INTERACTIONS_FILE" "2" "successful label append should grow interactions log"
assert_eq "$(jsonl_last_field "$INTERACTIONS_FILE" kind)" "label" "json success persisted kind"
assert_eq "$(jsonl_last_field "$INTERACTIONS_FILE" parent_id)" "int-seed" "json success persisted parent"
assert_eq "$(jsonl_last_field "$INTERACTIONS_FILE" label)" "good" "json success persisted label"
assert_eq "$(jsonl_last_field "$INTERACTIONS_FILE" reason)" "verified by shell automation" "json success persisted reason"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "successful label append should not mutate sq tasks"

run_capture human_success "$SQ_BIN" audit label int-seed --label bad --reason "human mode"
assert_eq "$RUN_CODE" "0" "sq audit label human success"
assert_not_contains "$RUN_OUT" '"id"' "human success should not emit raw JSON"
HUMAN_ID="$(printf '%s' "$RUN_OUT" | grep -o 'int-[a-z0-9]\+' | head -1)"
assert_int_id_like "$HUMAN_ID" "human success id"
assert_file_lines "$INTERACTIONS_FILE" "3" "human success should append interactions log"
assert_eq "$(jsonl_last_field "$INTERACTIONS_FILE" label)" "bad" "human success persisted label"
assert_eq "$(jsonl_last_field "$INTERACTIONS_FILE" parent_id)" "int-seed" "human success persisted parent"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "human success should not mutate sq tasks"

run_capture missing_id "$SQ_BIN" audit label --json
assert_ne "$RUN_CODE" "0" "sq audit label missing id should fail"
assert_contains "$RUN_ERR" "accepts 1 arg(s), received 0" "missing id error"
assert_file_lines "$INTERACTIONS_FILE" "3" "missing id should not append interactions log"

run_capture missing_label "$SQ_BIN" audit label int-seed --reason "missing label" --json
assert_ne "$RUN_CODE" "0" "sq audit label missing label should fail"
assert_contains "$RUN_ERR" "--label is required" "missing label error"
assert_file_lines "$INTERACTIONS_FILE" "3" "missing label should not append interactions log"

run_capture bad_flag "$SQ_BIN" audit label int-seed --label good --wat
assert_ne "$RUN_CODE" "0" "sq audit label unknown flag should fail"
assert_contains "$RUN_ERR" "unknown flag: --wat" "unknown flag error"
assert_file_lines "$INTERACTIONS_FILE" "3" "unknown flag should not append interactions log"
assert_eq "$($SQ_BIN count --json)" "$COUNT_BEFORE" "failure cases should not mutate sq tasks"

echo "[audit-label] PASS"
