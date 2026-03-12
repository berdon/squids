#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  (cd "$ROOT_DIR" && go build -o bin/sq ./cmd/sq)
fi
SQ_BIN="$(cd "$(dirname "$SQ_BIN")" && pwd)/$(basename "$SQ_BIN")"
export SQ_BIN

mapfile -t scripts < <(find "$ROOT_DIR/tests/shell-automation" -mindepth 2 -maxdepth 2 -name test.sh | sort)

if [[ ${#scripts[@]} -eq 0 ]]; then
  echo "[shell-automation] no tests found"
  exit 0
fi

for script in "${scripts[@]}"; do
  echo "[shell-automation] running ${script#$ROOT_DIR/}"
  bash "$script"
done

echo "[shell-automation] PASS (${#scripts[@]} script(s))"
