#!/usr/bin/env bash
# Run govulncheck as a strict release gate. Any reported vulnerability or
# scanner failure is blocking; the gate must never turn a real finding green.
set -euo pipefail

GOVULNCHECK_BIN="${GOVULNCHECK_BIN:-govulncheck}"
if ! command -v "$GOVULNCHECK_BIN" >/dev/null 2>&1; then
  echo "govulncheck.sh: govulncheck binary not found: $GOVULNCHECK_BIN" >&2
  exit 2
fi

OUT=$(mktemp)
trap 'rm -f "$OUT"' EXIT

packages=()
while IFS= read -r package; do
  packages+=("$package")
done < <(go list ./... | grep -v '/node_modules/')

set +e
"$GOVULNCHECK_BIN" -json "${packages[@]}" > "$OUT"
status=$?
set -e

if [ "$status" -eq 0 ]; then
  echo "govulncheck: no vulnerabilities reported"
  exit 0
fi

echo "govulncheck: vulnerabilities reported or scan failed — failing build" >&2
cat "$OUT" >&2
exit "$status"
