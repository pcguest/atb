#!/usr/bin/env bash
# Split provability enhancement work into 12 logical commits (no co-authors).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

AUTHOR='Paddy Guest <patrickcguest@proton.me>'

commit() {
  local msg="$1"
  shift
  if [[ $# -eq 0 ]]; then
    echo "split-provability-commits: no files staged for: $msg" >&2
    exit 1
  fi
  git add "$@"
  if git diff --cached --quiet; then
    echo "split-provability-commits: nothing staged for: $msg" >&2
    exit 1
  fi
  git commit --author="$AUTHOR" -m "$msg"
  echo "✓ $msg"
}

git reset HEAD >/dev/null 2>&1 || true

commit 'docs: fix roadmap path and refresh v1.11 status' \
  docs/roadmap.md docs/ROADMAP.md

commit 'docs: add provability ladder and public surface notice' \
  docs/provability-ladder.md \
  docs/public-surface.md \
  docs/guides/instrumentation-checklist.md \
  docs/why-atb.md \
  docs/security.md \
  README.md

commit 'fix(cli): hide generic import stub from user-facing surface' \
  cmd/atb/import.go \
  cmd/atb/import_test.go

git add \
  examples/README.md \
  examples/bundles/project-bootstrap.atb \
  examples/bundles/generate.sh \
  docs/guides/tamper-demo.md \
  internal/capture/chatlog.go \
  internal/capture/chatlog_contract_test.go \
  sdk/python/atb/capture.py \
  sdk/typescript/src/capture.ts \
  sdk/typescript/src/index.ts
git add -f examples/bundles/project-bootstrap-tampered.atb
git commit --author="$AUTHOR" -m "$(cat <<'EOF'
examples: add verifiable bundle fixtures including tampered pair

Include generate.sh, tamper-demo guide, and openai-jsonl chatlog import parity
across Go and SDK helpers so demo bundles and retrospective import stay aligned.
EOF
)"

commit 'verify: add structured provability_gaps to report JSON' \
  internal/verify/provability_gaps.go \
  internal/verify/provability_gaps_test.go \
  internal/verify/report.go \
  internal/verify/evaluate.go \
  internal/verify/verify.go \
  internal/verify/trust_report.go \
  internal/verify/export_sidecar.go \
  internal/verify/export_sidecar_test.go \
  cmd/atb/export.go \
  cmd/atb/export_verify_test.go \
  docs/integrations/siem-grc.md \
  pkg/api/v1/types.go \
  pkg/api/v1/handlers.go \
  cmd/atb/view.go \
  web/lib/schemas/profile.ts \
  web/components/dashboard/ProfileCAS.tsx \
  docs/spec-dashboard.md

commit 'profiles: rewrite blind_spots as conditional mitigations' \
  internal/profiles/templates/background_automation.yaml \
  internal/profiles/templates/data_export.yaml \
  internal/profiles/templates/human_override.yaml \
  internal/profiles/templates/policy_decision.yaml \
  internal/profiles/templates/privileged_tool_action.yaml \
  internal/profiles/templates/rag_answer.yaml

commit 'verify: add strict source signature gate flag' \
  cmd/atb/verify.go \
  internal/verify/strict_signatures_test.go \
  docs/key-management.md

commit 'profiles: promote temporal rules to critical where required' \
  cmd/atb/trust_report_snapshot_test.go \
  cmd/atb/trust_report_test.go \
  cmd/atb/export_compliance_manifest_test.go \
  cmd/atb/testdata/export_compliance_manifest.golden.json

commit 'corroboration: add second adapter with tests' \
  internal/corroboration/file_receipt.go \
  internal/corroboration/file_receipt_test.go \
  cmd/atb/main.go

commit 'scripts: add public demo export manifest and scanner' \
  scripts/export-public-demo.sh \
  scripts/public-export.manifest.yaml

commit 'ci: add Cypress E2E and export workflow_dispatch job' \
  .github/workflows/ci.yml \
  .github/workflows/export-public-demo.yml \
  Makefile \
  cmd/atb/platform.go \
  docs/case-study-remediation-audit.md \
  docs/launch/assets/README.md \
  docs/launch/assets/atb-verify-demo.gif \
  docs/launch/assets/atb-verify-report.png \
  scripts/record-launch-demo.sh

commit 'web: align landing with v1.11 scope and provability messaging' \
  web/app/layout.tsx \
  web/components/Hero.tsx \
  web/components/CurrentScope.tsx \
  web/components/Footer.tsx

echo ""
echo "Created commits:"
git log -12 --format='%h %an <%ae> | %s'

echo ""
echo "Remaining unstaged/uncommitted:"
git status --short
