#!/usr/bin/env bash
# Debian bookworm build dependencies for Docker py-bundler stage (v1).
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  tar \
  gzip \
  build-essential \
  patchelf
rm -rf /var/lib/apt/lists/*
