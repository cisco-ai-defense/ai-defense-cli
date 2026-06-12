#!/usr/bin/env bash
# Validate VERSION and tag preconditions before pushing v{VERSION}.
# Usage: validate-release-tag.sh <version> [commit_sha]
set -euo pipefail

VERSION_INPUT="${1:?usage: validate-release-tag.sh VERSION [commit_sha]}"
TARGET_SHA="${2:-HEAD}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! printf '%s' "$VERSION_INPUT" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'; then
  echo "::error::Invalid semver: $VERSION_INPUT (expected e.g. 0.1.0)" >&2
  exit 1
fi

file_version="$(tr -d '[:space:]' < VERSION)"
if [ "$file_version" != "$VERSION_INPUT" ]; then
  echo "::error::VERSION file is '$file_version' but expected '$VERSION_INPUT'" >&2
  exit 1
fi

TAG="v${VERSION_INPUT}"
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "::error::Tag $TAG already exists locally" >&2
  exit 1
fi
if git ls-remote --exit-code origin "refs/tags/$TAG" >/dev/null 2>&1; then
  echo "::error::Tag $TAG already exists on origin" >&2
  exit 1
fi

sha="$(git rev-parse "$TARGET_SHA")"
git fetch origin main --quiet || true
for branch in $(git branch -r --list 'origin/release/*'); do
  git fetch origin "${branch#origin/}" --quiet || true
done

if git merge-base --is-ancestor "$sha" origin/main 2>/dev/null; then
  echo "Commit $sha is on origin/main history"
  exit 0
fi

for branch in $(git branch -r --list 'origin/release/*'); do
  if git merge-base --is-ancestor "$sha" "$branch" 2>/dev/null; then
    echo "Commit $sha is on ${branch} history"
    exit 0
  fi
done

echo "::error::Commit $sha must be on origin/main or origin/release/* history" >&2
exit 1
