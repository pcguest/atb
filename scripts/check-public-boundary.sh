#!/usr/bin/env bash
set -euo pipefail

root="$(git rev-parse --show-toplevel)"
cd "$root"

fail() {
  echo "public-boundary check failed: $*" >&2
  exit 1
}

forbidden_paths=(
  "W3-HARNESS-BASELINE-REPORT.md"
  "w3-practitioner-dryrun.md"
  "docs/concepts/acquisition-modes.md"
  "docs/releases/v1.15.3-convergence.md"
  "docs/releases/v1.16.0-claims-hygiene.md"
  "docs/releases/v1.16.0-clean-consumer-proof.md"
  "docs/releases/v1.16.0-cli-view-treatment.md"
  "docs/releases/v1.16.0-convergence-plan.md"
  "docs/releases/v1.16.0-firefox-a11y.md"
  "docs/releases/v1.16.0-freeze-adversary.md"
  "docs/releases/v1.16.0-freeze-candidate-state.md"
  "docs/releases/v1.16.0-freeze-ledger.md"
  "docs/releases/v1.16.0-gold-gate.md"
  "docs/releases/v1.16.0-live-investigation.md"
  "docs/releases/v1.16.0-product-convergence.md"
  "docs/releases/v1.16.0-semantic-freeze-proof.md"
  "docs/releases/v1.16.0-view-performance.md"
  "docs/maintainers/repository-inventory.md"
  "docs/maintainers/repository-inventory.json"
  "docs/maintainers/visual-system.md"
  "docs/development/operating-model.md"
  "docs/pilot/external-pilot-protocol.md"
  "docs/pilot/pilot-evidence-and-interview.md"
  "docs/research/ATB_MARKET_UI_APPLICATION.md"
  "docs/research/MARKET_UI_PATTERN_MATRIX.md"
  "docs/research/PROFESSIONAL_UI_GRAMMAR.md"
)
for path in "${forbidden_paths[@]}"; do
  if [[ -e "$path" || -L "$path" ]]; then
    fail "private or superseded working material is present in the public tree: $path"
  fi
done
if git ls-files -- .opencode/ | grep -q .; then
  fail "personal OpenCode configuration must not be tracked in the public tree"
fi

if git ls-files --error-unmatch -- .venv >/dev/null 2>&1; then
  venv_status="$(git status --short -- .venv)"
  if [[ "$venv_status" != " D .venv" ]]; then
    fail "root .venv must be deleted from the tracked tree"
  fi
fi
if ! git check-ignore --quiet --no-index -- .venv; then
  fail "root .venv must be ignored whether it is a file, symlink, or directory"
fi

if git grep -nIE '/Users/[^/[:space:]]+/(atb|custos)(/|$)|file:///Users/' -- '*.md' '*.yaml' '*.yml'; then
  fail "machine-specific checkout paths found in public documentation"
fi

echo "public-boundary check passed"
