# Quickstart

This example demonstrates a minimal `privileged_tool_action` workflow that produces a verifiable ATB bundle with a CAS score, suitable for understanding the core ATB loop before integrating into a real application.

## Prerequisites

- Go installed
- `atb` installed with `go install github.com/pcguest/atb/cmd/atb@latest`
- `bash`

## How to run

```bash
cd examples/quickstart
bash run.sh
```

## What a passing run looks like

```bash
✓ Initialised ATB bundle at run.atb/bundle.atb
✓ Appended event #1 [ai.request.received]
✓ Appended event #2 [ai.action.precommit]
✓ Appended event #3 [ai.policy.decision]
✓ Appended event #4 [ai.action.executed]
✓ Appended event #5 [ai.human.approval]
✓ Appended event #6 [ai.action.committed]
{
  "bundle_path": "run.atb/bundle.atb",
  "integrity": {
    "chain_valid": true,
    "canonicalization": "rfc8785",
    "hash_algo": "sha256",
    "first_seq": 0,
    "last_seq": 6
  },
  "anchoring": {
    "anchor_required": false,
    "anchor_present": false,
    "status": "failed",
    "summary": "anchor: failed (anchor record not present)",
    "message_imprint_verified": false,
    "signature_verified": false,
    "tsa_verified": false,
    "cert_chain_verified": false,
    "reason": "anchor record not present"
  },
  "profiles": [
    {
      "profile_id": "atb.profile.privileged_tool_action",
      "version": 1,
      "workflow_class": "privileged_tool_action",
      "pass": true,
      "critical_failures": [],
      "required_warnings": [
        "ai.policy.decision: policy_signature absent"
      ],
      "informational_notes": [
        "If an operator bypasses the gate, this profile cannot detect that bypass without external reconciliation.",
        "Tool provider internal processing is not attested unless tool receipts are cryptographically verifiable."
      ]
    }
  ],
  "cas": {
    "overall": 0.76,
    "grade": "Medium",
    "sub_scores": {
      "AC": 0,
      "EC": 1,
      "FC": 1,
      "GC": 1,
      "RC": 1,
      "SC": 0.6,
      "TC": 1,
      "XC": 0
    },
    "weight_vector": {
      "AC": 0.1,
      "EC": 0.2,
      "FC": 0.15,
      "GC": 0.1,
      "RC": 0.2,
      "SC": 0.1,
      "TC": 0.05,
      "XC": 0.1
    }
  },
  "exclusions": [
    "If an operator bypasses the gate, this profile cannot detect that bypass without external reconciliation.",
    "Tool provider internal processing is not attested unless tool receipts are cryptographically verifiable."
  ],
  "residual_risk": {
    "level": "Medium",
    "drivers": [
      "AC",
      "SC",
      "XC"
    ],
    "recommended_next_evidence": [
      "Add RFC 3161 TSA anchoring for exports and privileged actions.",
      "Add source signatures from policy engine or action gate.",
      "Configure external corroboration: gateway log, DB receipt, or queue dequeue."
    ]
  }
}
Bundle:   run.atb/bundle.atb
Status:   WARN
Gate:     PASS
Summary:  total=13 pass=12 warn=1 fail=0

Categories:
  cryptographic_integrity: PASS
  operational_safety: PASS
  test_coverage: PASS
  documentation: PASS
  obligation_profile: WARN

Check Details:
  [WARN] obligation_profile Profile Required Warning: ai.policy.decision: policy_signature absent

CAS:
  Profile:   atb.profile.privileged_tool_action
  Class:     privileged_tool_action
  Grade:     Medium  (0.76)
  Anchor:    absent  (XC=0.00 AC=0.00)
  Sub-scores:
    EC  1.00   FC  1.00   RC  1.00   TC  1.00
    SC  0.60   XC  0.00   AC  0.00   GC  1.00
=== CAS grade: B ===
Quickstart complete. Bundle is at run.atb/bundle.atb
```

## What to do next

- [Incident Review Workflow](../../docs/guides/incident-review-workflow.md)
- [Integrations](../../docs/integrations/README.md)
