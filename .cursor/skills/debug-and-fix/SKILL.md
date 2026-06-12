---
name: debug-and-fix
description: Debug workflow for ai-defense-cli bugs and regressions. Reproduce → find root cause → fix → verify. Ask follow-up questions if the report is vague.
---

# Debug & Fix

## 1. Reproduce

Gather:

- Command and flags
- Exit code (`echo $?`)
- CLI version, OS/arch
- Config source (flag, env, TOML)

```bash
go build -o aid-cisco-cli .
./aid-cisco-cli <command> ... ; echo exit=$?
```

## 2. Classify

| Symptom | Look in |
|---------|---------|
| Wrong exit code | `internal/clierror`, `internal/verdict`, command handler |
| Auth/config (exit 3) | `internal/config`, `internal/cli/root.go` |
| API failure (4–5) | `api/` |
| Bad JSON output | `internal/verdict`, `internal/output` |
| Threshold wrong | `internal/cmdutil/scanresult.go` |
| AIBOM / embed failure | `internal/embedded/python/`, `scripts/requirements-bundle.txt`, Dockerfile embed stage |
| Bad release binary | `.github/workflows/`, Dockerfile |

## 3. Fix

- Minimal change
- Add regression test in `*_test.go`
- Update spec/docs if behavior was wrong in docs or contract changed

```bash
go build ./... && go vet ./... && go test ./...
```

## 4. Verify

Re-run original reproduction. Test adjacent commands if shared code changed.

## 5. Close

`CHANGELOG.md` + `CONTEXT.md` for user-visible fixes.
