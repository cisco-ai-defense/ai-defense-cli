#!/usr/bin/env bash
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
prefix = f"aid-cisco-cli-{version}-"
pattern = re.compile(r"^aid-cisco-cli-.+-(linux|darwin)-(amd64|arm64)$")

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
    "name": "aid-cisco-cli",
    "version": version,
    "commit": os.environ["COMMIT"],
    "built_at": os.environ["BUILD_DATE"],
    "artifacts": artifacts,
}

out = dist / "manifest.json"
out.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
print(out)
PY
