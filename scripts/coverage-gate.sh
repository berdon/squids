#!/usr/bin/env bash
set -euo pipefail

MIN_COVERAGE="${1:-${COVERAGE_MIN:-90.0}}"

go test ./... -coverpkg=./... -coverprofile=coverage.out

TOTAL=$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%", "", $3); print $3}')
if [[ -z "${TOTAL}" ]]; then
  echo "Coverage gate failed: unable to parse total coverage" >&2
  exit 1
fi

echo "Total coverage: ${TOTAL}%"
awk -v c="${TOTAL}" -v min="${MIN_COVERAGE}" 'BEGIN {
  if (c+0 < min+0) {
    printf("Coverage gate failed: %.2f%% < %.2f%%\n", c, min);
    exit 1;
  }
}'
