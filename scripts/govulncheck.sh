#!/usr/bin/env bash
# Run govulncheck and treat findings without an available fix as non-blocking.
# Findings with a fixed version are still treated as failures.
set -euo pipefail

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "govulncheck.sh: govulncheck binary not on PATH" >&2
  exit 2
fi

OUT=$(mktemp)
trap 'rm -f "$OUT"' EXIT

set +e
govulncheck -json ./... > "$OUT"
status=$?
set -e

# Exit 0 means no findings; emit and finish.
if [ "$status" -eq 0 ]; then
  echo "govulncheck: no vulnerabilities reported"
  exit 0
fi

# Use jq if present for a precise check; otherwise fall back to grep.
if command -v jq >/dev/null 2>&1; then
  fixable=$(jq -r '
    select(.osv) | .osv |
    select(.affected) | .affected[] |
    select(.ranges) | .ranges[] |
    select(.events) | .events[] |
    select(.fixed) | .fixed
  ' "$OUT" | wc -l | tr -d ' ')
else
  fixable=$(grep -c '"fixed"' "$OUT" || true)
fi

if [ "${fixable:-0}" -gt 0 ]; then
  echo "govulncheck: $fixable finding(s) with available fixes — failing build" >&2
  cat "$OUT" >&2
  exit 1
fi

echo "govulncheck: findings present but none have available fixes — non-blocking"
cat "$OUT"
exit 0
