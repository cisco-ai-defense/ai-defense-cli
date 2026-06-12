#!/usr/bin/env bash
# Bump VERSION and move CHANGELOG [Unreleased] into a dated release section.
# Usage: prepare-release.sh <patch|minor|major> [custom_version]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BUMP="${1:?usage: prepare-release.sh patch|minor|major [custom_version]}"
CUSTOM="${2:-}"

current="$(tr -d '[:space:]' < VERSION)"
if [ -n "$CUSTOM" ]; then
  new="$CUSTOM"
else
  IFS=. read -r major minor patch _ <<< "${current%%-*}"
  patch="${patch%%[^0-9]*}"
  case "$BUMP" in
    patch) patch=$((patch + 1)) ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    major) major=$((major + 1)); minor=0; patch=0 ;;
    *) echo "invalid bump: $BUMP" >&2; exit 1 ;;
  esac
  new="${major}.${minor}.${patch}"
fi

if ! printf '%s' "$new" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'; then
  echo "invalid semver: $new" >&2
  exit 1
fi

if [ "$new" = "$current" ]; then
  echo "new version equals current VERSION ($current)" >&2
  exit 1
fi

export ROOT CURRENT="$current" NEW="$new" RELEASE_DATE="$(date -u +%Y-%m-%d)"
python3 <<'PY'
import os
import re
from pathlib import Path

root = Path(os.environ["ROOT"])
current = os.environ["CURRENT"]
new = os.environ["NEW"]
release_date = os.environ["RELEASE_DATE"]

version_path = root / "VERSION"
changelog_path = root / "CHANGELOG.md"

version_path.write_text(new + "\n", encoding="utf-8")

text = changelog_path.read_text(encoding="utf-8")
marker = "## [Unreleased]"
if marker not in text:
    raise SystemExit("CHANGELOG.md has no [Unreleased] section")

before, rest = text.split(marker, 1)
# rest starts with optional whitespace then ### sections until next ## [version]
m = re.match(r"\s*\n?(.*?)(?=\n## \[|\Z)", rest, re.DOTALL)
if not m:
    raise SystemExit("could not parse [Unreleased] body")
unreleased_body = m.group(1).rstrip()
after = rest[m.end() :]

if not unreleased_body.strip():
    raise SystemExit(
        "[Unreleased] is empty — add changelog entries or run autofill-changelog.py first"
    )

new_section = (
    f"## [{new}] - {release_date}\n\n"
    f"{unreleased_body}\n\n"
    f"## [Unreleased]\n\n"
)

changelog_path.write_text(before + new_section + after.lstrip("\n"), encoding="utf-8")
print(f"VERSION: {current} -> {new}")
print(f"CHANGELOG: finalized [{new}] - {release_date}")
PY
