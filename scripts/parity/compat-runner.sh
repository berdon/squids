#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PARITY_SCRIPT="$ROOT_DIR/scripts/parity/run-parity.sh"

BEADS_BIN="${BEADS_BIN:-bd}"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

OUT_DIR="${OUT_DIR:-$ROOT_DIR/.parity-results}"
mkdir -p "$OUT_DIR"

run_target() {
  local name="$1"
  local bin="$2"
  local out="$OUT_DIR/${name}.log"

  echo "[compat] running target=$name bin=$bin"
  set +e
  TARGET_BIN="$bin" "$PARITY_SCRIPT" >"$out" 2>&1
  local code=$?
  set -e

  echo "[compat] target=$name exit=$code log=$out"
  return $code
}

# Ensure sq exists (build if needed)
if [[ ! -x "$SQ_BIN" ]]; then
  echo "[compat] sq binary missing, building..."
  (cd "$ROOT_DIR" && make build >/dev/null)
fi

BEADS_OK=0
SQ_OK=0

if run_target "beads" "$BEADS_BIN"; then
  BEADS_OK=1
fi

if run_target "sq" "$SQ_BIN"; then
  SQ_OK=1
fi

echo ""
echo "[compat] summary"
echo "  beads: $BEADS_OK"
echo "  sq:    $SQ_OK"

if [[ $BEADS_OK -eq 1 && $SQ_OK -eq 1 ]]; then
  echo "[compat] both targets passed parity suite"
  exit 0
fi

echo "[compat] parity mismatch detected"
if [[ -f "$OUT_DIR/beads.log" && -f "$OUT_DIR/sq.log" ]]; then
  echo "[compat] diff (beads vs sq):"
  diff -u "$OUT_DIR/beads.log" "$OUT_DIR/sq.log" || true
fi

exit 1
