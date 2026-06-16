# ADR 0002: Binary name `cisco-aid-cli`

## Status

Accepted (2026-06-16)

## Context

The Go CLI shipped as `aid-cisco-cli` (product abbreviation first, aligned with
the legacy Python `aidcli`). Product naming feedback preferred the Cisco-prefixed
form `cisco-aid-cli`, consistent with other `cisco-*` tools.

The GitHub repository (`ai-defense-cli`) and Go module path are unchanged; only the
**installed command** and **release artifact basenames** change.

## Decision

1. **Binary / Cobra root command:** `cisco-aid-cli`
2. **Release artifacts:** `cisco-aid-cli-<version>-<os>-<arch>`
3. **Install target:** `<prefix>/cisco-aid-cli` (via `cisco-aid-cli install`)
4. **Spec REQ-01** updated accordingly.

No symlink or alias for `aid-cisco-cli` in v1 of this rename — users update scripts
and PATH once. Published releases before the rename use `aid-cisco-cli-*` artifacts;
`cisco-aid-cli install --version` resolves either prefix. Old `aid-cisco-cli install`
cannot self-update across the rename boundary without a manual download.

## Consequences

- Breaking change for CI, docs, and operator scripts using `aid-cisco-cli`.
- `internal/selfupdate` artifact matching prefers `cisco-aid-cli-*`, falls back to
  `aid-cisco-cli-*` for pinned installs of pre-rename releases.
- Migration note in `docs/getting-started-client.md` §7.1.

## Alternatives considered

| Option | Pros | Cons |
|--------|------|------|
| Keep `aid-cisco-cli` | No user churn | Conflicts with Cisco naming guidance |
| Symlink both names for one release | Softer migration | Release layout complexity, two names in support |
| **`cisco-aid-cli` only (chosen)** | Clear product branding | One-time user migration |

## References

- Spec 001 REQ-01
- `docs/install-upgrade.md`
