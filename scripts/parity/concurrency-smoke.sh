#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"
WORKERS="${WORKERS:-16}"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/.parity-results}"

mkdir -p "$OUT_DIR"
TMP_DIR="$(mktemp -d -t sq-concurrency-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

DB_PATH="$TMP_DIR/tasks.sqlite"
export SQ_DB_PATH="$DB_PATH"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && make build >/dev/null)
fi

"$SQ_BIN" init --json >/dev/null

pids=()
for i in $(seq 1 "$WORKERS"); do
  (
    "$SQ_BIN" create "Concurrent task $i" --type task --priority 1 --description "worker-$i" --json >/dev/null
  ) &
  pids+=("$!")
done

fail=0
for pid in "${pids[@]}"; do
  if ! wait "$pid"; then
    fail=1
  fi
done

if [[ $fail -ne 0 ]]; then
  echo "[concurrency] one or more concurrent creates failed"
  exit 1
fi

LIST_JSON="$($SQ_BIN list --json --flat --no-pager)"
COUNT="$(python3 - <<PY
import json
x=json.loads('''$LIST_JSON''')
print(len(x) if isinstance(x,list) else 0)
PY
)"

if [[ "$COUNT" -lt "$WORKERS" ]]; then
  echo "[concurrency] expected at least $WORKERS tasks, got $COUNT"
  exit 1
fi

echo "[concurrency] PASS workers=$WORKERS count=$COUNT"
