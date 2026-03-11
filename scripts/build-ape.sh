#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
mkdir -p dist

OUT_APE="dist/sq-ape"
OUT_FALLBACK="dist/sq-ape-fallback-linux-amd64"
NOTE_UNAVAILABLE="dist/sq-ape.UNAVAILABLE.txt"
NOTE_README="dist/sq-ape.README.txt"

rm -f "$OUT_APE" "$OUT_FALLBACK" "$NOTE_UNAVAILABLE" "$NOTE_README"

if command -v cosmocc >/dev/null 2>&1 && command -v ape >/dev/null 2>&1; then
  echo "[ape] cosmocc+ape detected; building APE binary"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/sq-linux-amd64 ./cmd/sq
  ape -o "$OUT_APE" dist/sq-linux-amd64
  chmod +x "$OUT_APE"
  cat > "$NOTE_README" <<'TXT'
This artifact was built with Cosmopolitan/APE tooling.
Run directly on supported systems:
  ./sq-ape

If your kernel/host cannot run APE binaries, use the platform-native artifacts
or the fallback linux binary in this release.
TXT
  echo "[ape] built: $OUT_APE"
else
  echo "[ape] cosmocc/ape unavailable; creating fallback artifact"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT_FALLBACK" ./cmd/sq
  cat > "$NOTE_UNAVAILABLE" <<'TXT'
APE build was skipped because Cosmopolitan tooling (cosmocc + ape) was not available
in this build environment.

Fallback strategy:
- Use sq-linux-amd64 from this release.
- Or run scripts/build-ape.sh in an environment with cosmocc + ape installed.
TXT
fi
