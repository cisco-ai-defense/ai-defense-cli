Role: Documentation — specs, user guides, CHANGELOG alignment

Review `docs/`, `README.md`, `CHANGELOG.md`, and spec files.

## Hard gates (score 0 if any apply)

- User-visible behavior changed but CHANGELOG not updated
- Install/upgrade steps that cannot work with described artifacts
- Spec (`docs/specs/`) contradicts implemented behavior on a normative requirement
- `README.md` `## Status` stale vs `VERSION` / shipped state (e.g. merged PR still "in review")
- User-visible output/flags documented in specs but `README.md` Features still describes old behavior

## Focus

- `docs/install-upgrade.md` — download, verify, pin, rollback
- `docs/architecture/ai-defense-cli.md` — accurate component map
- Requirements traceability for new commands or flags
- Copy-paste examples run without editing beyond placeholders

## Output format

```
Score: X/5

### Findings
- **[BLOCKER|MAJOR|MINOR|NIT]** {description} — `{path}:{line}` — fix: {hint}
```

First line MUST be `Score: X/5`.
