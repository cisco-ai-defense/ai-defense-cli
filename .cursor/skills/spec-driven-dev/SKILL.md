---
name: spec-driven-dev
description: Spec-first workflow for ai-defense-cli with persona gates after SPEC and IMPLEMENT. Use when implementing features, fixes, or release changes. Ends with pr-review before merge. Ask follow-up questions if scope is unclear.
---

# Spec-Driven Development

For **ai-defense-cli** only. No epics/stories machinery unless the user asks for it.

Personas (from `.github/prompts/`) catch contract, security, and doc issues **during** development — not only at PR time.

## Workflow

```
1. UNDERSTAND  → CONTEXT.md, relevant docs/specs
2. SPEC        → Update docs before code (if behavior changes)
3. GATE: SPEC  → Persona review of spec/docs only → fix BLOCKER/MAJOR
4. IMPLEMENT   → Code + tests
5. VERIFY      → make test vet / go build
6. GATE: CODE  → Persona review of code + tests → fix BLOCKER/MAJOR
7. CONTEXT     → CONTEXT.md, CHANGELOG.md
8. (merge)     → Full pr-review skill on the branch
```

**Spec before code** when user-visible behavior, exit codes, flags, or release layout changes.

**Do not skip gates** for non-trivial work (new commands, contract changes, embed/release changes).
Skip gates only for trivial fixes (typos, comments) — note the skip in the commit/PR.

---

## Personas

Read `.github/prompts/system.md` before any persona prompt.

| Persona | Prompt | Focus |
|---------|--------|--------|
| Engineer | `persona-engineer.md` | Go structure, tests, errors |
| Security | `persona-security.md` | Credentials, SSRF, supply chain |
| CLI Contract | `persona-cli-contract.md` | Exit codes, flags, output, semver |
| Release | `persona-release.md` | Makefile, CI, embed bundle, artifacts |
| Docs | `persona-docs.md` | Specs, install docs, CHANGELOG |

Gate reviews are **scoped** — only files touched in that phase. No PRS score; findings only.
Fix BLOCKER and MAJOR before advancing. Defer MINOR/NIT to PR or follow-up.

---

## Phase 1: UNDERSTAND

Read:

- `CONTEXT.md` — status and migration plan
- `docs/architecture/ai-defense-cli.md`
- Relevant `docs/specs/{area}/` if it exists
- `docs/install-upgrade.md` for distribution work

---

## Phase 2: SPEC

Update as needed:

| Change | Update |
|--------|--------|
| New command or flag | `docs/specs/`, `README.md`, `CHANGELOG.md` |
| Exit code or JSON output | `docs/specs/`, `docs/architecture/ai-defense-cli.md` |
| Release/install | `docs/install-upgrade.md`, workflow comments |
| Architecture | `docs/architecture/ai-defense-cli.md`, optional ADR in `docs/adr/` |

---

## Phase 3: GATE — SPEC

Review **only** files changed in Phase 2 (`docs/**`, `README.md`, `CHANGELOG.md`).

| Persona | Include when |
|---------|----------------|
| Docs | Always |
| CLI Contract | Commands, flags, exit codes, or output documented |
| Release | Install/upgrade or distribution docs touched |

Run personas (parallel if using subagents). Each outputs `Score: X/5` and findings tagged BLOCKER | MAJOR | MINOR | NIT.

**Stop** if any BLOCKER/MAJOR — fix specs, then re-run affected personas.

Optional log: `reviews/dev-{branch}-spec-gate.md` (local; not required to commit).

---

## Phase 4: IMPLEMENT

Typical layout:

```
internal/cli/               # Cobra commands
internal/                 # config, clierror, verdict, cmdutil
internal/embedded/python/ # relocatable CPython + cisco-aibom bridge
scripts/                  # Python bundle requirements / build helpers
api/                        # HTTP client
```

Embedded Python: stub + full-embed paths; see `.cursor/rules/embedded-python.mdc`.

```bash
make build-go test vet    # daily dev
# after embed changes:
make bundle build-embed
```

---

## Phase 5: VERIFY

- Re-run tests and build for the mode you changed (go-only vs embed)
- If CLI contract changed: exercise command and check exit code
- `ReadLints` on edited files

---

## Phase 6: GATE — CODE

Review **only** code/test/build files changed in Phase 4–5.

| Persona | Include when |
|---------|----------------|
| Engineer | `*.go` changed |
| Security | Go, Makefile, scripts, workflows |
| CLI Contract | `internal/cli/`, flags, exit mapping, JSON output |
| Release | `Makefile`, `scripts/`, Dockerfile, `.github/workflows/` |
| Docs | If code change requires doc fix not done in SPEC |

Fix BLOCKER/MAJOR before Phase 7. Re-run affected personas after fixes.

Optional log: `reviews/dev-{branch}-code-gate.md`.

---

## Phase 7: CONTEXT

Update `CONTEXT.md` and `CHANGELOG.md` (user-visible changes).

Create `docs/adr/{NNNN}-{slug}.md` only for non-obvious architectural decisions.

---

## Phase 8: Merge readiness

Before opening or merging a PR, run the full **[pr-review](../pr-review/SKILL.md)** skill on the branch.

- Spec/code gates = **early, scoped** feedback during development
- `pr-review` = **final** synthesized PRS report on the whole branch diff

If `pr-review` finds new BLOCKER/MAJOR issues, fix via [address-review-comments](../address-review-comments/SKILL.md) and re-run `pr-review`.

---

## Checklist

```
- [ ] UNDERSTAND: CONTEXT + specs read
- [ ] SPEC: docs updated before code
- [ ] GATE SPEC: docs/cli-contract/release personas; no open BLOCKER/MAJOR
- [ ] IMPLEMENT + VERIFY: build/test pass
- [ ] GATE CODE: relevant personas; no open BLOCKER/MAJOR
- [ ] CONTEXT: CHANGELOG + CONTEXT updated
- [ ] pr-review before merge
```
