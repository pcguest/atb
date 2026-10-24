# Optional Mortise custody handoff

This page documents ATB’s optional client and compatibility contracts for
handing off evidence to an external custody endpoint. It does not describe or
verify the receiving service’s current implementation or operation. ATB capture,
verification, incident analysis, profiles, packs, and local viewing work without
that endpoint.

## Stable ATB contracts

A compatible receiver should consume, rather than reinterpret, these ATB contracts:

| Contract | Authority |
| --- | --- |
| Bundle and chain | [bundle specification](../specification/bundle-v1.md) and golden vectors |
| Event vocabulary | [`schemas/event.v1.json`](../../schemas/event.v1.json) |
| Canonicalisation | `pkg/jcs` |
| Custody verifier/export | `pkg/custody` |
| Verification report | [verify.report.v1](../specification/verify-report.md) |
| Profiles and CAS | [profiles](../evidence/profiles.md) and [CAS](../evidence/cas.md) |

## Optional handoff

Set the bearer token in `ATB_MORTISE_TOKEN`; do not put it in command-line
arguments or tracked scripts.

```bash
export ATB_MORTISE_TOKEN="<mortise-api-key>"

atb incident export \
  --bundle evidence.atb \
  --session <session-id> \
  --mortise-endpoint https://mortise.example
```

`atb intercept --mortise <endpoint>` can lodge a completed session bundle, and
`atb compliance pack ... --mortise-endpoint <endpoint>` can include the returned
receipt. The authoritative object handed to custody is the bundle, not a derived
report archive. A compatible receiver should verify bundle integrity with the
ATB custody verifier before persistence; this contract does not attest that an
external receiver follows that requirement.

The legacy `--custos`, `--custos-endpoint`, and `ATB_CUSTOS_TOKEN` names remain
compatibility aliases for the documented migration window. The historical
`custos.receipt.v1` string is a frozen wire identifier, not a current product
name.

The receiving custody service is responsible for its own persistence,
retention, receipts, access policy, and operational controls. ATB does not
verify that service’s current implementation or storage enforcement.
