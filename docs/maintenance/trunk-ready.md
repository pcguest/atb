# Trunk-ready checklist

ATB is the open (MIT) **contract trunk**: a forward-facing, feature-frozen repo
that downstreams — including the Custos product — depend on. This page is the one
place to confirm the trunk is publishable and tag-ready, and to record the
contracts that must not break.

Trunk policy: **changes are limited to bugfixes, dependency hygiene, and
doc-truth.** New product features do not land here (they belong to the Custos
product). Any change that could alter a hash or schema must keep cross-language
golden parity and must not require a bundle-manifest migration.

## 1. Local gates (run before tagging)

```bash
cd ~/atb
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 make test-golden          # cross-language hash vectors
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 go test ./... -count=1     # full Go suite
make hygiene-quick                                                          # profile goldens + web lint/typecheck
/bin/bash scripts/check-support-matrix.sh                                   # support-matrix agreement
ATB_SKIP_TAG_CHECK=1 /bin/bash scripts/check-versions.sh                    # 9 version strings agree
GOCACHE=$(pwd)/.gocache/dev GOTOOLCHAIN=go1.26.3 go test ./test/custos/... -tags=integration -count=1   # custos conformance
(cd custos && GOCACHE=$(pwd)/../.gocache/dev GOTOOLCHAIN=go1.26.3 go test ./... -race -count=1)          # custos module (race)
(cd web && npm test -- --run && npm run lint && npm run typecheck)          # viewer
```

Docs-sync (also enforced in CI) — README and CHANGELOG must agree with the CLI
version:

```bash
CLI=$(go run ./cmd/atb version)   # or read from cmd/atb/main.go
grep -Fq "Current release: [\`v${CLI}\`](CHANGELOG.md)" README.md
grep -Eq "^## \[v${CLI}\]" CHANGELOG.md
```

> Note: `scripts/check-versions.sh` does **not** check README's "Current release:"
> line — only the docs-sync greps and CI do. Always run both.

## 2. Tag and release (when budget returns)

The version is already cut to **1.14.0** on `main` (all nine version strings,
CHANGELOG `## [v1.14.0]`, README current-release line). The full per-step
procedure lives in [`docs/release.md`](../release.md); the short form:

```bash
SKIP_DOCKER=1 scripts/release-check.sh        # release preflight
git push origin main                          # publish the staged trunk
git tag -s v1.14.0 -m "ATB v1.14.0"           # signed tag at the release commit
git push origin v1.14.0
gh release create v1.14.0 --title v1.14.0 --notes-from-tag   # or from the CHANGELOG section
```

`check-versions.sh` (without `ATB_SKIP_TAG_CHECK`) treats the latest git tag as
the source of truth, so tag only when the strings already agree.

## 3. Frozen contracts (must not break)

These are the surfaces downstreams import. Breaking one is a major-version event,
not a trunk change. See [`docs/custos-handoff.md`](../custos-handoff.md) →
*Stable contracts (do not fork)* for the authoritative table.

| Contract | Public location | Notes |
|---|---|---|
| Bundle format (NDJSON, RFC 8785 canonical, SHA-256 chain) | `docs/spec-v1.0.md` + golden vectors | The hash identity. Never reimplement. |
| Verifier output (`verify.report.v1`) | `pkg/custody` (`Evaluate`, `VerifierReport`), `docs/api/verify-schema.md` | The report schema downstreams parse. |
| Bundle load + head hash | `pkg/custody.LoadBundle`, `pkg/custody.HeadHash` | Content addressing. |
| Export wire | `pkg/custody.WireExporter` / `BundleExport` | Bundle export envelope. |
| Profile IDs (`atb.profile.*`) + CAS sub-scores | `docs/profiles.md`, `docs/cas-guide.md` | Classification + completeness signal. |
| Attestation message (custody receipts) | `bundleHash\nreceiptID\nsubmittedAt\nsignedAt` | Re-implemented identically by custody layers; changing it breaks receipt verification. |

**Importer rule:** the verifier lives in `internal/verify` and is **not**
importable across modules. Downstreams (Custos product) import only the public
`pkg/custody` surface — they call the Go verifier, they do not re-score CAS or
re-hash bundles. Cross-language hash parity is locked by the Go/Python/TS golden
tests; run `make test-golden` before any change that touches canonicalisation or
hashing.

## 4. What stays out of the trunk

Multi-tenant auth, billing, SSO, legal-hold UI, hosted Custos, a production
transparency-log service, and any compliance-certification claim. The in-repo
`custos/` module is a **reference** layer (ingest/receipt/registry/signing
demonstrations); the sellable product lives in a separate repository that depends
on this trunk as a frozen contract.
