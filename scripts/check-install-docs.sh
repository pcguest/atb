#!/usr/bin/env bash
# check-install-docs.sh — guard first-run install docs against drifting back to
# unpinned or incomplete evaluator install guidance.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  printf 'install docs check: %s\n' "$1" >&2
  exit 1
}

require_contains() {
  local file="$1" needle="$2" label="$3"
  grep -Fq "$needle" "$ROOT_DIR/$file" || fail "$file missing $label"
}

for file in README.md docs/quickstart.md docs/guides/capture.md; do
  if grep -Fq 'github.com/pcguest/atb/cmd/atb@latest' "$ROOT_DIR/$file"; then
    fail "$file uses unpinned go install @latest in evaluator install docs"
  fi
done

require_contains README.md 'make build' 'complete source build guidance'
require_contains README.md 'github.com/pcguest/atb/cmd/atb@v1.15.1' 'pinned viewer-minimal Go module install'
require_contains README.md 'Tenon compatibility matrix' 'compatibility matrix reference'
require_contains README.md 'do not infer PyPI, npm, Go module proxy, or GitHub Release availability' 'source-versus-publication warning'

require_contains docs/quickstart.md 'make build' 'complete source build guidance'
require_contains docs/quickstart.md 'github.com/pcguest/atb/cmd/atb@v1.15.1' 'pinned viewer-minimal Go module install'
require_contains docs/quickstart.md 'generated `./atb` binary' 'source uninstall guidance'
require_contains docs/quickstart.md 'do not infer PyPI, npm, Go module proxy, or GitHub Release availability' 'source-versus-publication warning'

require_contains docs/guides/capture.md 'github.com/pcguest/atb/cmd/atb@v1.15.1' 'pinned viewer-minimal Go module install'
require_contains docs/mortise-handoff.md 'Tenon compatibility matrix' 'cross-product compatibility reference'

printf 'ok: install docs are explicit and pinned\n'
