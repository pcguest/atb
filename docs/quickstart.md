# ATB Quickstart

ATB provides tamper-evident, verifiable audit trails for AI workflows.

## 1. Quick Start (Incident Review)

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append agent.run --data='{"workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append policy.alert --data='{"check":"pii_redaction","outcome":"fail","ticket_id":"case-1042","reason":"customer_email_left_visible"}'
atb snapshot incident_review --gate fail
atb verify
atb trust-report --format markdown
```

The Go CLI is the primary install path. The Python and TypeScript packages are SDKs that write the same `.atb` format.

This is the canonical ATB workflow: the AI workflow itself can fail while the audit evidence still verifies cleanly. `atb bundle new` is the explicit alias for `atb init`; both are supported. For the full local review and export path, continue with the [Incident Review Workflow](./guides/incident-review-workflow.md).

### Verify Against a Specific Profile

ATB can auto-detect built-in profiles from the bundle contents. Use `--profile` when you want to lock verification to a specific built-in profile or a local YAML definition:

```bash
atb verify --bundle run.atb/bundle.atb --profile atb.profile.privileged_tool_action
atb verify --bundle run.atb/bundle.atb --profile ./profiles/release-review.yaml
```

## 2. Installation Options

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

The Python and TypeScript packages are SDKs only. Their installed `atb` command is a compatibility stub that prints Go CLI install guidance and will be removed in a future major release.

## 3. Record Your First Trace (Python SDK)

```python
from atb import Bundle

bundle = Bundle()

bundle.append("agent.prompt", {
    "timestamp": "2026-03-03T10:00:00Z",
    "actor": "planner",
    "model": "gpt-4",
    "prompt": "Outline a blog post about AI safety",
})

bundle.append("agent.response", {
    "timestamp": "2026-03-03T10:00:02Z",
    "actor": "planner",
    "tokens": 142,
    "output": "Draft outline...",
})

bundle.append("snapshot.build", {
    "gate": "pass",
    "timestamp": "2026-03-03T10:00:03Z",
})

bundle.save("my-trace.atb")
bundle.verify()
```

## 4. Open the Visual Dashboard

```bash
# Default bundle path
atb view --ui-experimental

# Custom bundle path
atb view my-trace.atb --port 8080 --ui-experimental

# Privacy reveal auditing is on by default
atb view --bundle my-trace.atb --ui-experimental
```

Plain `atb view` still serves the legacy local viewer at `/`. The role-based dashboard UI is available behind `--ui-experimental`.

Dashboard details:

- [Dashboard Specification](./spec-dashboard.md)

## 5. Incident Evidence Export

```bash
atb export --format compliance --output incident-review-evidence.zip
```

The compliance evidence export is the strongest default incident review pack because it includes the verified bundle, trust report, verification report, checksums, and core reference docs.

## 6. Compliance Exports

```bash
atb config retention --days 90
atb archive
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

Reference docs:

- [Retention](./compliance/retention.md)
- [Compliance Export Overview](./compliance/export.md)
- [SOC 2 Export Specification](./compliance/soc2.md)
- [GDPR Export Specification](./compliance/gdpr.md)
- [Incident Review Workflow](./guides/incident-review-workflow.md)

## 7. AI Integrations

```python
from atb.langchain_callback import ATBCallbackHandler

handler = ATBCallbackHandler(privacy_mode="hash")
```

Integration docs:

- [LangChain Integration](./integrations/langchain.md)
- [Vercel AI SDK Integration](./integrations/vercel-ai.md)

## 8. Next Steps

- [Docs Home](./README.md)
- [Customer Handoff Workflow](./guides/customer-handoff-workflow.md)
- [ATB Specification v1.0](./spec-v1.0.md)
- [Security Model](./security.md)
- [Contributing](../CONTRIBUTING.md)
