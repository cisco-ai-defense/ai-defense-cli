#!/usr/bin/env bash
# Build relocatable CPython + cisco-aibom[all] and stage embed artifacts for Go.
#
# Output (gitignored):
#   internal/embedded/python/bin/aid-py-bundle.tar.gz
#   internal/embedded/python/bin/aid-py-bundle.id
#
# Requires: curl, tar, gzip, sha256sum (or shasum), python3 with pip (build host only)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT}/internal/embedded/python/bin"
REQ_FILE="${ROOT}/scripts/requirements-bundle.txt"
STAGING="${ROOT}/.build/aid-py-bundle"

# Pinned python-build-standalone (https://github.com/astral-sh/python-build-standalone)
PBS_RELEASE="${PBS_RELEASE:-20250723}"
PYTHON_VERSION="${PYTHON_VERSION:-3.12.11}"
case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  PBS_TRIPLE="${PBS_TRIPLE:-x86_64-unknown-linux-gnu}" ;;
  Linux-aarch64|Linux-arm64) PBS_TRIPLE="${PBS_TRIPLE:-aarch64-unknown-linux-gnu}" ;;
  Darwin-x86_64) PBS_TRIPLE="${PBS_TRIPLE:-x86_64-apple-darwin}" ;;
  Darwin-arm64)  PBS_TRIPLE="${PBS_TRIPLE:-aarch64-apple-darwin}" ;;
  *)
    echo "Unsupported platform for bundle build: $(uname -s) $(uname -m)" >&2
    echo "Set PBS_TRIPLE and PBS_SHA256 for your python-build-standalone asset." >&2
    exit 1
    ;;
esac

# Pinned SHA256 per PBS asset (cpython 3.12.11 + release 20250723, install_only_stripped); override with PBS_SHA256.
if [[ -z "${PBS_SHA256:-}" ]]; then
  case "${PBS_TRIPLE}" in
    x86_64-unknown-linux-gnu)   PBS_SHA256=7263ed87e9d2d4c52b01f1ad8cdca315cf859c0848b8ffdcb70de205fa8b0336 ;;
    aarch64-unknown-linux-gnu)  PBS_SHA256=32b17b62fd621481bb4eab0ed5549cbad43d88780e29fc2c0a222a1722470c4b ;;
    x86_64-apple-darwin)        PBS_SHA256=de7786cb936a0ae87ee8b89906ba7a5e02b2880d8665f7ba5d3ecff5f5e14d1f ;;
    aarch64-apple-darwin)       PBS_SHA256=ba814d8d611898aee3c6daed53a535a93345a6b4984ebe62f9aa38646255a5bf ;;
    *)
      echo "No default PBS_SHA256 for PBS_TRIPLE=${PBS_TRIPLE}; set PBS_SHA256 explicitly." >&2
      exit 1
      ;;
  esac
fi

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    shasum -a 256 "$@"
  fi
}

if [[ ! -f "${REQ_FILE}" ]]; then
  echo "Missing ${REQ_FILE}" >&2
  exit 1
fi

mkdir -p "${OUT_DIR}" "${STAGING}"

asset="cpython-${PYTHON_VERSION}+${PBS_RELEASE}-${PBS_TRIPLE}-install_only_stripped.tar.gz"
base="https://github.com/astral-sh/python-build-standalone/releases/download/${PBS_RELEASE}"
archive="${STAGING}/${asset}"

echo "Downloading relocatable CPython: ${asset}"
curl -fsSL --retry 5 --retry-all-errors -o "${archive}" "${base}/${asset}"
echo "${PBS_SHA256}  ${archive}" | sha256_file -c -

rm -rf "${STAGING}/python"
mkdir -p "${STAGING}"
tar -xzf "${archive}" -C "${STAGING}"

PYBIN="${STAGING}/python/bin/python3"
if [[ ! -x "${PYBIN}" ]]; then
  echo "Expected interpreter at ${PYBIN}" >&2
  exit 1
fi

echo "Installing Python deps from ${REQ_FILE}"
"${PYBIN}" -m pip install --no-warn-script-location --no-cache-dir \
  --timeout 60 --retries 10 \
  -r "${REQ_FILE}"

bash "${ROOT}/scripts/prune-python-bundle.sh" "${STAGING}/python"

bundle_tar="${OUT_DIR}/aid-py-bundle.tar.gz"
bundle_id="${OUT_DIR}/aid-py-bundle.id"

tar -czf "${bundle_tar}" -C "${STAGING}" python
sha256_file "${bundle_tar}" | cut -c1-16 > "${bundle_id}"

echo "Wrote ${bundle_tar} ($(wc -c < "${bundle_tar}") bytes)"
echo "Wrote ${bundle_id} (id: $(cat "${bundle_id}"))"
