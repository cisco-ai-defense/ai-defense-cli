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

set -euo pipefail

target="${1:?usage: verify-python-bundle-target.sh <goos_goarch>}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bundle="${root}/internal/embedded/python/bin/aid-py-bundle.tar.gz"
bundle_id="${root}/internal/embedded/python/bin/aid-py-bundle.id"

case "${target}" in
  linux_amd64|linux_amd64_*) expected_target="linux_amd64" ;;
  darwin_amd64|darwin_amd64_*) expected_target="darwin_amd64" ;;
  darwin_arm64|darwin_arm64_*) expected_target="darwin_arm64" ;;
  *)
    echo "unsupported release target: ${target}" >&2
    exit 1
    ;;
esac

test -f "${bundle}" || {
  echo "missing Python bundle: ${bundle}" >&2
  exit 1
}
test -f "${bundle_id}" || {
  echo "missing Python bundle id: ${bundle_id}" >&2
  exit 1
}
command -v file >/dev/null || {
  echo "file command is required to verify the embedded interpreter" >&2
  exit 1
}

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT
tar -xzf "${bundle}" -C "${tmp_dir}" python/bin/python3
python_bin="${tmp_dir}/python/bin/python3"
description="$(file -b "${python_bin}")"

case "${expected_target}" in
  linux_amd64)
    [[ "${description}" == *"ELF 64-bit"* && "${description}" == *"x86-64"* ]]
    ;;
  darwin_amd64)
    [[ "${description}" == *"Mach-O 64-bit executable x86_64"* ]]
    ;;
  darwin_arm64)
    [[ "${description}" == *"Mach-O 64-bit executable arm64"* ]]
    ;;
esac || {
  echo "bundle target mismatch: expected ${expected_target}, found ${description}" >&2
  exit 1
}

if command -v sha256sum >/dev/null 2>&1; then
  actual_id="$(sha256sum "${bundle}" | cut -c1-16)"
else
  actual_id="$(shasum -a 256 "${bundle}" | cut -c1-16)"
fi
expected_id="$(tr -d '[:space:]' < "${bundle_id}")"
if [[ "${actual_id}" != "${expected_id}" ]]; then
  echo "bundle id mismatch: expected ${expected_id}, found ${actual_id}" >&2
  exit 1
fi

printf 'Verified embedded Python bundle: %s (%s)\n' "${expected_target}" "${description}"
