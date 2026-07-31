# ADR 0003: GoReleaser Pro as the release build and publication pipeline

## Status

Accepted (2026-07-31); activation requires an approved GoReleaser Pro entitlement
and the masked repository Actions secret `GORELEASER_KEY`.

## Context

The CLI publishes three platform-specific binaries. Each contains a relocatable
CPython distribution and `cisco-aibom`, so the Python bundle must be assembled on
the target operating system before the Go binary is compiled.

The initial integration used GoReleaser OSS only as the publisher and retained
custom Make/Docker/native compilation plus metadata generation. That reduced
publication scripting but did not centralize the release lifecycle. GoReleaser
split/merge is designed for target-native multi-runner builds and is available in
GoReleaser Pro.

## Decision

Use GoReleaser Pro for the complete tagged release lifecycle:

1. Run `goreleaser release --split` on Linux amd64, macOS amd64, and macOS arm64.
2. Stage and verify the native bundle before the secret-bearing GoReleaser step,
   let GoReleaser compile with `embed_python`, then smoke-test the binary and
   embedded analyzer in a separate step without the license key.
3. Transfer the complete split directories as immutable workflow artifacts.
4. Run `goreleaser continue --merge` once to generate SHA-256 checksums, render
   the compatibility manifest, and publish the curated GitHub Release.
5. Pin the GoReleaser Pro binary version, require the repository secret
   `GORELEASER_KEY`, and grant `contents: write` only to the merge/publish job.

Release Engineering owns license renewal and secret rotation. A trusted preflight
job runs `goreleaser license-verify` before starting the platform matrix. If the
license is missing or expired, maintainers renew the entitlement, replace the
masked secret, and re-run the workflow; no artifacts are published.

## Consequences

- Public artifact names, checksums, manifest schema, release notes, and the
  self-update contract remain unchanged.
- GoReleaser owns compilation, raw-binary packaging, checksums, manifest
  rendering, and GitHub Release publication.
- The repository requires an active GoReleaser Pro license stored as the masked
  Actions secret `GORELEASER_KEY`.
- Repository and dependency code used to prepare or exercise the bundle runs
  outside the step that receives the license key.
- Release jobs continue to use ephemeral `GITHUB_TOKEN` credentials. Split jobs
  are read-only; only the merge/publish job can write repository contents.
- Rollback requires restoring the previous custom platform build and fan-in
  workflow, not only swapping the publisher.

The last known-good hybrid implementation is commit `a296a2e`. Emergency rollback
restores `.goreleaser.yaml` and `.github/workflows/release.yml` from that commit,
then validates the workflow before creating another release tag.

## Alternatives considered

| Option | Pros | Cons |
|--------|------|------|
| GoReleaser OSS single-runner build | No license | Cannot safely produce all target-native Python bundles |
| GoReleaser OSS hybrid publisher | Low migration risk | Retains custom compilation, fan-in, checksums, and manifest generation |
| Keep the custom workflow | No license or migration | No centralized release lifecycle |
| **GoReleaser Pro split/merge (chosen)** | Complete native build-to-publish lifecycle; tested artifacts are published unchanged | Paid license and masked repository secret |

## References

- Spec 003 REQ-10
- `docs/install-upgrade.md`
- <https://goreleaser.com/customization/general/partial/>
- <https://goreleaser.com/customization/general/templatefiles/>
