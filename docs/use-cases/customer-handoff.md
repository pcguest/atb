# Customer handoff without platform lock-in

ATB is useful when a team needs to hand over a trustworthy AI execution
record to someone outside the delivery team.

This is common for consultancies, internal platform teams, and
enterprise builders shipping AI workflows into regulated environments.

## The problem

Customer handoff usually breaks down in one of three ways:

- the delivery team sends screenshots and reconstructed notes
- the customer is asked to trust a hosted observability platform they do not control
- the original trace is spread across logs, dashboards, and ad hoc exports

That makes review harder than it should be. It also creates unnecessary
dependency on the original delivery environment.

## What ATB changes

ATB produces a portable bundle that can be:

- stored locally
- verified independently with `atb verify`
- inspected through the local viewer
- exported into deterministic evidence archives when formal review is needed

The handoff is the bundle itself, not continued reliance on the
original platform.

For the operator runbook, use the [Customer handoff
workflow](../guides/customer-handoff-workflow.md).

## Example workflow

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received --data='{"request_id":"req-204","actor_id_hash":"sha256-customer-acme","purpose_tag":"claims-triage"}'
atb append ai.action.precommit --data='{"action_id":"act-204","action_type":"route_to_manual_review","action_parameters_digest":"sha256-manual-review-route","target_resource_id":"claims-queue","intended_effect":"manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-claims-review","policy_version":"2026-04","decision":"allow","decision_reason_codes":["confidence_below_threshold"],"subject_id_hash":"sha256-customer-acme","action_id":"act-204"}'
atb append ai.action.executed --data='{"action_id":"act-204","execution_outcome":"success","tool_receipt_digest":"sha256-tool-receipt-204"}'
atb append ai.action.committed --data='{"action_id":"act-204","commit_outcome":"committed","sink_receipt_digest":"sha256-sink-receipt-204"}'
atb snapshot customer_handoff_ready
atb verify --profile atb.profile.privileged_tool_action --format json
ATB_PASSWORD='shared-review-secret' atb encrypt run.atb/bundle.atb --output handoff/acme-review.atb.enc
atb export --format compliance --output handoff/acme-review-evidence.zip

ATB_PASSWORD='shared-review-secret' atb decrypt incoming/acme-review.atb.enc --output review/acme-review.atb
atb verify review/acme-review.atb --profile atb.profile.privileged_tool_action --format json
```

## Why teams use this

- **Portable review artefact:** the bundle can move with the delivery, not stay trapped in an environment.
- **Independent verification:** the recipient can check integrity without trusting the original operator.
- **Clear privacy posture:** traces stay local by default and can be encrypted before transfer.
- **Deterministic outputs:** evidence exports are reproducible instead of hand-built slide decks or one-off notes.

## Best-fit teams

- consultancies shipping AI systems into customer environments
- internal platform teams handing an AI workflow to security, legal, or audit
- enterprises that need a reviewable artefact for procurement or post-incident follow-up

## Weak fit

ATB is not the best answer if the handoff requirement is:

- shared online commenting and multi-user queue management
- managed collaboration inside a hosted control plane
- a broad vendor platform with prompt and eval tooling

ATB is for the handoff artefact itself.
