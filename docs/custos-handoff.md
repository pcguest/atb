# Custos integration handoff

Custos is the separate proprietary companion to ATB:
[github.com/pcguest/custos-product](https://github.com/pcguest/custos-product).
The current integration baseline is ATB `v1.14.2` and Custos `v0.5.0`.

Use the Custos
[end-to-end guide](https://github.com/pcguest/custos-product/blob/main/docs/e2e-atb-custos.md)
for the evaluator path and its
[capability boundary](https://github.com/pcguest/custos-product/blob/main/docs/capability-boundary.md)
for the canonical four rings and never-claims.

The `custos/` module in this repository is a reference scaffold and
compatibility harness. It is not the supported Custos product. New custody
features, hosted operations, auditor-product work, and Ring 4 scope belong in
the separate repository.

## Stable contracts

Custos imports ATB's public Go packages and must not independently reimplement
or reinterpret these contracts:

| Contract | ATB authority | Custos use |
| --- | --- | --- |
| Bundle format and hash chain | [`spec-v1.0.md`](./spec-v1.0.md), golden vectors | Verify before custody acceptance |
| RFC 8785 canonicalisation | `pkg/jcs` | Canonical receipt and proof inputs |
| Custody verifier | `pkg/custody` | Load, verify, evaluate profiles, and read head hashes |
| Verifier report | [`api/verify-schema.md`](./api/verify-schema.md) | Embed `verify.report.v1` in receipts |
| Profile IDs and CAS | [`profiles.md`](./profiles.md), [`cas-guide.md`](./cas-guide.md) | Preserve ATB's completeness interpretation |
| Export envelope | `pkg/custody.BundleExport` | Transfer original bundle bytes and verifier metadata |

Go is canonical. Python and TypeScript must continue to match Go byte-for-byte
through `make test-golden`.

## Push integration

`atb intercept --custos <endpoint>` pushes the immutable bytes of a closed
bundle. When `ATB_CUSTOS_TOKEN` is set, ATB sends it as an
`Authorization: Bearer` header. The token is environment-sourced so it does not
appear in CLI arguments or shell history.

Custos must verify bundle integrity through `pkg/custody` before persistence.
A green integrity result proves that the recorded bytes still form the expected
chain. It does not prove capture completeness, actor identity, model
correctness, or compliance.

## Production hardening checklist

The in-repo `custosd` scaffold is loopback-first development infrastructure. If
it is used outside a local compatibility test, the operator must provide:

- TLS termination at an operator-controlled reverse proxy
- authentication rotation and revocation beyond a single static bearer token
- tenant and user identity controls
- rate limiting and abuse controls
- durable WORM storage and tested recovery procedures
- explicit interpretation of integrity-only re-verification versus profile
  completeness

For supported production deployment, use the
[Custos deployment guide](https://github.com/pcguest/custos-product/blob/main/docs/deploy-production.md).
