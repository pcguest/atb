# ATB quickstart

This guide is for a developer who wants to record and verify an AI workflow decision for the
first time. It gets you to your first bundle in 5 minutes and requires no backend or network
access. At the end you will have a verified `.atb` bundle on disk and a working understanding of
the record-and-verify cycle.

## 1. Your first bundle in 5 minutes

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042","actor_id_hash":"sha256-user-1042","purpose_tag":"support-triage"}'
atb append ai.action.precommit --data='{"action_id":"act-1042","action_type":"route_case","action_parameters_digest":"sha256-route-tier-2","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-severity-routing","policy_version":"2026-04","decision":"deny","decision_reason_codes":["sev2_requires_manual_review"],"subject_id_hash":"sha256-user-1042","action_id":"act-1042"}'
atb snapshot incident_review_failed
atb verify --profile atb.profile.policy_decision --format json
atb trust-report --profile atb.profile.policy_decision --format markdown
```

The Go CLI is the primary install path. The Python and TypeScript
packages are SDKs that write the same `.atb` format.

This is the canonical ATB workflow: the AI workflow itself can fail
while the audit evidence still verifies cleanly. Use the snapshot name
to label workflow state. `atb verify` checks the hash chain and any
recorded bundle signature or anchor evidence. `atb bundle new` is the
explicit alias for `atb init`; both are supported. The example uses
`atb.profile.policy_decision` because the recorded evidence shows a
request, a pending action, and the policy outcome that blocked it. Use an
explicit `--profile` for first-run verification so the result is stable
even when a bundle contains events that could match more than one built-in
profile. For
the full local review and export path, continue with the [Incident review
workflow](./guides/incident-review-workflow.md).

### What the output means

Abbreviated `atb verify --profile atb.profile.policy_decision --format json` output:

```json
{
  "profile_id": "atb.profile.policy_decision",
  "pass": true,
  "cas_score": 0.70,
  "cas_grade": "Medium",
  "critical_failures": [],
  "residual_risk": "Medium"
}
```

- `pass: true` means the bundle chain is intact and the selected profile passed its required checks.
- `cas_score` and `cas_grade` describe how much of the expected evidence ATB can see for that workflow.
- `residual_risk` summarises what is still weak or missing in the recorded evidence, not whether the underlying decision was correct.

If you want the full integrity fields, run `atb verify --profile atb.profile.policy_decision --json`
and check `integrity.chain_valid: true`.

### Common first-run outcomes

- `pass: true` with a populated `profile_id` means the selected profile passed and the bundle chain is intact.
- `profile_id: ""` means the bundle verified for integrity, but no workflow profile was selected or matched. This is common for manifest-only or zero-event bundles.
- `pass: false` with non-empty `critical_failures` means the selected profile is missing required evidence.
- `residual_risk: "Critical"` means do not treat the bundle as trustworthy evidence until you inspect the failure. Use `atb verify --json` when you need the full integrity report.

### Verify against a specific profile

Use `--profile <id>` to evaluate against a specific built-in profile.
See `atb events --profile <id>` for the required event types per
profile. Local YAML definitions are also supported via
`--profile ./path/to/profile.yaml`:

```bash
atb events --profile atb.profile.rag_answer
atb verify --bundle run.atb/bundle.atb --profile atb.profile.privileged_tool_action
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --format json
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --json
atb verify --bundle run.atb/bundle.atb --profile ./profiles/release-review.yaml
```

Use `--format json` for the compact verifier report (`pass`, `cas_score`, `cas_grade`,
`critical_failures`, `residual_risk`). Use `--json` for the full report, including
`integrity.chain_valid` and the full CAS breakdown.

## 2. Installation options

### Go CLI

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

### Python SDK

```bash
pip install atb-sdk
```

### TypeScript SDK

```bash
npm install @pcguest/atb-sdk
```

The Python and TypeScript packages are SDKs only. Their installed `atb`
command is a compatibility stub that prints Go CLI install guidance and
will be removed in a future major release.

## 3. Record your first bundle (Python SDK)

```python
from atb import Bundle

bundle = Bundle()

bundle.append("ai.request.received", {
    "request_id": "req-001",
    "actor_id_hash": "sha256-user-001",
    "purpose_tag": "rag_answer",
}, timestamp="2026-03-03T10:00:00Z")

bundle.append("ai.model.invoked", {
    "model_provider": "openai",
    "model_id": "gpt-4",
    "model_parameters_digest": "sha256-params-001",
    "prompt_digest": "sha256-prompt-001",
}, timestamp="2026-03-03T10:00:02Z")

bundle.append("ai.model.output", {
    "output_digest": "sha256-output-001",
    "output_format": "text",
}, timestamp="2026-03-03T10:00:03Z")

bundle.save("my-trace.atb")
bundle.verify()
```

## 4. Open the local review UI

```bash
# Default bundle path
atb view

# Custom bundle path
atb view my-trace.atb --port 8080

# Evaluate against a profile at startup (shows profile/CAS summary in the viewer)
atb view --profile atb.profile.rag_answer
atb view --bundle run.atb/bundle.atb --profile ./profiles/custom.yaml
```

`atb view` opens one local review surface for one bundle at a time. `--profile` runs verify at
startup and makes the profile and CAS summary available immediately in the UI. Without
`--profile`, use the "Run verify" button in the UI to trigger `POST /api/v1/bundle/verify`.
The summary shows profile ID, pass or fail, completeness (CAS) score and grade, chain and anchor
status, and any critical obligation failures.

`atb view` requires building from source to include the embedded review UI:
`cd web && npm ci && npm run build && cd .. && go build -o atb ./cmd/atb`

If you install with `go install`, `atb view` serves a minimal install-guidance page with no
bundle data. Build from source when you need the embedded review UI.

Security note: `atb view` binds to `127.0.0.1` by default. All API endpoints require a
session token generated at startup and delivered in the browser URL fragment. Do not expose
the viewer on a non-loopback interface.

Dashboard details:

- [Dashboard specification](./spec-dashboard.md)

## 5. Incident evidence export

```bash
atb export --format compliance --output incident-review-evidence.zip
```

The compliance evidence export is the strongest default incident review
pack because it includes the verified bundle, trust report,
verification report, checksums, and core reference docs.

## 6. Compliance exports

```bash
atb config retention --days 90
atb archive
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
atb export --format soc2 --output soc2-evidence.zip --with-verify
# Writes soc2-evidence.zip and soc2-evidence.zip.verify.json
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

Reference docs:

- [Retention](./compliance/retention.md)
- [Compliance export overview](./compliance/export.md)
- [SOC 2 export specification](./compliance/soc2.md)
- [GDPR export specification](./compliance/gdpr.md)
- [Incident review workflow](./guides/incident-review-workflow.md)

## 7. AI integrations

```python
from atb.langchain_callback import ATBCallbackHandler

handler = ATBCallbackHandler(privacy_mode="hash")
```

Integration docs:

- [LangChain integration](./integrations/langchain.md)
- [Vercel AI SDK integration](./integrations/vercel-ai.md)

## 8. Next steps

- [Go example](../examples/go/README.md)
- [Python SDK](../sdk/python/README.md)
- [TypeScript SDK](../sdk/typescript/README.md)
- [Docs home](./README.md)
- [Customer handoff workflow](./guides/customer-handoff-workflow.md)
- [ATB specification v1.0](./spec-v1.0.md)
- [Security model](./security.md)
- [Contributing](../CONTRIBUTING.md)
