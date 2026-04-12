# Incident review workflow

Use this workflow when an AI system misfires and you need a trustworthy
local review path without sending raw traces to a hosted platform by
default.

## Goal

Produce one local incident bundle that:

- records the workflow timeline
- marks the workflow outcome as failed
- still verifies as untampered evidence
- exports into a deterministic evidence pack for follow-up

## Canonical flow

```bash
go install github.com/pcguest/atb/cmd/atb@latest

atb bundle new

atb append ai.request.received --data='{"request_id":"req-1042","workflow":"support-triage","case_id":"case-1042","severity":"sev2"}'
atb append ai.action.precommit --data='{"action_id":"act-1042","action_type":"route_case","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-pii-redaction","policy_version":"2026-04","decision":"deny","decision_reason_codes":["customer_email_visible"],"subject_id_hash":"sha256-user-1042","action_id":"act-1042"}'
atb snapshot incident_review_failed

atb verify --format json
atb trust-report --profile atb.profile.privileged_tool_action --format markdown
atb view --ui-experimental
atb export --format compliance --output incident-review-evidence.zip
```

## What each step proves

1. `atb bundle new`
   Creates the local evidence store at `run.atb/bundle.atb`.

2. `atb append ...`
   Records the incident timeline as hash-chained events.

3. `atb snapshot incident_review_failed`
   Appends a named snapshot checkpoint after the incident events. Use the snapshot name to label workflow state; it does not change bundle integrity semantics.

4. `atb verify --format json`
   Confirms the hash chain is intact.

5. `atb trust-report --profile atb.profile.privileged_tool_action --format markdown`
   Produces a review-friendly summary of bundle integrity, profile checks, and shipped trust evidence.

6. `atb view --ui-experimental`
   Opens the local review UI with masked fields by default.

7. `atb export --format compliance --output incident-review-evidence.zip`
   Produces the strongest default incident pack: verified bundle, trust report, verification report, checksums, and reference docs.

## Expected outcome

The important outcome is not that every status says `pass`.

The expected pattern for incident review is:

- the bundle contains a snapshot named `incident_review_failed`
- `atb verify` is `valid`
- `atb trust-report` is `warn` or `pass`, depending on whether optional evidence such as policy signatures is present

That combination means the workflow needs review, but the evidence trail
is still trustworthy.

## Resulting artefacts

- local bundle: `run.atb/bundle.atb`
- exported pack: `incident-review-evidence.zip`

Key files inside the exported pack:

- `evidence/reports/trust-report.json`
- `evidence/reports/verify.json`
- `evidence/docs/spec-v1.0.md`

## Review notes

- Keep raw traces local unless a formal follow-up requires export or encrypted handoff.
- Use privacy reveals sparingly; reveal actions are authenticated, rate-limited, and logged back into the bundle.
- Prefer the compliance export for incident follow-up because it carries the clearest all-in-one evidence package.
