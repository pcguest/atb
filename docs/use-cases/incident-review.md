# Incident Review for Private AI Workflows

ATB is built for the moment after something goes wrong.

When an AI workflow misfires, most teams can retrieve logs. Fewer teams can show that the trace they are reviewing is complete, untampered, and still under their control. That is the gap ATB is designed to close.

## The Problem

Private AI workflows create a difficult trade-off during incident review:

- security and compliance teams want a trustworthy record
- engineering wants enough detail to debug
- policy or customer commitments may prevent raw traces from being sent to a hosted vendor

ATB keeps that workflow local-first. The trace stays on disk as a tamper-evident bundle, can be verified with the CLI, inspected through the local viewer, and exported into deterministic evidence archives when needed.

## ATB Workflow

1. Record events with the Go CLI or SDKs into a local `.atb` bundle.
2. Verify integrity with `atb verify`.
3. Review the bundle locally with `atb view` or `atb view --ui-experimental`.
4. Use masked payloads and explicit privacy reveals during review.
5. Export deterministic evidence archives if the incident needs a formal handoff.

## Example Review Flow

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb append agent.run --data='{"agent":"support-triage","ticket_id":"case-1042"}'
atb append tool.call --data='{"tool":"crm.lookup","status":"ok"}'
atb verify
atb view --ui-experimental
atb export --format soc2 --bundle run.atb/bundle.atb --output incident-evidence.zip
```

## Why This Fits Sensitive Environments

- **Local-first by default:** raw traces remain in local storage unless you explicitly export or encrypt them.
- **Verifiable integrity:** hash chaining plus canonical JSON make post-incident mutation detectable.
- **Controlled reveal flow:** configured PII fields stay masked by default, and reveals are authenticated, rate-limited, and audit-logged.
- **Portable review artefact:** the bundle can be handed to a customer, auditor, or internal review team without giving them a vendor account.
- **Vendor-independent review path:** the review artefact remains usable even if the original delivery environment or hosted platform is unavailable.

## Best-Fit Teams

- internal copilots touching customer or employee data
- consultancies handing over AI delivery work to a client
- enterprise teams that must explain agent behaviour during a security, privacy, or legal review

## Weak Fit

ATB is probably not your first tool if the primary need is:

- team-wide cloud dashboards
- managed prompt and eval workflows
- shared workspaces with collaborative commenting

Those are different categories of product. ATB is focused on the evidence layer for private review workflows.
