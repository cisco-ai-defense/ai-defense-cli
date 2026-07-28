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

"""Extract a version section from CHANGELOG.md for GitHub Release notes."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


def extract(changelog: str, version: str) -> str:
    pattern = (
        rf"^## \[{re.escape(version)}\]"
        rf"(?:\s+-\s+[^\n]*)?"
        rf"\s*\n(.*?)(?=\n## \[|\Z)"
    )
    match = re.search(pattern, changelog, re.MULTILINE | re.DOTALL)
    if not match:
        raise SystemExit(f"CHANGELOG.md has no section for [{version}]")
    body = match.group(1).strip()
    if not body:
        raise SystemExit(f"CHANGELOG.md section [{version}] is empty")
    return body


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True, help="Semver without v prefix (e.g. 0.1.1)")
    parser.add_argument(
        "--changelog",
        type=Path,
        default=Path("CHANGELOG.md"),
        help="Path to CHANGELOG.md",
    )
    parser.add_argument(
        "-o",
        "--output",
        type=Path,
        help="Write notes to file (default: stdout)",
    )
    args = parser.parse_args()

    text = args.changelog.read_text(encoding="utf-8")
    body = extract(text, args.version)

    if args.output:
        args.output.write_text(body + "\n", encoding="utf-8")
    else:
        sys.stdout.write(body + "\n")


if __name__ == "__main__":
    main()
