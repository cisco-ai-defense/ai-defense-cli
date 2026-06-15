#!/usr/bin/env python3
"""Fill missing Keep a Changelog [Unreleased] bullets from merged PRs since last tag."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path

MARKER = "## [Unreleased]"
SECTIONS = ("Added", "Changed", "Fixed", "Removed", "Deprecated", "Security")
SKIP_TITLE_RE = re.compile(
    r"(?i)(chore\(release\)|\[skip changelog\]|^dependabot)",
)
MERGE_PR_RE = re.compile(r"#(\d+)\b")


@dataclass(frozen=True)
class Entry:
    section: str
    bullet: str
    key: str


def run(cmd: list[str], *, cwd: Path | None = None) -> str:
    result = subprocess.run(
        cmd,
        cwd=cwd,
        check=True,
        capture_output=True,
        text=True,
    )
    return result.stdout.strip()


def normalize(text: str) -> str:
    text = text.lower()
    text = re.sub(r"\[#?\d+\]", "", text)
    text = re.sub(r"\(\#\d+\)", "", text)
    text = re.sub(r"[^a-z0-9]+", " ", text)
    return " ".join(text.split())


def parse_unreleased(changelog: str) -> tuple[str, dict[str, list[str]], str]:
    if MARKER not in changelog:
        raise SystemExit("CHANGELOG.md has no [Unreleased] section")

    before, rest = changelog.split(MARKER, 1)
    match = re.match(r"\s*\n?(.*?)(?=\n## \[|\Z)", rest, re.DOTALL)
    if not match:
        raise SystemExit("could not parse [Unreleased] body")

    body = match.group(1)
    after = rest[match.end() :]
    sections: dict[str, list[str]] = {name: [] for name in SECTIONS}

    current: str | None = None
    current_bullet: list[str] = []

    def flush_bullet() -> None:
        nonlocal current_bullet
        if current and current_bullet:
            sections[current].append("\n".join(current_bullet))
        current_bullet = []

    for line in body.splitlines():
        heading = re.match(r"^### (Added|Changed|Fixed|Removed|Deprecated|Security)\s*$", line)
        if heading:
            flush_bullet()
            current = heading.group(1)
            continue
        if current is None:
            continue
        if line.startswith("- "):
            flush_bullet()
            current_bullet = [line]
            continue
        if current_bullet and (line.startswith(" ") or line.startswith("\t")):
            current_bullet.append(line)
            continue
        if line.strip() == "":
            flush_bullet()
            continue

    flush_bullet()
    return before, sections, after


def render_unreleased(sections: dict[str, list[str]]) -> str:
    parts: list[str] = []
    for name in SECTIONS:
        bullets = sections[name]
        if not bullets:
            continue
        parts.append(f"### {name}\n")
        parts.extend(f"{b}\n" for b in bullets)
        parts.append("\n")
    return "".join(parts).rstrip()


def write_changelog(
    path: Path,
    before: str,
    sections: dict[str, list[str]],
    after: str,
) -> None:
    body = render_unreleased(sections)
    if body:
        body += "\n\n"
    path.write_text(before + MARKER + "\n\n" + body + after.lstrip("\n"), encoding="utf-8")


def latest_tag(repo: Path) -> str | None:
    try:
        tags = run(["git", "tag", "--list", "v*", "--sort=-v:refname"], cwd=repo)
    except subprocess.CalledProcessError:
        return None
    for tag in tags.splitlines():
        tag = tag.strip()
        if tag:
            return tag
    return None


def classify(title: str, labels: list[str]) -> str | None:
    if SKIP_TITLE_RE.search(title):
        return None

    lower = title.lower()
    label_set = {label.lower() for label in labels}

    if "breaking" in label_set or lower.startswith("breaking") or "!" in title.split(":", 1)[0]:
        return "Changed"

    if lower.startswith("fix") or "bug" in label_set:
        return "Fixed"

    if lower.startswith("feat") or "enhancement" in label_set or "feature" in label_set:
        return "Added"

    if lower.startswith(("docs", "doc:", "chore", "ci", "test", "refactor")):
        return "Changed"

    return "Changed"


def format_bullet(title: str, number: int | None, breaking: bool) -> str:
    prefix = "**Breaking:** " if breaking else ""
    clean = title.strip()
    if number is not None:
        return f"- {prefix}{clean} (#{number})"
    return f"- {prefix}{clean}"


def entry_key(bullet: str) -> str:
    return normalize(bullet)


def already_present(sections: dict[str, list[str]], key: str) -> bool:
    for bullets in sections.values():
        for bullet in bullets:
            if key == entry_key(bullet) or key in normalize(bullet) or normalize(bullet) in key:
                return True
    return False


def merged_prs_gh(base: str, repo: Path) -> list[tuple[str, list[str], int]]:
    try:
        raw = run(
            [
                "gh",
                "pr",
                "list",
                "--state",
                "merged",
                "--base",
                base,
                "--json",
                "title,number,labels",
                "--limit",
                "200",
            ],
            cwd=repo,
        )
    except (subprocess.CalledProcessError, FileNotFoundError):
        return []

    items: list[tuple[str, list[str], int]] = []
    for row in json.loads(raw or "[]"):
        labels = [label.get("name", "") for label in row.get("labels") or []]
        items.append((row["title"], labels, int(row["number"])))
    return items


def merged_prs_git(since_tag: str | None, repo: Path) -> list[tuple[str, list[str], int | None]]:
    rev_range = f"{since_tag}..HEAD" if since_tag else "HEAD"
    try:
        log = run(
            ["git", "log", rev_range, "--merges", "--first-parent", "--pretty=%s"],
            cwd=repo,
        )
    except subprocess.CalledProcessError:
        return []

    items: list[tuple[str, list[str], int | None]] = []
    for subject in log.splitlines():
        subject = subject.strip()
        if not subject:
            continue
        number_match = MERGE_PR_RE.search(subject)
        number = int(number_match.group(1)) if number_match else None
        title = re.sub(r"^Merge pull request #\d+ from \S+\s*", "", subject)
        title = re.sub(r"^Merge branch '[^']+' into \S+\s*", "", title).strip()
        if not title:
            title = subject
        items.append((title, [], number))
    return items


def collect_entries(
    base: str,
    since_tag: str | None,
    repo: Path,
    skip_internal: bool,
) -> list[Entry]:
    if since_tag:
        candidates = merged_prs_git(since_tag, repo)
    else:
        candidates = merged_prs_gh(base, repo)
        if not candidates:
            candidates = merged_prs_git(since_tag, repo)

    entries: list[Entry] = []
    for title, labels, number in candidates:
        section = classify(title, labels)
        if section is None:
            continue
        if skip_internal and section == "Changed" and re.match(
            r"(?i)^(docs|chore|ci|test|refactor)\b", title.strip()
        ):
            continue
        breaking = "breaking" in {label.lower() for label in labels} or title.strip().startswith(
            "Breaking"
        )
        bullet = format_bullet(title, number, breaking)
        entries.append(Entry(section=section, bullet=bullet, key=entry_key(bullet)))
    return entries


def apply_entries(sections: dict[str, list[str]], entries: list[Entry]) -> list[Entry]:
    added: list[Entry] = []
    for entry in entries:
        if already_present(sections, entry.key):
            continue
        sections[entry.section].append(entry.bullet)
        added.append(entry)
    return added


def resolve_base_branch(ref: str, repo: Path) -> str:
    if ref == "HEAD":
        try:
            return run(["git", "rev-parse", "--abbrev-ref", "HEAD"], cwd=repo)
        except subprocess.CalledProcessError:
            return "main"
    return ref.split("/", 1)[-1] if ref.startswith("refs/heads/") else ref


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--ref", default="HEAD", help="Branch to scope merged PRs (default: current)")
    parser.add_argument("--since-tag", default="", help="Override baseline tag (default: latest v*)")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument(
        "--skip-internal",
        action="store_true",
        help="Skip docs/chore/ci/test/refactor-only Changed entries",
    )
    args = parser.parse_args()

    repo = args.repo.resolve()
    changelog_path = repo / "CHANGELOG.md"
    text = changelog_path.read_text(encoding="utf-8")
    before, sections, after = parse_unreleased(text)

    since_tag = args.since_tag or latest_tag(repo)
    base = resolve_base_branch(args.ref, repo)

    entries = collect_entries(base, since_tag, repo, args.skip_internal)
    added = apply_entries(sections, entries)

    if args.dry_run:
        print(f"Baseline tag: {since_tag or '(none)'}")
        print(f"Base branch: {base}")
        print(f"Would add {len(added)} bullet(s):")
        for entry in added:
            print(f"  [{entry.section}] {entry.bullet}")
        return 0

    write_changelog(changelog_path, before, sections, after)
    print(f"Autofill: added {len(added)} bullet(s) under [Unreleased]")
    for entry in added:
        print(f"  [{entry.section}] {entry.bullet}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
