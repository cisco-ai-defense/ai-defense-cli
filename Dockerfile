# v1: Debian bookworm build stages; exports CLI binary (not a container image).
ARG PY_BUILD_IMAGE=debian:bookworm-slim
ARG GO_BUILD_IMAGE=golang:1.23-bookworm

### Stage 1: relocatable CPython + cisco-aibom[all] ###
FROM ${PY_BUILD_IMAGE} AS py-bundler

ARG PBS_RELEASE=20250723
ARG PYTHON_VERSION=3.12.11
ARG TARGETARCH=amd64
# Set per-arch from Makefile build-release-linux (see SHA256SUMS in PBS release).
ARG PBS_TRIPLE=x86_64-unknown-linux-gnu
ARG PBS_SHA256=7263ed87e9d2d4c52b01f1ad8cdca315cf859c0848b8ffdcb70de205fa8b0336

COPY scripts/docker/install-build-deps.sh /tmp/install-build-deps.sh
RUN chmod +x /tmp/install-build-deps.sh && /tmp/install-build-deps.sh

ENV AID_PY_HOME=/opt/aid-py
RUN mkdir -p ${AID_PY_HOME}

RUN asset="cpython-${PYTHON_VERSION}+${PBS_RELEASE}-${PBS_TRIPLE}-install_only_stripped.tar.gz" && \
    base="https://github.com/astral-sh/python-build-standalone/releases/download/${PBS_RELEASE}" && \
    curl -fsSL --retry 5 --retry-all-errors -o /tmp/python.tar.gz "${base}/${asset}" && \
    echo "${PBS_SHA256}  /tmp/python.tar.gz" | sha256sum -c - && \
    tar -xzf /tmp/python.tar.gz -C ${AID_PY_HOME} && \
    rm -f /tmp/python.tar.gz

ENV PYBIN=${AID_PY_HOME}/python/bin/python3

COPY scripts/requirements-bundle.txt /tmp/requirements-bundle.txt
COPY scripts/prune-python-bundle.sh /tmp/prune-python-bundle.sh
RUN --mount=type=cache,target=/root/.cache/pip \
    ${PYBIN} -m pip install --no-warn-script-location --no-cache-dir \
        --timeout 60 --retries 10 \
        -r /tmp/requirements-bundle.txt

RUN chmod +x /tmp/prune-python-bundle.sh && \
    /tmp/prune-python-bundle.sh ${AID_PY_HOME}/python

RUN tar -czf /aid-py-bundle.tar.gz -C ${AID_PY_HOME} . && \
    sha256sum /aid-py-bundle.tar.gz | cut -c1-16 > /aid-py-bundle.id

### Stage 2: Go binary with embed_python ###
FROM ${GO_BUILD_IMAGE} AS go-builder

ARG CLI_VERSION=dev
ARG CLI_COMMIT=unknown
ARG TARGETARCH=amd64

ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/root/.cache/go-build

WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN mkdir -p internal/embedded/python/bin
COPY --from=py-bundler /aid-py-bundle.tar.gz internal/embedded/python/bin/aid-py-bundle.tar.gz
COPY --from=py-bundler /aid-py-bundle.id internal/embedded/python/bin/aid-py-bundle.id

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -tags embed_python \
    -ldflags="-s -w \
      -X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.version=${CLI_VERSION} \
      -X github.com/cisco-ai-defense/ai-defense-cli/internal/cli.commit=${CLI_COMMIT}" \
    -o /out/aid-cisco-cli .

### Stage 3: export binary only (docker build --target release --output dist/) ###
FROM scratch AS release
COPY --from=go-builder /out/aid-cisco-cli /aid-cisco-cli
