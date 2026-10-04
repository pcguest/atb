#!/usr/bin/env bash
# Dependency-vulnerability gate for ATB's Python surfaces.
#
# Audits, with a pinned and reproducible tool and fully-pinned inputs:
#   1. the release-tooling lock (sdk/python/requirements-release.txt)
#   2. the SDK runtime dependency graph (sdk/python/requirements-runtime.txt)
#
# This is complementary to the Bandit source (SAST) scan; it does not replace
# it. The gate fails closed if pip-audit is unavailable or cannot produce a
# valid result, and never suppresses or downgrades a reported advisory.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PYTHON="${PYTHON:-python3}"

if ! command -v "$PYTHON" >/dev/null 2>&1; then
  echo "pip-audit gate: interpreter '$PYTHON' not found" >&2
  exit 2
fi

if ! "$PYTHON" -m pip_audit --version >/dev/null 2>&1; then
  echo "pip-audit gate: pip-audit is not installed; install sdk/python/requirements-security.txt" >&2
  exit 2
fi

status=0
for input in sdk/python/requirements-release.txt sdk/python/requirements-runtime.txt; do
  echo "== pip-audit: $input =="
  # --no-deps: the inputs are complete pinned closures, so audit exactly the
  #            pinned set rather than re-resolving from the network.
  # --strict:  fail if any dependency cannot be audited (fail closed).
  # --progress-spinner off: deterministic, CI-friendly output.
  if ! "$PYTHON" -m pip_audit -r "$input" --no-deps --strict --progress-spinner off; then
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  echo "pip-audit gate: FAILED" >&2
else
  echo "pip-audit gate: OK"
fi
exit "$status"
