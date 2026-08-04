#!/usr/bin/env bash
# pin-actions.sh used to rewrite workflow action pins from a hardcoded map.
# That map drifted behind the repository's current SHA pins and would
# downgrade Actions if run. Refusing to mutate workflows is intentional.
set -euo pipefail

cat >&2 <<'EOF'
pin-actions.sh is disabled.

Workflows in .github/workflows/ already pin actions to full commit SHAs.
The historical pin map in this script was stale (v4/v5-era tags) and would
replace current pins with older SHAs if executed.

To update an action pin:
  1. Look up the release tag's commit SHA on GitHub.
  2. Edit the workflow uses: line to owner/repo@<full-sha> # vx.y.z
  3. Open a PR; do not batch-rewrite pins from this script.
EOF
exit 1
