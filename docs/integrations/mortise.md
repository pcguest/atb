# Mortise custody integration

Mortise is the optional commercial custody and assurance layer for organisations
that need ATB evidence preserved, governed, and independently receipted beyond
the originating host. ATB remains fully useful for local capture, verification,
incident analysis, profiles, packs, and viewing when Mortise is absent.

Tenon is the umbrella product identity. It is not an ATB runtime component.
The boundary is:

```text
local evidence creation and verification = ATB
organisational custody and assurance      = Mortise
```

## Stable ATB contracts

Mortise integrations consume, rather than reinterpret, these contracts:

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
report archive.

The legacy `--custos`, `--custos-endpoint`, and `ATB_CUSTOS_TOKEN` names remain
compatibility aliases for the documented migration window. The historical
`custos.receipt.v1` string is a frozen wire identifier, not a current product
name.

Mortise must verify the bundle before persistence. Production storage, WORM
retention, receipts, tenant policy, RBAC, SSO, witnesses, audit access, and
operational controls belong to Mortise or another custody provider—not ATB.
