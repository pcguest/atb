# Case Study: Remediation Audit Trail

## Overview

ATB was used to capture and verify an end-to-end remediation audit trail for a
Next.js web application security review. The engagement produced a structured,
hash-chained bundle of remediation events that could be independently verified
against the obligation profile for a privileged-action workflow, without
requiring access to the original runtime environment.

## What was captured

The following ATB event types were recorded during the engagement:

- `atb.bundle.manifest` — Bundle manifest (seq 0, always first in a new bundle)
- `atb.bundle.signature` — Ed25519 bundle signature
- `atb.bundle.anchor` — RFC 3161 TSA timestamp anchor
- `ai.request.received` — AI request received at app boundary
- `ai.tool.exec` — Canonical tool execution lifecycle event for integrations
- `ai.policy.decision` — Policy engine decision (allow/deny)
- `ai.action.precommit` — Pre-commit record for a gated privileged action
- `ai.action.executed` — Privileged action executed through gate
- `ai.action.committed` — Privileged action committed to sink
- `ai.human.approval` — Human approval of an action or override
- `atb.bundle.pushed` — Bundle pushed to remote WORM target (atb push)

## Verification output

- Profile applied: `atb.profile.privileged_tool_action`
- Obligations satisfied: request, precommit, policy, executed, committed chain
- Critical failures: none (synthetic fixture)
- Completeness Assurance Score: Medium (`cas_score` ≈ 0.72 in representative fixture)
- Export format: compliance ZIP with `.verify.json` sidecar (`provability_layers` checklist)
- Provability gaps closed at L1–L2; L3–L4 optional (signatures, anchor, corroboration)

Run locally:

```bash
bash examples/quickstart/run.sh
./atb verify --profile atb.profile.privileged_tool_action --format json
```

## Limitations

ATB captures what the instrumented session records. In this engagement it does
not prove:
- That no remediation actions occurred outside the captured CLI session
- The identity of the operator beyond the key used to sign the bundle
- The correctness of the remediation itself — only that the declared sequence
  was captured and chained

These are structural blind spots acknowledged by the tool's design, not gaps
introduced by the engagement.

## Reproducibility

Any operator with the ATB CLI can reproduce an equivalent audit trail by
initialising a bundle, appending events of the same types in sequence, signing
with an Ed25519 key, and running `atb verify --profile [profile-name]`. The
resulting compliance export is a self-contained ZIP that can be verified
offline without any network dependency.