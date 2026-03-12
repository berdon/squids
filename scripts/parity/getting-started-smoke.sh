#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  echo "[getting-started] sq binary not found at $SQ_BIN (run make build first)" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

extract_id() {
  grep -oE '"id"[[:space:]]*:[[:space:]]*"[^"]+"' | head -1 | cut -d'"' -f4
}

run_json() {
  local out
  out="$($SQ_BIN "$@")"
  echo "$out" >/dev/null
  printf '%s' "$out"
}

echo "[getting-started] workspace=$TMP_DIR"

pushd "$TMP_DIR" >/dev/null

# 2) Initialize workspace
run_json init --json >/dev/null

# 3) ready
run_json ready --json >/dev/null

# 4) Basic lifecycle
issue_json="$(run_json create "Ship migration docs" --type task --priority 1 --description "Document sq rollout" --json)"
ISSUE_ID="$(printf '%s' "$issue_json" | extract_id)"
if [[ -z "$ISSUE_ID" ]]; then
  echo "[getting-started] failed to parse issue id" >&2
  exit 1
fi

run_json list --json --flat --no-pager >/dev/null
run_json show "$ISSUE_ID" --json >/dev/null
run_json update "$ISSUE_ID" --status in_progress --assignee guppy --json >/dev/null
run_json close "$ISSUE_ID" --reason "Done" --json >/dev/null

# 5) Command families
run_json label add "$ISSUE_ID" triage --json >/dev/null
run_json label list "$ISSUE_ID" --json >/dev/null

a_json="$(run_json create "Dependency A" --type task --priority 2 --json)"
b_json="$(run_json create "Dependency B" --type task --priority 2 --json)"
A_ID="$(printf '%s' "$a_json" | extract_id)"
B_ID="$(printf '%s' "$b_json" | extract_id)"
run_json dep add "$A_ID" "$B_ID" --json >/dev/null
run_json dep list "$A_ID" --json >/dev/null

run_json comments add "$A_ID" "needs review" --json >/dev/null
run_json comments "$A_ID" --json >/dev/null

todo_json="$(run_json todo add "Follow up" --json)"
TODO_ID="$(printf '%s' "$todo_json" | extract_id)"
run_json todo --json >/dev/null
run_json todo done "$TODO_ID" --reason "Completed" --json >/dev/null

# 6) Reports and triage
run_json blocked --json >/dev/null
run_json stale --days 30 --json >/dev/null
run_json orphans --json >/dev/null
run_json query "status=open AND priority<=2" --json >/dev/null
run_json status --json >/dev/null

# 7) Environment override
SQ_DB_PATH="$TMP_DIR/override.sqlite" run_json init --json >/dev/null
SQ_DB_PATH="$TMP_DIR/override.sqlite" run_json list --json >/dev/null

popd >/dev/null

echo "[getting-started] ok"
