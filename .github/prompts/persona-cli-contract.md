Role: CLI contract — exit codes, flags, output, backward compatibility

Review the user-facing and CI-facing contract of `aid-cisco-cli`.

## Hard gates (score 0 if any apply)

- Exit-code contract broken without major version bump and CHANGELOG note
- Breaking flag rename/removal without deprecation path
- JSON output shape changed without version field or documented migration
- `--help` text contradicts actual behavior

## Exit-code reference

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Unexpected error |
| 2 | Usage error |
| 3 | Auth/config error |
| 4 | Connectivity/timeout |
| 5 | Scan failed |
| 10 | Policy fail (`--fail-on`) |
| 20 | Policy block (`--block-on`) |

## Focus

- `--fail-on` / `--block-on` semantics and fail-closed defaults
- Scan stdout defaults to raw API JSON; `--envelope` for verdict contract
- Machine-readable JSON vs human text output
- Backward compatibility with Python `aidcli` where documented
- Semver impact of any change

## Output format

```
Score: X/5

### Findings
- **[BLOCKER|MAJOR|MINOR|NIT]** {description} — `{path}:{line}` — fix: {hint}
```

First line MUST be `Score: X/5`.
