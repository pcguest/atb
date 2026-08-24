# Provability ladder

ATB separates **integrity** (whether the presented records form the declared
chain) from **coverage** (which expected profile evidence is present in those
records). The provability ladder maps each claim to a layer. Configure higher
layers to shrink conditional blind spots.

## Layers

| Layer | Name | Mechanism | Provable when |
|-------|------|-----------|---------------|
| L1 | Integrity | SHA-256 hash chain, RFC 8785 canonical JSON, manifest record, `LoadVerified`, viewer verification gate | `atb verify` reports `integrity.chain_valid: true` |
| L2 | Workflow shape | Six obligation profiles, relations, `required_when`, DSL v1 | Selected profile `pass: true` |
| L3 | Source binding | Policy decision signatures, bundle signatures, SC sub-score | `policy_signature` verifies; bundle signature verifies |
| L4 | External witness | RFC 3161 anchor (AC), WORM push, `atb.corroboration.external` (XC) | Anchor verified; corroboration events present and valid |
| L5 | Capture boundary | `atb capture run`, SDK middleware, `import chatlog`, instrumentation checklist | Capture env vars set; required events emitted at call sites |

## Gap to layer mapping

| Common concern | Layer | Mitigation | Residual (always document) |
|----------------|-------|------------|----------------------------|
| Post-capture tampering | L1 | Hash chain + verify | Pre-load wholesale file replacement (host trust) |
| Missing workflow events | L2 | Profile + instrumentation | Events outside declared profile boundary |
| Causal ordering fraud | L2 | Temporal profile rules, TC sub-score | Clock skew without external time source |
| Unsigned policy decisions | L3 | `policy_signature` on `ai.policy.decision` | Caller-asserted identity without external IdP |
| Bundle replaced before handoff | L4 | WORM export, anchor, `.verify.json` sidecar | Operator bypasses export controls |
| Actions outside session | L4/L5 | Corroboration, ACP gate | Behaviour never instrumented |
| Import-only / chatlog gaps | L5 | Live SDK capture vs retrospective import | Chatlog may omit tool or policy events |

## Irreducible limits

ATB never claims to prove:

- Model correctness or decision quality
- Human comprehension of context at approval time
- Legal or compliance certification (CAS is not an audit opinion)
- Behaviour that never crossed the instrumentation boundary

## Verifier output

`atb verify --format json` includes `provability_gaps`: structured items with
`gap`, `layer`, `mitigation`, and `closed_when`. Use these to close layers in
order rather than treating all limits as permanent.

See also [security.md](./security.md) and
[cas-guide.md](./cas-guide.md).
