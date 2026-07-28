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
