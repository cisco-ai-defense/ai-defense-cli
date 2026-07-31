#!/usr/bin/env python3
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

"""Validate GoReleaser checksum and compatibility-manifest contracts."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
from pathlib import Path


CHECKSUM_LINE = re.compile(r"^([0-9a-f]{64})  (\S+)$")
TOP_LEVEL_KEYS = {"name", "version", "commit", "built_at", "artifacts"}
ARTIFACT_KEYS = {"file", "os", "arch", "sha256"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--checksums", required=True, type=Path)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    return parser.parse_args()


def expected_files(version: str) -> set[str]:
    return {
        f"cisco-aid-cli-{version}-linux-amd64",
        f"cisco-aid-cli-{version}-darwin-amd64",
        f"cisco-aid-cli-{version}-darwin-arm64",
    }


def read_checksums(path: Path) -> dict[str, str]:
    checksums: dict[str, str] = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        match = CHECKSUM_LINE.fullmatch(line)
        if match is None:
            raise SystemExit(f"invalid checksum line: {line!r}")
        digest, filename = match.groups()
        if filename in checksums:
            raise SystemExit(f"duplicate checksum entry: {filename}")
        checksums[filename] = digest
    return checksums


def validate_timestamp(value: object) -> None:
    if not isinstance(value, str):
        raise SystemExit("manifest built_at must be a string")
    try:
        dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as error:
        raise SystemExit(f"manifest built_at is not RFC3339: {value!r}") from error


def main() -> None:
    args = parse_args()
    expected = expected_files(args.version)
    checksums = read_checksums(args.checksums)
    if set(checksums) != expected:
        raise SystemExit(
            f"checksum inventory mismatch: expected {sorted(expected)}, "
            f"found {sorted(checksums)}"
        )

    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    if not isinstance(manifest, dict) or set(manifest) != TOP_LEVEL_KEYS:
        raise SystemExit("manifest top-level schema mismatch")
    if manifest["name"] != "cisco-aid-cli":
        raise SystemExit(f"unexpected manifest name: {manifest['name']!r}")
    if manifest["version"] != args.version:
        raise SystemExit(f"unexpected manifest version: {manifest['version']!r}")
    if manifest["commit"] != args.commit:
        raise SystemExit(f"unexpected manifest commit: {manifest['commit']!r}")
    validate_timestamp(manifest["built_at"])

    artifacts = manifest["artifacts"]
    if not isinstance(artifacts, list) or len(artifacts) != 3:
        raise SystemExit("manifest must contain exactly three artifacts")

    manifest_digests: dict[str, str] = {}
    for artifact in artifacts:
        if not isinstance(artifact, dict) or set(artifact) != ARTIFACT_KEYS:
            raise SystemExit("manifest artifact schema mismatch")
        filename = artifact["file"]
        if filename in manifest_digests:
            raise SystemExit(f"duplicate manifest artifact: {filename}")
        if artifact["os"] not in {"linux", "darwin"}:
            raise SystemExit(f"unsupported manifest OS: {artifact['os']!r}")
        if artifact["arch"] not in {"amd64", "arm64"}:
            raise SystemExit(f"unsupported manifest architecture: {artifact['arch']!r}")
        manifest_digests[filename] = artifact["sha256"]

    if set(manifest_digests) != expected:
        raise SystemExit(
            f"manifest inventory mismatch: expected {sorted(expected)}, "
            f"found {sorted(manifest_digests)}"
        )
    if manifest_digests != checksums:
        raise SystemExit("manifest digests do not match checksums.txt")

    print("Verified GoReleaser release metadata contract")


if __name__ == "__main__":
    main()
