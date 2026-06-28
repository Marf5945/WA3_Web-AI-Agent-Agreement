#!/usr/bin/env bash
# OpenHands repo setup hook for the WA3 bundle.
# Runs once when the workspace is prepared. Keep it idempotent and side-effect free
# beyond fetching the Go module cache; it must never write secrets into the repo.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/conformance"

if command -v go >/dev/null 2>&1; then
  echo "[wa3 setup] go found: $(go version)"
  go mod download
  echo "[wa3 setup] running bundle-check"
  go run ./tools/wa3 bundle-check
else
  echo "[wa3 setup] Go toolchain not found; skipping conformance build."
  echo "[wa3 setup] Install Go to run: go run ./tools/wa3 bundle-check"
fi

echo "[wa3 setup] done. Entry points: ../SKILL.md, ../AGENTS.md, ../README.md"
