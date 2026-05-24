#!/usr/bin/env bash
# Regenerate examples/bundles/project-bootstrap.atb and tampered copy.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ATB_BIN="${ATB_BIN:-atb}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

cd "$WORK"
"$ATB_BIN" bundle new >/dev/null
"$ATB_BIN" append ai.request.received \
  --data '{"request_id":"req-bootstrap","actor_id_hash":"sha256-user-bootstrap","purpose_tag":"project_bootstrap"}' >/dev/null
"$ATB_BIN" append ai.action.precommit \
  --data '{"action_id":"act-bootstrap","action_type":"init_project","action_parameters_digest":"sha256-params-bootstrap","target_resource_id":"repo-bootstrap","intended_effect":"create_verified_fixture"}' >/dev/null
"$ATB_BIN" append ai.policy.decision \
  --data '{"policy_id":"pol-bootstrap","policy_version":"1","decision":"allow","decision_reason_codes":["demo_fixture"],"subject_id_hash":"sha256-user-bootstrap","action_id":"act-bootstrap"}' >/dev/null
"$ATB_BIN" snapshot project_bootstrap >/dev/null

mkdir -p "$ROOT/examples/bundles"
cp run.atb/bundle.atb "$ROOT/examples/bundles/project-bootstrap.atb"
cp run.atb/bundle.atb "$ROOT/examples/bundles/project-bootstrap-tampered.atb"
python3 - "$ROOT/examples/bundles/project-bootstrap-tampered.atb" <<'PY'
import sys
from pathlib import Path
p = Path(sys.argv[1])
data = bytearray(p.read_bytes())
for i, b in enumerate(data):
    if b == ord("b") and i > 200:
        data[i] = ord("X")
        break
p.write_bytes(data)
PY

"$ATB_BIN" verify --bundle "$ROOT/examples/bundles/project-bootstrap.atb" \
  --profile atb.profile.policy_decision --format json >/dev/null
echo "✓ Wrote examples/bundles/project-bootstrap.atb (+ tampered copy)"
