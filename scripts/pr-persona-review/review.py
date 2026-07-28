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

"""Read-only multi-persona PR review for GitHub Actions.

Posts advisory findings only — never modifies the repository or creates commits.
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[2]
PROMPTS_DIR = REPO_ROOT / ".github" / "prompts"
MARKER = "<!-- persona-review-bot -->"
MAX_DIFF_CHARS = 120_000

PERSONAS = [
    (
        "Engineer",
        "persona-engineer.md",
        lambda files: any(
            f.endswith(".go") or "embedded/python" in f for f in files
        ),
    ),
    (
        "Security",
        "persona-security.md",
        lambda files: any(
            f.endswith((".go", ".yml", ".yaml"))
            or f.lower().startswith("dockerfile")
            or "dockerfile" in f.lower()
            for f in files
        ),
    ),
    (
        "CLI Contract",
        "persona-cli-contract.md",
        lambda files: any(
            p in f
            for p in (
                "internal/cli/",
                "internal/clierror/",
                "internal/verdict/",
                "internal/cmdutil/",
            )
            for f in files
        ),
    ),
    (
        "Release",
        "persona-release.md",
        lambda files: any(
            p in f
            for p in (
                "Makefile",
                "Dockerfile",
                ".github/workflows/",
                "scripts/",
                "docs/install-upgrade",
            )
            for f in files
        ),
    ),
    (
        "Docs",
        "persona-docs.md",
        lambda files: any(
            f.startswith("docs/") or f in ("README.md", "CHANGELOG.md", "CONTEXT.md")
            for f in files
        ),
    ),
]


def run_git(*args: str) -> str:
    return subprocess.check_output(
        ["git", *args], cwd=REPO_ROOT, text=True, stderr=subprocess.STDOUT
    )


def changed_files(base: str, head: str) -> list[str]:
    out = run_git("diff", "--name-only", f"{base}...{head}")
    return [line.strip() for line in out.splitlines() if line.strip()]


def diff_text(base: str, head: str) -> str:
    text = run_git("diff", f"{base}...{head}")
    if len(text) > MAX_DIFF_CHARS:
        return (
            text[:MAX_DIFF_CHARS]
            + f"\n\n... [diff truncated at {MAX_DIFF_CHARS} chars] ..."
        )
    return text


def read_prompt(name: str) -> str:
    return (PROMPTS_DIR / name).read_text(encoding="utf-8")


def call_llm(system: str, user: str) -> str:
    api_key = os.environ.get("PERSONA_REVIEW_API_KEY", "").strip()
    if not api_key:
        raise RuntimeError("missing PERSONA_REVIEW_API_KEY")

    provider = os.environ.get("PERSONA_REVIEW_PROVIDER", "anthropic").strip().lower()
    model = os.environ.get("PERSONA_REVIEW_MODEL", "").strip()

    if provider == "openai":
        model = model or "gpt-4o"
        body = {
            "model": model,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
            "temperature": 0.2,
        }
        req = urllib.request.Request(
            "https://api.openai.com/v1/chat/completions",
            data=json.dumps(body).encode(),
            headers={
                "Authorization": f"Bearer {api_key}",
                "Content-Type": "application/json",
            },
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=180) as resp:
            data = json.loads(resp.read().decode())
        return data["choices"][0]["message"]["content"]

    # default: anthropic
    model = model or "claude-sonnet-4-20250514"
    body = {
        "model": model,
        "max_tokens": 4096,
        "system": system,
        "messages": [{"role": "user", "content": user}],
        "temperature": 0.2,
    }
    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages",
        data=json.dumps(body).encode(),
        headers={
            "x-api-key": api_key,
            "anthropic-version": "2023-06-01",
            "Content-Type": "application/json",
        },
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=180) as resp:
        data = json.loads(resp.read().decode())
    parts = [b.get("text", "") for b in data.get("content", []) if b.get("type") == "text"]
    return "\n".join(parts)


def review_persona(name: str, persona_file: str, diff: str, files: list[str]) -> str:
    system = read_prompt("system.md") + "\n\n" + read_prompt(persona_file)
    user = (
        f"Review this pull request diff for ai-defense-cli.\n\n"
        f"Changed files ({len(files)}):\n"
        + "\n".join(f"- {f}" for f in files[:200])
        + ("\n- ..." if len(files) > 200 else "")
        + f"\n\n```diff\n{diff}\n```\n\n"
        "Respond using your required output format. Advisory only — do not propose commits."
    )
    return call_llm(system, user)


def synthesize(persona_outputs: dict[str, str], diff_summary: str) -> str:
    system = read_prompt("system.md") + "\n\n" + read_prompt("synthesizer.md")
    blocks = []
    for name, out in persona_outputs.items():
        blocks.append(f"## {name}\n\n{out}")
    user = (
        "Synthesize these persona reviews into one markdown report.\n\n"
        f"PR summary context:\n{diff_summary}\n\n"
        + "\n\n".join(blocks)
        + "\n\nOutput markdown only. Include PRS table and readiness band."
    )
    return call_llm(system, user)


def skipped_report(reason: str) -> str:
    return (
        f"{MARKER}\n"
        "## Persona review (advisory)\n\n"
        f"**Status:** skipped — {reason}\n\n"
        "This check is read-only and never commits changes."
    )


def main() -> int:
    base = os.environ.get("PR_REVIEW_BASE_SHA") or os.environ.get("GITHUB_BASE_SHA")
    head = os.environ.get("PR_REVIEW_HEAD_SHA") or os.environ.get("GITHUB_SHA")
    if not base or not head:
        print(skipped_report("missing base/head SHA"), file=sys.stdout)
        return 0

    if not os.environ.get("PERSONA_REVIEW_API_KEY", "").strip():
        print(
            skipped_report(
                "`PERSONA_REVIEW_API_KEY` repository secret is not configured."
            ),
            file=sys.stdout,
        )
        return 0

    try:
        files = changed_files(base, head)
        if not files:
            print(
                skipped_report("no changed files in diff."),
                file=sys.stdout,
            )
            return 0

        diff = diff_text(base, head)
        persona_outputs: dict[str, str] = {}
        for name, prompt_file, include in PERSONAS:
            if not include(files):
                persona_outputs[name] = "Score: 5/5\n\n### Findings\n- Skipped (no relevant files in diff)."
                continue
            persona_outputs[name] = review_persona(name, prompt_file, diff, files)

        summary = f"{len(files)} files changed between {base[:7]} and {head[:7]}."
        report = synthesize(persona_outputs, summary)
        print(f"{MARKER}\n## Persona review (advisory)\n\n{report}\n\n---\n*Read-only bot — does not commit or modify the branch.*")
        return 0
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")
        print(skipped_report(f"LLM API error {e.code}: {body[:500]}"), file=sys.stdout)
        return 0
    except Exception as e:  # noqa: BLE001
        print(skipped_report(f"review failed: {e}"), file=sys.stdout)
        return 0


if __name__ == "__main__":
    sys.exit(main())
