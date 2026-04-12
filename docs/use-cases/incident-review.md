# Incident review for private AI workflows

ATB is built for the moment after something goes wrong.

When an AI workflow misfires, most teams can retrieve logs. Fewer teams
can show that the recorded event sequence is tamper-evident, and fewer
still can explain evidence completeness against an explicit workflow
profile. That is the gap ATB closes.

## The problem

Private AI workflows create a difficult trade-off during incident
review:

- security and compliance teams want a trustworthy record
- engineering wants enough detail to debug
- policy or customer commitments may prevent raw traces from being sent to a hosted vendor

ATB keeps that workflow local-first. The trace stays on disk as a
tamper-evident bundle, can be verified with the CLI, inspected through
the local viewer, and exported into deterministic evidence archives when
needed.

## ATB workflow

1. Record events with the Go CLI or SDKs into a local `.atb` bundle.
2. Mark the review state with a failing snapshot when the workflow outcome is bad.
3. Verify integrity with `atb verify`.
4. Generate a trust report so engineering and reviewers can separate workflow failure from evidence validity.
5. Review the bundle locally with `atb view` or `atb view --ui-experimental`.
6. Use masked payloads and explicit privacy reveals during review.
7. Export the deterministic evidence pack if the incident needs formal follow-up.

## Example review flow

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042","workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append ai.action.precommit --data='{"action_id":"act-1042","action_type":"route_case","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-pii-redaction","policy_version":"2026-04","decision":"deny","decision_reason_codes":["customer_email_visible"],"subject_id_hash":"sha256-user-1042","action_id":"act-1042"}'
atb snapshot incident_review_failed
atb verify
atb trust-report --profile atb.profile.privileged_tool_action --format markdown
atb view --ui-experimental
atb export --format compliance --output incident-review-evidence.zip
```

Important distinction:

- `snapshot incident_review_failed` records that the workflow needs attention by name.
- `atb verify` should still pass if the bundle itself is intact.
- `atb trust-report` may report `warn` rather than `pass` when optional evidence such as policy signatures is absent.

That is the core ATB value in incident review: the workflow can fail
without weakening the evidence.

## Why this fits sensitive environments

- **Local-first by default:** raw traces remain in local storage unless you explicitly export or encrypt them.
- **Verifiable integrity:** hash chaining plus canonical JSON make post-incident mutation detectable.
- **Controlled reveal flow:** configured PII fields stay masked by default, and reveals are authenticated, rate-limited, and audit-logged.
- **Portable review artefact:** the bundle can be handed to a customer, auditor, or internal review team without giving them a vendor account.
- **Vendor-independent review path:** the review artefact remains usable even if the original delivery environment or hosted platform is unavailable.
- **One default evidence pack:** `atb export --format compliance` packages the bundle, trust report, verification report, checksums, and reference docs into a single local archive.

## Best-fit teams

- internal copilots touching customer or employee data
- consultancies handing over AI delivery work to a client
- enterprise teams that must explain agent behaviour during a security, privacy, or legal review

## Weak fit

ATB is probably not your first tool if the primary need is:

- team-wide cloud dashboards
- managed prompt and eval workflows
- shared workspaces with collaborative commenting

Those are different product categories. ATB is focused on the evidence
layer for private review workflows.
