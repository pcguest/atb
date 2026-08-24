# Mortise integration handoff

Mortise is the optional commercial custody and organisational layer for ATB
evidence. Both products sit under the Tenon umbrella: ATB is the MIT-licensed,
local-first evidence core; Mortise operates across custody and organisational
concerns such as durable retention, receipts, transparency or witness evidence,
fleet views, and enterprise access controls.

ATB contains only the optional client boundary and conformance tests. Mortise
runtime code lives exclusively outside ATB. Compatibility is established
against explicit ATB contracts and the versions selected for each controlled
release, not an evergreen cross-repository assumption.

## Stable contracts

Mortise imports ATB's public Go packages and must not independently reimplement
or reinterpret these contracts:

| Contract | ATB authority | Mortise use |
| --- | --- | --- |
| Bundle format and hash chain | [`spec-v1.0.md`](./spec-v1.0.md), golden vectors | Verify before custody acceptance |
| Event contract | [`../schemas/event.v1.json`](../schemas/event.v1.json) | Interpret frozen event records |
| RFC 8785 canonicalisation | `pkg/jcs` | Canonical receipt and proof inputs |
| Custody verifier | `pkg/custody` | Verify, evaluate profiles, and read head hashes |
| Verifier report | [`api/verify-schema.md`](./api/verify-schema.md) | Embed `verify.report.v1` in receipts |
| Profile IDs and CAS | [`profiles.md`](./profiles.md), [`cas-guide.md`](./cas-guide.md) | Preserve ATB profile-scoped evidence-coverage semantics |
| Export envelope | `pkg/custody.BundleExport` | Transfer original bundle bytes and verifier metadata |

Go is canonical. Python and TypeScript bindings must match the frozen schema
and golden vectors through `make test-golden`.

## Push integration

- `atb intercept --mortise <endpoint>` lodges the completed bundle bytes when a session
  closes.
- `atb incident export --mortise-endpoint <endpoint>` lodges the authoritative
  bundle instead of writing a derived incident archive.
- `atb compliance pack ... --mortise-endpoint <endpoint>` lodges the
  authoritative bundle and includes the complete signed receipt at
  `mortise/receipt.json`, covered by the pack manifest and checksums.

Set `ATB_MORTISE_TOKEN` for Bearer authentication. The token is
environment-sourced so it does not appear in CLI arguments or process lists.
`--custos`, `--custos-endpoint`, and `ATB_CUSTOS_TOKEN` remain deprecated
compatibility aliases for one migration cycle.

Mortise must verify ATB integrity through `pkg/custody` before persistence. A
green integrity result proves the recorded bytes still form the expected chain;
it does not prove capture completeness, actor identity, model correctness, or
regulatory compliance.

## Boundary checks

- ATB has no Mortise runtime, storage, authentication, or daemon module.
- Mortise depends on ATB; ATB never imports the Mortise repository.
- The historical `custos.receipt.v1` string remains a frozen signed wire
  identifier and is not a product name.
- Production TLS, tenant policy, Object Lock, key management, witnesses,
  observability, and incident response are Mortise responsibilities.
