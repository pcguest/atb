#!/bin/bash
# Regenerate examples/bundles/demo-workflow/demo-workflow.atb (signed + snapshotted).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ATB_BIN="${ATB_BIN:-}"
if [[ -z "$ATB_BIN" && -x "$ROOT/atb" ]]; then
  ATB_BIN="$ROOT/atb"
fi
ATB_BIN="${ATB_BIN:-atb}"
BUNDLE="$OUT_DIR/demo-workflow.atb"
KEY="$OUT_DIR/demo-signing-key.pem"

cd "$ROOT"
GOCACHE="${GOCACHE:-$ROOT/.gocache/dev}" GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.4}" \
  go run ./examples/bundles/demo-workflow/

if [[ ! -f "$KEY" ]]; then
  "$ATB_BIN" keygen --out-dir "$OUT_DIR" >/dev/null
  mv "$OUT_DIR/atb-key.pem" "$KEY"
  mv "$OUT_DIR/atb-key.pub.pem" "$OUT_DIR/demo-signing-key.pub.pem"
fi

"$ATB_BIN" sign --bundle "$BUNDLE" --key "$KEY" >/dev/null
"$ATB_BIN" snapshot support_escalation_resolved --bundle "$BUNDLE" >/dev/null

for profile in atb.profile.policy_decision atb.profile.human_override; do
  "$ATB_BIN" verify --bundle "$BUNDLE" --profile "$profile" --format json >/dev/null
done

echo "✓ Regenerated $BUNDLE (signed + snapshotted)"
