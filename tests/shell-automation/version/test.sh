#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SQ_BIN="${SQ_BIN:-$ROOT_DIR/bin/sq}"

if [[ ! -x "$SQ_BIN" ]]; then
  echo "[version] sq binary not found at $SQ_BIN (run make build first)" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d -t sq-version-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "[version] ASSERT FAILED: expected output to contain: $needle" >&2
    echo "[version] ----- output -----" >&2
    printf '%s\n' "$haystack" >&2
    exit 1
  fi
}

assert_eq() {
  local got="$1"
  local want="$2"
  if [[ "$got" != "$want" ]]; then
    echo "[version] ASSERT FAILED: expected '$want' got '$got'" >&2
    exit 1
  fi
}

json_field() {
  local json="$1"
  local field="$2"
  python3 - <<PY
import json
obj=json.loads('''$json''')
val=obj
for part in "$field".split('.'):
    val=val[part]
print(val if not isinstance(val, (dict, list)) else json.dumps(val))
PY
}

run_expect_ok() {
  local label="$1"
  shift
  local out
  if ! out="$($SQ_BIN "$@" 2>&1)"; then
    echo "[version] command failed unexpectedly ($label): $*" >&2
    printf '%s\n' "$out" >&2
    exit 1
  fi
  printf '%s' "$out"
}

run_expect_fail() {
  local label="$1"
  shift
  local out
  set +e
  out="$($SQ_BIN "$@" 2>&1)"
  local code=$?
  set -e
  if [[ $code -eq 0 ]]; then
    echo "[version] command succeeded unexpectedly ($label): $*" >&2
    printf '%s\n' "$out" >&2
    exit 1
  fi
  printf '%s' "$out"
}

pushd "$TMP_DIR" >/dev/null

echo "[version] workspace=$TMP_DIR"
echo "[version] binary=$SQ_BIN"

run_expect_ok init init --json >/dev/null
count_before="$(run_expect_ok count-before count --json)"
status_before="$(run_expect_ok status-before status --json)"

help_out="$(run_expect_ok help-version help version)"
assert_contains "$help_out" "Print version information"
assert_contains "$help_out" "sq version [flags]"
assert_contains "$help_out" "--dolt-auto-commit"

help_flag_out="$(run_expect_ok version-help version --help)"
assert_contains "$help_flag_out" "Print version information"
assert_contains "$help_flag_out" "sq version [flags]"
assert_contains "$help_flag_out" "--dolt-auto-commit"
assert_eq "$help_out" "$help_flag_out"

version_out="$(run_expect_ok version-default version)"
assert_contains "$version_out" "sq version"
assert_contains "$version_out" "(source)"

json_out="$(run_expect_ok version-json version --json)"
assert_eq "$(json_field "$json_out" version)" "dev"
assert_eq "$(json_field "$json_out" build)" "source"
assert_eq "$(json_field "$json_out" branch)" "vdev"

quiet_out="$(run_expect_ok version-quiet version --quiet)"
verbose_out="$(run_expect_ok version-verbose version --verbose)"
profile_out="$(run_expect_ok version-profile version --profile)"
readonly_out="$(run_expect_ok version-readonly version --readonly)"
sandbox_out="$(run_expect_ok version-sandbox version --sandbox)"
actor_out="$(run_expect_ok version-actor version --actor tester)"
db_out="$(run_expect_ok version-db version --db "$TMP_DIR/custom.sqlite")"
dolt_out="$(run_expect_ok version-dolt version --dolt-auto-commit off)"
short_alias_out="$(run_expect_ok version-short-alias -V)"
long_alias_out="$(run_expect_ok version-long-alias --version)"

assert_eq "$quiet_out" "$version_out"
assert_eq "$verbose_out" "$version_out"
assert_eq "$profile_out" "$version_out"
assert_eq "$readonly_out" "$version_out"
assert_eq "$sandbox_out" "$version_out"
assert_eq "$actor_out" "$version_out"
assert_eq "$db_out" "$version_out"
assert_eq "$dolt_out" "$version_out"
assert_eq "$short_alias_out" "$version_out"
assert_eq "$long_alias_out" "$version_out"

bad_flag_out="$(run_expect_fail version-bad-flag version --bogus-flag)"
assert_contains "$bad_flag_out" "unknown flag: --bogus-flag"

count_after="$(run_expect_ok count-after count --json)"
status_after="$(run_expect_ok status-after status --json)"
assert_eq "$count_after" "$count_before"
assert_eq "$status_after" "$status_before"

popd >/dev/null

echo "[version] PASS"
