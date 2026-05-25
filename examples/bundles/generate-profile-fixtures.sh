#!/usr/bin/env bash
# Regenerate profile fixture bundles under examples/bundles/profiles/.
# Requires a built atb binary (set ATB_BIN or ensure `atb` is on PATH).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="${ROOT}/examples/bundles/profiles"
ATB_BIN="${ATB_BIN:-atb}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if ! command -v "$ATB_BIN" >/dev/null 2>&1; then
  echo "error: ATB_BIN=$ATB_BIN not found; run 'go build -o /tmp/atb ./cmd/atb' and export ATB_BIN=/tmp/atb" >&2
  exit 1
fi

mkdir -p "$OUT"

new_bundle() {
  rm -rf "$WORK/run.atb"
  mkdir -p "$WORK"
  cd "$WORK"
  "$ATB_BIN" bundle new >/dev/null
}

append_json() {
  local event_type="$1"
  local data="$2"
  "$ATB_BIN" append "$event_type" --data "$data" >/dev/null
}

verify_pass() {
  local name="$1"
  local profile="$2"
  "$ATB_BIN" verify --bundle "$WORK/run.atb/bundle.atb" --profile "$profile" --format json >/dev/null
  cp "$WORK/run.atb/bundle.atb" "$OUT/${name}-pass.atb"
  echo "  ✓ ${name}-pass.atb (${profile})"
}

verify_fail() {
  local name="$1"
  local profile="$2"
  if "$ATB_BIN" verify --bundle "$WORK/run.atb/bundle.atb" --profile "$profile" --format json >/dev/null 2>&1; then
    echo "error: expected ${name}-fail.atb to fail profile ${profile}" >&2
    exit 1
  fi
  cp "$WORK/run.atb/bundle.atb" "$OUT/${name}-fail.atb"
  echo "  ✓ ${name}-fail.atb (${profile}, expected fail)"
}

echo "Generating profile fixtures in ${OUT}"

# privileged_tool_action
new_bundle
append_json ai.request.received '{"request_id":"req-pta-pass","actor_id_hash":"sha256:actor-pta","purpose_tag":"privileged_tool_action"}'
append_json ai.action.precommit '{"action_id":"act-pta-pass","action_type":"deploy_change","action_parameters_digest":"sha256:params-pta","target_resource_id":"svc-prod","intended_effect":"deploy approved build"}'
append_json ai.policy.decision '{"policy_id":"pol-pta","policy_version":"2026-04","decision":"allow","decision_reason_codes":["approved"],"subject_id_hash":"sha256:subject-pta","action_id":"act-pta-pass"}'
append_json ai.action.executed '{"action_id":"act-pta-pass","execution_outcome":"success","tool_receipt_digest":"sha256:receipt-pta"}'
append_json ai.human.approval '{"approval_id":"appr-pta-pass","approver_id_hash":"sha256:approver-pta","approval_outcome":"approved","justification_digest":"sha256:just-pta","action_id":"act-pta-pass"}'
append_json ai.action.committed '{"action_id":"act-pta-pass","commit_outcome":"success","sink_receipt_digest":"sha256:sink-pta"}'
verify_pass privileged_tool_action atb.profile.privileged_tool_action

new_bundle
append_json ai.request.received '{"request_id":"req-pta-fail","actor_id_hash":"sha256:actor-pta","purpose_tag":"privileged_tool_action"}'
append_json ai.policy.decision '{"policy_id":"pol-pta","policy_version":"2026-04","decision":"allow","decision_reason_codes":["approved"],"subject_id_hash":"sha256:subject-pta","action_id":"act-pta-fail"}'
append_json ai.action.executed '{"action_id":"act-pta-fail","execution_outcome":"success","tool_receipt_digest":"sha256:receipt-pta"}'
verify_fail privileged_tool_action atb.profile.privileged_tool_action

# rag_answer
new_bundle
append_json ai.request.received '{"request_id":"req-rag-pass","actor_id_hash":"sha256:actor-rag","purpose_tag":"rag_answer"}'
append_json ai.model.invoked '{"model_provider":"openai","model_id":"gpt-4o","model_parameters_digest":"sha256:params-rag","prompt_digest":"sha256:prompt-rag"}'
append_json ai.model.output '{"output_digest":"sha256:output-rag","output_format":"text/plain"}'
append_json ai.response.sent '{"request_id":"req-rag-pass","output_digest":"sha256:output-rag"}'
verify_pass rag_answer atb.profile.rag_answer

new_bundle
append_json ai.request.received '{"request_id":"req-rag-fail","actor_id_hash":"sha256:actor-rag","purpose_tag":"rag_answer"}'
append_json ai.model.output '{"output_digest":"sha256:output-rag","output_format":"text/plain"}'
verify_fail rag_answer atb.profile.rag_answer

# data_export
new_bundle
append_json ai.request.received '{"request_id":"req-export-pass","actor_id_hash":"sha256:actor-export","purpose_tag":"data_export"}'
append_json ai.policy.decision '{"policy_id":"pol-export","policy_version":"2026-04","decision":"allow","decision_reason_codes":["export_allowed"],"subject_id_hash":"sha256:subject-export","action_id":"act-export-pass"}'
append_json data.export.precommit '{"action_id":"act-export-pass","action_type":"export_data","action_parameters_digest":"sha256:params-export","target_resource_id":"dataset-1","intended_effect":"export approved dataset"}'
append_json data.export.executed '{"action_id":"act-export-pass","execution_outcome":"success","tool_receipt_digest":"sha256:receipt-export"}'
append_json ai.human.approval '{"approval_id":"appr-export-pass","approver_id_hash":"sha256:approver-export","approval_outcome":"approved","justification_digest":"sha256:just-export","action_id":"act-export-pass"}'
verify_pass data_export atb.profile.data_export

new_bundle
append_json ai.request.received '{"request_id":"req-export-fail","actor_id_hash":"sha256:actor-export","purpose_tag":"data_export"}'
append_json ai.policy.decision '{"policy_id":"pol-export","policy_version":"2026-04","decision":"allow","decision_reason_codes":["export_allowed"],"subject_id_hash":"sha256:subject-export","action_id":"act-export-fail"}'
append_json data.export.executed '{"action_id":"act-export-fail","execution_outcome":"success","tool_receipt_digest":"sha256:receipt-export"}'
verify_fail data_export atb.profile.data_export

# policy_decision
new_bundle
append_json ai.request.received '{"request_id":"req-policy-pass","actor_id_hash":"sha256:actor-policy","purpose_tag":"policy_decision"}'
append_json ai.action.precommit '{"action_id":"act-policy-pass","action_type":"approve_change","action_parameters_digest":"sha256:params-policy"}'
append_json ai.policy.decision '{"policy_id":"pol-policy","policy_version":"2026-04","decision":"allow","decision_reason_codes":["approved"],"subject_id_hash":"sha256:subject-policy","action_id":"act-policy-pass"}'
verify_pass policy_decision atb.profile.policy_decision

new_bundle
append_json ai.request.received '{"request_id":"req-policy-fail","actor_id_hash":"sha256:actor-policy","purpose_tag":"policy_decision"}'
append_json ai.action.precommit '{"action_id":"act-policy-fail","action_type":"approve_change","action_parameters_digest":"sha256:params-policy"}'
verify_fail policy_decision atb.profile.policy_decision

# human_override
new_bundle
append_json ai.request.received '{"request_id":"req-override-pass","actor_id_hash":"sha256:actor-override","purpose_tag":"human_override"}'
append_json ai.human.approval '{"approval_id":"appr-override-pass","approver_id_hash":"sha256:approver-override","approval_outcome":"approved","justification_digest":"sha256:just-override","action_id":"act-override-pass"}'
append_json ai.action.precommit '{"action_id":"act-override-pass","action_type":"override_action","action_parameters_digest":"sha256:params-override","target_resource_id":"svc-1","intended_effect":"run approved override"}'
append_json ai.action.executed '{"action_id":"act-override-pass","execution_outcome":"success","tool_receipt_digest":"sha256:receipt-override"}'
verify_pass human_override atb.profile.human_override

new_bundle
append_json ai.request.received '{"request_id":"req-override-fail","actor_id_hash":"sha256:actor-override","purpose_tag":"human_override"}'
append_json ai.action.precommit '{"action_id":"act-override-fail","action_type":"override_action","action_parameters_digest":"sha256:params-override","target_resource_id":"svc-1","intended_effect":"run override"}'
append_json ai.action.executed '{"action_id":"act-override-fail","execution_outcome":"success","tool_receipt_digest":"sha256:receipt-override"}'
verify_fail human_override atb.profile.human_override

# background_automation
new_bundle
append_json ai.job.scheduled '{"job_id":"job-bg-pass","job_type":"nightly_sync","trigger_source":"cron","scheduled_by_id_hash":"sha256:scheduler-bg"}'
append_json ai.job.started '{"job_id":"job-bg-pass","worker_id_hash":"sha256:worker-bg","started_at":"2026-05-24T12:00:00Z"}'
append_json ai.job.completed '{"job_id":"job-bg-pass","outcome":"success","completion_reason":"completed"}'
verify_pass background_automation atb.profile.background_automation

new_bundle
append_json ai.job.scheduled '{"job_id":"job-bg-fail","job_type":"nightly_sync","trigger_source":"cron","scheduled_by_id_hash":"sha256:scheduler-bg"}'
append_json ai.job.started '{"job_id":"job-bg-fail","worker_id_hash":"sha256:worker-bg","started_at":"2026-05-24T12:00:00Z"}'
verify_fail background_automation atb.profile.background_automation

echo "Done. Generated 12 profile fixtures in ${OUT}"
