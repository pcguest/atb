# Customer Handoff Without Platform Lock-In

ATB is useful when a team needs to hand over a trustworthy AI execution record to someone outside the delivery team.

This is common for consultancies, internal platform teams, and enterprise builders shipping AI workflows into regulated environments.

## The Problem

Customer handoff usually breaks down in one of three ways:

- the delivery team sends screenshots and reconstructed notes
- the customer is asked to trust a hosted observability platform they do not control
- the original trace is spread across logs, dashboards, and ad hoc exports

That makes review harder than it should be. It also creates unnecessary dependency on the original delivery environment.

## What ATB Changes

ATB produces a portable bundle that can be:

- stored locally
- verified independently with `atb verify`
- inspected through the local viewer
- exported into deterministic evidence archives when formal review is needed

The handoff is the bundle itself, not continued reliance on the original platform.

## Example Workflow

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb init
atb append agent.run --data='{"workflow":"claims-triage","customer":"acme"}'
atb append decision --data='{"action":"route_to_manual_review","reason":"confidence_below_threshold"}'
atb verify
ATB_PASSWORD='shared-review-secret' atb encrypt run.atb/bundle.atb
atb export --format soc2 --bundle run.atb/bundle.atb --output acme-review-evidence.zip
```

## Why Teams Use This

- **Portable review artefact:** the bundle can move with the delivery, not stay trapped in an environment.
- **Independent verification:** the recipient can check integrity without trusting the original operator.
- **Clear privacy posture:** traces stay local by default and can be encrypted before transfer.
- **Deterministic outputs:** evidence exports are reproducible instead of hand-built slide decks or one-off notes.

## Best-Fit Teams

- consultancies shipping AI systems into customer environments
- internal platform teams handing an AI workflow to security, legal, or audit
- enterprises that need a reviewable artefact for procurement or post-incident follow-up

## Weak Fit

ATB is not the best answer if the handoff requirement is:

- shared online commenting and multi-user queue management
- managed collaboration inside a hosted control plane
- a broad vendor platform with prompt and eval tooling

ATB is for the handoff artefact itself.
