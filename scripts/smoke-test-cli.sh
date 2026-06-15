#!/usr/bin/env bash
# Post-build smoke tests: --version and --help for every registered command.
#
# Usage:
#   smoke-test-cli.sh <binary-path>
#
# Environment:
#   SMOKE_RUNNER=direct|docker   How to invoke the binary (default: direct).
#   SMOKE_DOCKER_IMAGE           Image for docker runner (default: debian:bookworm-slim).
#   VERSION                      When set, --version output must contain this string.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${1:?usage: smoke-test-cli.sh <binary-path>}"
RUNNER="${SMOKE_RUNNER:-direct}"
DOCKER_IMAGE="${SMOKE_DOCKER_IMAGE:-debian:bookworm-slim}"

if [[ ! -f "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi
BIN="$(cd "$(dirname "$BIN")" && pwd)/$(basename "$BIN")"
chmod +x "$BIN"

run_cli() {
  if [[ "$RUNNER" == docker ]]; then
    docker run --rm --entrypoint /cli -v "${BIN}:/cli:ro" "$DOCKER_IMAGE" "$@"
  else
    "$BIN" "$@"
  fi
}

echo "smoke: --version"
version_out="$(run_cli --version)"
if [[ -z "$version_out" ]]; then
  echo "smoke: --version produced no output" >&2
  exit 1
fi
if [[ -n "${VERSION:-}" ]] && ! grep -Fq "${VERSION}" <<<"$version_out"; then
  echo "smoke: --version missing VERSION=${VERSION}: $version_out" >&2
  exit 1
fi

commands_file="$(mktemp)"
trap 'rm -f "$commands_file"' EXIT
(cd "$ROOT" && go run ./scripts/list-smoke-commands) >"$commands_file"
if [[ ! -s "$commands_file" ]]; then
  echo "smoke: no command paths from list-smoke-commands" >&2
  exit 1
fi

count=0
while IFS= read -r path || [[ -n "$path" ]]; do
  count=$((count + 1))
  label="${path:-root}"
  echo "smoke: ${label} --help"
  if [[ -z "$path" ]]; then
    help_out="$(run_cli --help)"
  else
    # shellcheck disable=SC2086
    help_out="$(run_cli ${path} --help)"
  fi
  if [[ -z "$help_out" ]]; then
    echo "smoke: ${label} --help produced no output" >&2
    exit 1
  fi
  if grep -Fq "Error:" <<<"$help_out"; then
    echo "smoke: ${label} --help contains Error:" >&2
    exit 1
  fi
done <"$commands_file"

echo "smoke: OK (${count} commands + version)"
