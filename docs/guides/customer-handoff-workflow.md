# Customer Handoff Workflow

Use this workflow when you need to hand over a trustworthy AI execution record to a customer, procurement team, or internal owner without leaving review trapped in your environment.

## Goal

Produce one handoff pack that:

- keeps the working bundle local until transfer is required
- encrypts the portable bundle before transfer
- exports a deterministic evidence pack alongside it
- lets the recipient decrypt, verify, and review locally

## Canonical Flow

Sender:

```bash
go install github.com/pcguest/atb/cmd/atb@latest

atb init

atb append agent.run --data='{"workflow":"claims-triage","customer":"acme","ticket_id":"handoff-204"}'
atb append decision --data='{"action":"route_to_manual_review","reason":"confidence_below_threshold","customer":"acme"}'
atb snapshot customer_handoff_ready

atb verify --format json
atb trust-report --format markdown
ATB_PASSWORD='shared-review-secret' atb encrypt run.atb/bundle.atb --output handoff/acme-review.atb.enc
atb export --format compliance --output handoff/acme-review-evidence.zip
```

Recipient:

```bash
ATB_PASSWORD='shared-review-secret' atb decrypt incoming/acme-review.atb.enc --output review/acme-review.atb
atb verify review/acme-review.atb --format json
atb trust-report review/acme-review.atb --format markdown
atb view review/acme-review.atb --ui-experimental
```

## What Each Step Proves

1. `atb append ...`
   Records the delivery timeline as a tamper-evident local bundle.

2. `atb snapshot customer_handoff_ready`
   Appends a named handoff checkpoint. Use the snapshot name to label workflow state; it is not a transport feature.

3. `atb verify --format json`
   Confirms the sender is handing over an untampered bundle.

4. `atb trust-report --format markdown`
   Produces a reviewer-friendly summary of bundle integrity and shipped trust evidence.

5. `atb encrypt ... --output handoff/acme-review.atb.enc`
   Produces a named encrypted artefact that is ready to transfer outside the original workspace.

6. `atb export --format compliance ...`
   Produces the strongest default handoff evidence pack: verified bundle, trust report, verification report, checksums, and reference docs.

7. `atb decrypt ... --output review/acme-review.atb`
   Lets the recipient restore the bundle into a chosen local review path.

8. `atb verify` and `atb trust-report`
   Let the recipient confirm the evidence independently, without trusting the sender's platform.

## Expected Handoff Pack

Send both of these:

- `handoff/acme-review.atb.enc`
- `handoff/acme-review-evidence.zip`

The password should travel out of band.

## Expected Recipient Outcome

The recipient should end with:

- a decrypted local bundle at a chosen review path
- `atb verify` reporting `valid`
- `atb trust-report` reporting `pass`
- optional local UI review via `atb view`

## Review Notes

- Use `--output` on both `encrypt` and `decrypt` so handoff artefacts have stable names.
- Prefer the compliance export for handoff because it is the clearest all-in-one evidence pack.
- ATB does not require a hosted control plane for this workflow; the artefact is the handoff.
