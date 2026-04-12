#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ATB_BIN="${ATB_BIN:-atb}"

if ! command -v "$ATB_BIN" >/dev/null 2>&1; then
  echo "atb CLI not found. Install it with: go install github.com/pcguest/atb/cmd/atb@latest" >&2
  exit 1
fi

rm -rf run.atb
"$ATB_BIN" bundle new >/dev/null
printf '✓ Initialised ATB bundle at run.atb/bundle.atb\n'

"$ATB_BIN" append ai.request.received \
  --data '{"request_id":"req-001","actor_id_hash":"hash-user-01","purpose_tag":"privileged_tool_demo"}' >/dev/null
printf '✓ Appended event #1 [ai.request.received]\n'

"$ATB_BIN" append ai.action.precommit \
  --data '{"action_id":"act-001","action_type":"delete_records","action_parameters_digest":"sha256-params-abc","target_resource_id":"table-users","intended_effect":"remove_test_rows"}' >/dev/null
printf '✓ Appended event #2 [ai.action.precommit]\n'

"$ATB_BIN" append ai.policy.decision \
  --data '{"policy_id":"pol-01","policy_version":"1.0","decision":"allow","decision_reason_codes":["within_scope"],"subject_id_hash":"hash-user-01","action_id":"act-001"}' >/dev/null
printf '✓ Appended event #3 [ai.policy.decision]\n'

"$ATB_BIN" append ai.action.executed \
  --data '{"action_id":"act-001","execution_outcome":"success","tool_receipt_digest":"sha256-receipt-def"}' >/dev/null
printf '✓ Appended event #4 [ai.action.executed]\n'

"$ATB_BIN" append ai.human.approval \
  --data '{"approval_id":"apr-001","approver_id_hash":"hash-approver-01","approval_outcome":"approved","justification_digest":"sha256-justification-jkl","action_id":"act-001"}' >/dev/null
printf '✓ Appended event #5 [ai.human.approval]\n'

"$ATB_BIN" append ai.action.committed \
  --data '{"action_id":"act-001","commit_outcome":"committed","sink_receipt_digest":"sha256-sink-ghi"}' >/dev/null
printf '✓ Appended event #6 [ai.action.committed]\n'

VERIFY_JSON="$("$ATB_BIN" verify --profile atb.profile.privileged_tool_action --json)"

if command -v jq >/dev/null 2>&1; then
  printf '%s\n' "$VERIFY_JSON" | jq .
else
  printf '%s\n' "$VERIFY_JSON"
fi

"$ATB_BIN" trust-report --profile atb.profile.privileged_tool_action --format text

CAS_LABEL="$(printf '%s\n' "$VERIFY_JSON" | grep -o '"grade":[[:space:]]*"[^"]*"' | head -n1 | sed 's/.*"grade":[[:space:]]*"\([^"]*\)".*/\1/')"

if [[ -z "$CAS_LABEL" ]]; then
  echo "Could not extract CAS grade from verify output." >&2
  exit 1
fi

case "$CAS_LABEL" in
  High)
    CAS_GRADE="A"
    ;;
  Medium)
    CAS_GRADE="B"
    ;;
  Low)
    CAS_GRADE="C"
    ;;
  Insufficient)
    CAS_GRADE="F"
    ;;
  *)
    CAS_GRADE="$CAS_LABEL"
    ;;
esac

printf '=== CAS grade: %s ===\n' "$CAS_GRADE"
printf 'Quickstart complete. Bundle is at run.atb/bundle.atb\n'
