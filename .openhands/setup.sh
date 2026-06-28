#!/usr/bin/env bash
# OpenHands repo setup hook for the W3A bundle.
# Runs once when the workspace is prepared. Keep it idempotent and side-effect free
# beyond fetching the Go module cache; it must never write secrets into the repo.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/conformance"

if command -v go >/dev/null 2>&1; then
  echo "[w3a setup] go found: $(go version)"
  go mod download
  echo "[w3a setup] running bundle-check"
  go run ./tools/w3a bundle-check
else
  echo "[w3a setup] Go toolchain not found; skipping conformance build."
  echo "[w3a setup] Install Go to run: go run ./tools/w3a bundle-check"
fi

echo "[w3a setup] done. Entry points: ../SKILL.md, ../AGENTS.md, ../W3A-SPEC.md"
