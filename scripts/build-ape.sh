#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
mkdir -p dist

OUT_APE="dist/sq-ape"
NOTE_README="dist/sq-ape.README.txt"

rm -f "$OUT_APE" "$NOTE_README"

CACHE_DIR="${COSMOCC_CACHE_DIR:-$HOME/.cache/squids-cosmocc}"

ensure_cosmocc_toolchain() {
  local need_install=0
  if ! command -v cosmocc >/dev/null 2>&1; then
    need_install=1
  fi
  if ! command -v apelink >/dev/null 2>&1; then
    need_install=1
  fi
  if [[ $need_install -eq 0 ]]; then
    return 0
  fi

  echo "[ape] cosmocc/apelink not found in PATH; bootstrapping into $CACHE_DIR"
  mkdir -p "$CACHE_DIR"
  local zip="$CACHE_DIR/cosmocc.zip"
  curl -fsSL -o "$zip" https://cosmo.zip/pub/cosmocc/cosmocc.zip
  python3 -c "import zipfile; zipfile.ZipFile('$zip').extractall('$CACHE_DIR')"
  chmod +x "$CACHE_DIR/bin/ape-x86_64.elf" "$CACHE_DIR/bin/cosmocc" "$CACHE_DIR/bin/apelink" || true
  if [[ -f "$CACHE_DIR/bin/ape-x86_64.elf" && ! -x "$CACHE_DIR/bin/ape" ]]; then
    install -m 0755 "$CACHE_DIR/bin/ape-x86_64.elf" "$CACHE_DIR/bin/ape"
  fi
  export PATH="$CACHE_DIR/bin:$PATH"
}

ensure_cosmocc_toolchain

if command -v cosmocc >/dev/null 2>&1 && command -v apelink >/dev/null 2>&1; then
  echo "[ape] cosmocc+apelink detected; building APE-compatible artifact"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/sq-linux-amd64 ./cmd/sq
  # Go ELF binaries don't expose the full set of symbols needed for multi-OS APE vectors.
  # Build the portable-ELF variant through apelink so the pipeline is deterministic.
  apelink -V linux -o "$OUT_APE" dist/sq-linux-amd64
  chmod +x "$OUT_APE"
  cat > "$NOTE_README" <<'TXT'
This artifact was produced using Cosmopolitan tooling via `apelink -V linux`.
It is intended for Linux hosts while preserving the APE-oriented build pipeline.

For macOS/Windows use the platform-native release binaries.
TXT
  echo "[ape] built: $OUT_APE"
else
  echo "[ape] ERROR: cosmocc/apelink unavailable; refusing fallback build" >&2
  echo "Install cosmocc (with apelink) before running scripts/build-ape.sh" >&2
  exit 1
fi
