# ATB quickstart

ATB provides tamper-evident, verifiable audit trails for AI workflows.

## 1. Quick start (incident review)

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042","workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append ai.action.precommit --data='{"action_id":"act-1042","action_type":"route_case","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-pii-redaction","policy_version":"2026-04","decision":"deny","decision_reason_codes":["customer_email_visible"],"subject_id_hash":"sha256-user-1042","action_id":"act-1042"}'
atb snapshot incident_review_failed
atb verify
atb trust-report --profile atb.profile.privileged_tool_action --format markdown
```

The Go CLI is the primary install path. The Python and TypeScript
packages are SDKs that write the same `.atb` format.

This is the canonical ATB workflow: the AI workflow itself can fail
while the audit evidence still verifies cleanly. Use the snapshot name
to label workflow state. `atb verify` checks the hash chain and any
recorded bundle signature or anchor evidence. `atb bundle new` is the
explicit alias for `atb init`; both are supported. For the full local
review and export path, continue with the [Incident review
workflow](./guides/incident-review-workflow.md).

### Verify against a specific profile

Use `--profile <id>` to evaluate against a specific built-in profile.
See `atb events --profile <id>` for the required event types per
profile. Local YAML definitions are also supported via
`--profile ./path/to/profile.yaml`:

```bash
atb events --profile atb.profile.rag_answer
atb verify --bundle run.atb/bundle.atb --profile atb.profile.privileged_tool_action
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --json
# The JSON output includes a "cas" object with "grade" and profile sub-scores
# such as AC, EC, FC, GC, RC, SC, TC, and XC.
atb verify --bundle run.atb/bundle.atb --profile ./profiles/release-review.yaml
```

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

## 3. Record your first trace (Python SDK)

```python
from atb import Bundle

bundle = Bundle()

bundle.append("agent.prompt", {
    "actor": "planner",
    "model": "gpt-4",
    "prompt": "Outline a blog post about AI safety",
}, timestamp="2026-03-03T10:00:00Z")

bundle.append("agent.response", {
    "actor": "planner",
    "tokens": 142,
    "output": "Draft outline...",
}, timestamp="2026-03-03T10:00:02Z")

bundle.append("snapshot.build", {
    "gate": "pass",
}, timestamp="2026-03-03T10:00:03Z")

bundle.save("my-trace.atb")
bundle.verify()
```

## 4. Open the visual dashboard

```bash
# Default bundle path
atb view --ui-experimental

# Custom bundle path
atb view my-trace.atb --port 8080 --ui-experimental

# Privacy reveal auditing is on by default
atb view --bundle my-trace.atb --ui-experimental
```

Plain `atb view` still serves the legacy local viewer at `/`. The
role-based dashboard UI is available behind `--ui-experimental`.

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

- [Docs home](./README.md)
- [Customer handoff workflow](./guides/customer-handoff-workflow.md)
- [ATB specification v1.0](./spec-v1.0.md)
- [Security model](./security.md)
- [Contributing](../CONTRIBUTING.md)
