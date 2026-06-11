# Customer handoff workflow

Use this workflow when you need to hand over a trustworthy AI execution
record to a customer, procurement team, or internal owner without
leaving review trapped in your environment.

## Goal

Produce one handoff pack that:

- keeps the working bundle local until transfer is required
- encrypts the portable bundle before transfer
- exports a deterministic evidence pack alongside it
- lets the recipient decrypt, verify, and review locally

## Canonical flow

Sender:

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
atb trust-report --profile atb.profile.privileged_tool_action --format markdown
ATB_PASSWORD='shared-review-secret' atb encrypt run.atb/bundle.atb --output handoff/acme-review.atb.enc
atb export --format compliance --output handoff/acme-review-evidence.zip
```

Recipient:

```bash
ATB_PASSWORD='shared-review-secret' atb decrypt incoming/acme-review.atb.enc --output review/acme-review.atb
atb verify review/acme-review.atb --profile atb.profile.privileged_tool_action --format json
atb trust-report review/acme-review.atb --profile atb.profile.privileged_tool_action --format markdown
atb view review/acme-review.atb
```

## What each step proves

1. `atb append ...`
   Records the delivery timeline as a tamper-evident local bundle.

2. `atb snapshot customer_handoff_ready`
   Appends a named handoff checkpoint. Use the snapshot name to label workflow state; it is not a transport feature.

3. `atb verify --profile atb.profile.privileged_tool_action --format json`
   Confirms the sender is handing over an untampered bundle with the expected privileged-action evidence.

4. `atb trust-report --profile atb.profile.privileged_tool_action --format markdown`
   Produces a reviewer-friendly summary of bundle integrity, profile checks, and shipped trust evidence.

5. `atb encrypt ... --output handoff/acme-review.atb.enc`
   Produces a named encrypted artefact that is ready to transfer outside the original workspace.

6. `atb export --format compliance ...`
   Produces the strongest default handoff evidence pack: verified bundle, trust report, verification report, checksums, and reference docs.

7. `atb decrypt ... --output review/acme-review.atb`
   Lets the recipient restore the bundle into a chosen local review path.

8. `atb verify --profile atb.profile.privileged_tool_action --format json` and `atb trust-report --profile atb.profile.privileged_tool_action --format markdown`
   Let the recipient confirm the evidence independently, without trusting the sender's platform.

## Expected handoff pack

Send both of these:

- `handoff/acme-review.atb.enc`
- `handoff/acme-review-evidence.zip`

The password should travel out of band.

## Expected recipient outcome

The recipient should end with:

- a decrypted local bundle at a chosen review path
- `atb verify --profile atb.profile.privileged_tool_action --format json` reporting `pass: true`
- `atb trust-report` reporting `warn` or `pass`, depending on optional evidence such as policy signatures
- optional local UI review via `atb view`

## Review notes

- Use `--output` on both `encrypt` and `decrypt` so handoff artefacts have stable names.
- Prefer the compliance export for handoff because it is the clearest all-in-one evidence pack.
- ATB does not require a hosted control plane for this workflow; the artefact is the handoff.
