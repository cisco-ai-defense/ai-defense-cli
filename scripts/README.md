# Build scripts

## Python bundle (`build-python-bundle.sh`)

Produces embed artifacts for the full CLI build:

```text
internal/embedded/python/bin/aid-py-bundle.tar.gz
internal/embedded/python/bin/aid-py-bundle.id
```

Invoked by `make bundle`. Uses [python-build-standalone](https://github.com/astral-sh/python-build-standalone) and `requirements-bundle.txt`.

## Makefile build modes

| Target | Embedded Python | Use case |
|--------|-----------------|----------|
| `make build-go` | No (stub) | Fast dev/CI tests; set `AID_CLI_PYTHON` for local aibom |
| `make build-embed` | Yes | Release binaries; run `make bundle` first |
| `make build BUILD_MODE=full` | Yes | Alias for embed path |

```bash
make build-go test vet          # Go-only
make bundle && make build-embed # Full CLI with cisco-aibom
```

## Release builds

| Target | Output |
|--------|--------|
| `make build-release-linux` | `dist/aid-cisco-cli-linux-$(GOARCH)` via Docker |
| `make build-release-darwin` | `dist/aid-cisco-cli-darwin-$(GOARCH)` native on macOS |
| `make checksums` | `dist/checksums.txt` |
