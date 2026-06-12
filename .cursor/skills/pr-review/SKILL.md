---
name: pr-review
description: Multi-persona branch review for ai-defense-cli before opening or merging a PR. Runs five CLI-focused personas, writes a report, fixes BLOCKER/MAJOR issues, re-reviews. Use when the user asks for a code review or branch readiness check.
---

# PR Review

Review a local branch against a target (default `main`) before merge.

**GitHub PRs:** `.github/workflows/pr-persona-review.yml` posts the same persona
synthesis as an **advisory PR comment only** (read-only — no auto-commits). Requires
repo secret `PERSONA_REVIEW_API_KEY`.

## Personas

Read `.github/prompts/system.md` plus each persona file:

| Persona | File | When to include |
|---------|------|-----------------|
| Engineer | `persona-engineer.md` | Go code changed |
| Security | `persona-security.md` | Go code, workflows, Dockerfile |
| CLI Contract | `persona-cli-contract.md` | Commands, flags, exit codes, output |
| Release | `persona-release.md` | Workflows, Makefile, Dockerfile, install docs |
| Docs | `persona-docs.md` | `docs/`, README, CHANGELOG, specs |

Omit personas with nothing relevant in the diff.

## Workflow

```
1. SCOPE     → git diff target...branch --stat
2. REVIEW    → Run personas (parallel batches of 3–5)
3. SYNTHESIZE → Apply .github/prompts/synthesizer.md
4. REPORT    → Write reviews/{branch-short}-review.md
5. FIX       → BLOCKER and MAJOR findings
6. RE-REVIEW → Re-run affected personas; overwrite report
7. PUSH      → When PRS ≥ 70 and no BLOCKERs (user asks to push)
```

## Scope commands

```bash
git diff <target>...<branch> --stat
git log <target>...<branch> --oneline
git diff <target>...<branch> --name-only
```

## Fix rules

| Finding | Action |
|---------|--------|
| Code bug, missing test, wrong exit mapping | Fix in code |
| Secret leak, missing timeout | Fix in code |
| Doc/spec drift | Fix docs |
| Release wiring | Fix workflow/Makefile |
| Architecture / external API change | Defer — list under "Human follow-up" |

After fixes:

```bash
go build ./... && go vet ./... && go test ./...
```

Commit only when the user asks:

```bash
git commit -m "fix(review): address PR review findings"
```

## Report location

`reviews/{branch-short-name}-review.md` (gitignored — local only, do not commit)

Optional: render a canvas if the user wants a visual report (see Cursor canvas skill).

## Checklist

```
- [ ] Diff scoped
- [ ] Personas run on relevant files
- [ ] Report written with PRS
- [ ] BLOCKER/MAJOR fixed or deferred with rationale
- [ ] Tests pass
```
