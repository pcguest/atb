# Public surface and private core

This repository is the **private source-of-truth** implementation for ATB. A
separate public demo or product repository may be generated from an allowlisted
subset of this tree. The export pipeline lives under `scripts/export-public-demo.sh`.

## Private core (do not export)

These areas contain the full obligation evaluator, CAS scoring, capture mapping
logic, KMS integrations, and operational harness material:

- `internal/` — bundle engine, verify, profiles, trust, capture, corroboration, push, sign
- `cmd/atb/` — full CLI implementation
- `AGENTS.md` — maintainer and agent harness
- `.github/workflows/release.yml`, `scripts/quality-evidence.sh`, and other release internals
- Adversarial and fuzz corpora under `internal/bundle/`

## Safe to expose publicly

- `schemas/event.v1.json` and trimmed specification docs
- `sdk/python/` and `sdk/typescript/` (integration surface)
- `examples/` and generated sample bundles
- `test/golden/` cross-language parity fixtures
- `web/` viewer and landing (with copy aligned to public positioning)
- Conceptual docs: `docs/why-atb.md`, `docs/security.md`, `docs/provability-ladder.md`, quickstart guides

## Public demo repository notice

When publishing a public tree, include this notice in the public README:

> This repository is the public product and demo surface for ATB. It includes
> specifications, SDKs, sample bundles, and a local viewer sufficient to evaluate
> tamper-evident audit trails for AI workflows. It is not the complete private
> implementation repository. Advanced obligation evaluation, enterprise
> integrations, and operational tooling may differ from what is published here.

## Before each export

1. Run `scripts/export-public-demo.sh --dry-run`
2. Review `EXPORT_REVIEW.md` for unexpected paths
3. Confirm secret scan passed (no PEM blocks, API keys, or private bundles)
4. Set `APPROVED_BY=<reviewer>` before any push to the public remote

## Public demo repository shape

The generated public tree (for example `atb-demo`) is intentionally narrow:

| Included | Purpose |
|----------|---------|
| `schemas/` | Event schema authority |
| Trimmed `docs/` | Specification, provability ladder, quickstart (no maintainer harness) |
| `sdk/python/`, `sdk/typescript/` | Integration surface with golden parity CI |
| `examples/` + sample bundles | Runnable demos including valid/tampered pairs |
| `test/golden/` | Cross-language hash regression contract |
| `web/` | Local viewer (mock API default when verifier binary absent) |
| Public README | Repository notice (see above) |

| Excluded | Reason |
|----------|--------|
| `internal/`, `cmd/atb/` | Full obligation evaluator and CLI source |
| Release/adversarial CI | Operational internals |
| Private keys and root `*.atb` | Secret and fixture hygiene |

The verifier ships as a **release binary artefact** in the public repo; obligation
evaluation logic is not re-exported as Go source.
