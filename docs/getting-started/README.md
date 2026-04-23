# Getting Started

Start here if you are new to ATB.

## First 5 minutes

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042","actor_id_hash":"hash-agent-a","purpose_tag":"support-triage"}'
atb append ai.action.precommit --data='{"action_id":"act-1042","action_type":"route_case","action_parameters_digest":"sha256-route-tier-2","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-severity-routing","policy_version":"2026-04","decision":"deny","decision_reason_codes":["sev2_requires_manual_review"],"subject_id_hash":"hash-agent-a","action_id":"act-1042"}'
atb snapshot incident_review_failed
atb verify --profile atb.profile.policy_decision --format json
```

## Recommended path

1. [Incident Review Workflow](../guides/incident-review-workflow.md)
2. [Customer Handoff Workflow](../guides/customer-handoff-workflow.md)
3. [Quickstart](../quickstart.md)
4. [Compliance Export Overview](../compliance/export.md)
5. [AI Integrations](../guides/README.md)

## Install options

- Go CLI: `go install github.com/pcguest/atb/cmd/atb@latest`
- Python SDK: `pip install atb-sdk`
- TypeScript SDK: `npm install @pcguest/atb-sdk`

The Go CLI is the authoritative CLI path. The Python and TypeScript packages are SDKs only. Their installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.
