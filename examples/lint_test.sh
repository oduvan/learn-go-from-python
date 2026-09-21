#!/usr/bin/env bash
# Verifies docs/13-third-party-libraries/02-golangci-lint-configuration.md
#
# The depguard rule in lintdemo/.golangci.yml must reject the deliberate
# violation in internal/web. A clean run here means the rule is NOT
# working, so the expected outcome is a specific failure.
set -uo pipefail
cd "$(dirname "$0")/lintdemo" || exit 1

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "SKIP  golangci-lint: not installed"
  exit 0
fi

out=$(golangci-lint run ./... 2>&1)
if grep -q "only internal/storage may hold a database handle" <<<"$out"; then
  if grep -q "internal/web/handler.go" <<<"$out"; then
    echo "ok    golangci-lint: depguard rejected the layering violation"
    exit 0
  fi
fi
echo "FAIL  golangci-lint: depguard did not fire as the article describes"
echo "$out" | head -20
exit 1
