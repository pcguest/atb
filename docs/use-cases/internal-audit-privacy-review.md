# Internal audit and privacy review on a local bundle

ATB is useful when an internal reviewer needs to inspect what an AI
workflow did without defaulting to external trace storage.

This is common when security, privacy, legal, or internal audit teams
need a reviewable artefact that engineering can still verify locally.

## The problem

Internal review usually breaks down in one of three ways:

- the review team receives screenshots, copied logs, or one-off notes
- raw traces cannot be moved freely because they contain customer or employee data
- the evidence pack depends on continued access to the original control plane

That creates friction for both engineering and reviewers. Engineering
loses time reconstructing the story. Reviewers inherit evidence they
cannot verify independently.

## What ATB changes

ATB keeps the review path local-first:

1. Record the workflow into a local `.atb` bundle with the Go CLI or an SDK.
2. Verify integrity with `atb verify` before review starts.
3. Inspect the bundle locally with masked fields by default.
4. Use explicit privacy reveals when a reviewer needs a specific field.
5. Export deterministic `soc2` or `gdpr` artefacts when formal evidence is required.

## Example workflow

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received --data='{"request_id":"req-123","workflow":"claims-triage","subject_id":"usr_123"}'
atb append ai.policy.decision --data='{"policy_id":"pol-sensitive-case","policy_version":"2026-04","decision":"allow","decision_reason_codes":["manual_review_required"],"subject_id_hash":"sha256-usr-123","action_id":"act-123"}'
atb append ai.action.precommit --data='{"action_id":"act-123","action_type":"manual_review","target_resource_id":"privacy-queue","intended_effect":"review_sensitive_case"}'
atb verify
atb view
atb export --format soc2 --bundle run.atb/bundle.atb --output internal-audit-evidence.zip
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output privacy-review.zip
```

## Why this fits review work

- **Local verification first:** reviewers can start from an integrity check, not an assertion.
- **Masked by default:** configured PII fields stay masked until a reviewer explicitly reveals them.
- **Reveal accountability:** privacy reveals are authenticated, rate-limited, and appended to the same bundle audit trail.
- **Deterministic outputs:** recurring review packs are reproducible instead of hand-assembled.
- **Vendor-independent evidence:** the bundle remains usable without continued access to the original delivery environment.

## Best-fit teams

- internal AI teams handling customer or employee data
- privacy or legal teams reviewing a specific workflow or subject record
- internal audit teams that need repeatable evidence rather than screenshots

## Weak fit

ATB is probably not the first purchase if the main requirement is:

- organisation-wide workflow orchestration
- collaborative review queues and approvals inside a hosted workspace
- policy management, RBAC, or enterprise admin tooling

ATB is for the review artefact and local verification path itself.
