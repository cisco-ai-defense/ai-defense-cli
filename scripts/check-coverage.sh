#!/usr/bin/env bash
# Enforce minimum Go statement coverage from a coverprofile produced by make test-cover.
set -euo pipefail

MIN="${COVER_MIN:?COVER_MIN must be set}"
PROFILE="${1:-coverage.out}"

if [[ ! -f "$PROFILE" ]]; then
  echo "coverage profile not found: $PROFILE (run make test-cover first)" >&2
  exit 1
fi

total="$(go tool cover -func="$PROFILE" | awk '/^total:/ {gsub(/%/,"",$3); print $3}')"
if [[ -z "$total" ]]; then
  echo "failed to parse total coverage from $PROFILE" >&2
  exit 1
fi

echo "total coverage: ${total}% (minimum ${MIN}%)"

awk -v got="$total" -v min="$MIN" 'BEGIN { exit !(got+0 >= min+0) }' || {
  echo "coverage ${total}% is below minimum ${MIN}%" >&2
  exit 1
}
