# Mortise integration handoff

Mortise is the separate proprietary custodian-of-record framework built on
ATB: [github.com/pcguest/mortise](https://github.com/pcguest/mortise). Both sit
under the Tenon umbrella: ATB is the open MIT evidence core; Mortise provides
commercial custody, WORM storage, signed receipts, timestamps, transparency,
and auditor operations.

The current ATB source/tag baseline is `v1.15.2`. Mortise `v0.5.0` currently
pins an older ATB module dependency and must be checked against the
[Tenon compatibility matrix](https://github.com/pcguest/tenon/blob/main/docs/compatibility.md)
before presenting any ATB/Mortise pair as the supported current combination.
ATB contains only the client boundary and conformance tests. Mortise runtime
code lives exclusively in its own repository.

Use Mortise's
[end-to-end guide](https://github.com/pcguest/mortise/blob/main/docs/e2e-atb-mortise.md),
[API quickstart](https://github.com/pcguest/mortise/blob/main/docs/api-quickstart.md),
and
[capability boundary](https://github.com/pcguest/mortise/blob/main/docs/capability-boundary.md)
for the supported evaluator and operator paths.

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
| Profile IDs and CAS | [`profiles.md`](./profiles.md), [`cas-guide.md`](./cas-guide.md) | Preserve ATB completeness semantics |
| Export envelope | `pkg/custody.BundleExport` | Transfer original bundle bytes and verifier metadata |

Go is canonical. Python and TypeScript bindings must match the frozen schema
and golden vectors through `make test-golden`.

## Push integration

- `atb intercept --mortise <endpoint>` lodges immutable bytes when a session
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
