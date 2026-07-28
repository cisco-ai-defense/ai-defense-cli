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

# Generate dist/manifest.json for a release build.
# Requires: VERSION, COMMIT; optional BUILD_DATE, DIST_DIR (default dist).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="${DIST_DIR:-dist}"
VERSION="${VERSION:?VERSION is required}"
COMMIT="${COMMIT:-unknown}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

export ROOT DIST_DIR VERSION COMMIT BUILD_DATE
python3 <<'PY'
import hashlib
import json
import os
import re
from pathlib import Path

dist = Path(os.environ["DIST_DIR"])
version = os.environ["VERSION"]
prefix = f"cisco-aid-cli-{version}-"
pattern = re.compile(r"^cisco-aid-cli-.+-(linux|darwin)-(amd64|arm64)$")

artifacts = []
for path in sorted(dist.iterdir()):
    if not path.is_file() or not path.name.startswith(prefix):
        continue
    m = pattern.match(path.name)
    if not m:
        continue
    os_name, arch = m.group(1), m.group(2)
    digest = hashlib.sha256(path.read_bytes()).hexdigest()
    artifacts.append(
        {
            "file": path.name,
            "os": os_name,
            "arch": arch,
            "sha256": digest,
        }
    )

if not artifacts:
    raise SystemExit(f"no release binaries matching {prefix}* in {dist}")

manifest = {
    "name": "cisco-aid-cli",
    "version": version,
    "commit": os.environ["COMMIT"],
    "built_at": os.environ["BUILD_DATE"],
    "artifacts": artifacts,
}

out = dist / "manifest.json"
out.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
print(out)
PY
