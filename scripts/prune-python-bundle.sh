#!/usr/bin/env bash
# Copyright 2026 Cisco Systems, Inc. and its affiliates
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

# Remove install-time cruft from a staged relocatable CPython tree before tar embed.
# Usage: prune-python-bundle.sh <python-root>
#   <python-root> is the directory containing bin/ and lib/ (e.g. .../python).
#
# Shared by Dockerfile py-bundler and scripts/build-python-bundle.sh.

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <python-root>" >&2
  exit 1
fi

PY_ROOT="$1"
if [[ ! -d "${PY_ROOT}/lib" ]]; then
  echo "expected Python lib/ under ${PY_ROOT}" >&2
  exit 1
fi

shopt -s nullglob
sites=("${PY_ROOT}"/lib/python*/site-packages)
if [[ ${#sites[@]} -eq 0 ]]; then
  echo "no site-packages under ${PY_ROOT}/lib" >&2
  exit 1
fi

# Bytecode and caches.
find "${PY_ROOT}" -depth -type d -name '__pycache__' -exec rm -rf {} + 2>/dev/null || true
find "${PY_ROOT}" -type f \( -name '*.pyc' -o -name '*.pyo' \) -delete 2>/dev/null || true

# Installers not needed at runtime (pip already ran).
for site in "${sites[@]}"; do
  for pkg in pip setuptools wheel _distutils_hack; do
    rm -rf "${site}/${pkg}" 2>/dev/null || true
  done
done

# Test trees inside installed packages only.
for site in "${sites[@]}"; do
  find "${site}" -depth -type d \( -name tests -o -name test -o -name testing \) \
    -exec rm -rf {} + 2>/dev/null || true
done

# Strip debug symbols from native extensions (Linux release builds).
if [[ "$(uname -s)" == Linux ]] && command -v strip >/dev/null 2>&1; then
  find "${PY_ROOT}" -name '*.so' -exec strip --strip-unneeded {} + 2>/dev/null || true
fi

echo "Pruned embed bundle under ${PY_ROOT}"
