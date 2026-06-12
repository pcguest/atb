# Public repository surface

ATB is a public, MIT-licensed open-source repository. The source tree,
specifications, SDKs, tests, fixtures, examples, local viewer, release tooling,
and maintainer documentation are world-readable and should be written on that
basis.

Custos is a separate proprietary companion product published for evaluation and
audit at [github.com/pcguest/custos-product](https://github.com/pcguest/custos-product).
ATB owns local capture, the `.atb` format, integrity verification, profiles,
CAS, and offline review. Custos owns durable custody, signed receipts,
transparency-log evidence, and auditor access. Neither product certifies
compliance or proves capture completeness.

## Compatibility-sensitive public contracts

The following surfaces require extra review because external users and Custos
depend on them:

- `docs/spec-v1.0.md`, `schemas/event.v1.json`, and canonical hash golden vectors
- `pkg/custody`, `pkg/jcs`, public SDK APIs, and `verify.report.v1`
- documented CLI help, exit codes, JSON output, profile IDs, and CAS semantics
- bundle signatures, anchors, encryption metadata, and custody export envelopes

Canonicalisation or on-disk compatibility changes require the versioning review
in `VERSIONING.md` and cross-language parity through `make test-golden`.

## Public product boundary

| Surface | Status | Boundary |
| --- | --- | --- |
| Hash chain, signatures, anchors, and `atb verify` | Shipped | Proves integrity of recorded evidence, not factual correctness |
| Six obligation profiles and CAS | Shipped | Measures expected evidence presence, not universal capture |
| CLI, Go/Python/TypeScript SDKs, and golden vectors | Shipped | Local-first and independently verifiable |
| `atb intercept`, SDK wrappers, imports, and OTel JSON | Shipped | Each sees only traffic or calls routed through that integration |
| `atb incident`, evidence packs, and local viewer | Shipped | Review and export surfaces; not a SIEM or hosted collaboration product |
| In-repo `custos/` Go module | Reference scaffold | Kept for contract tests and compatibility; not the Custos product |
| Hosted custody, SSO, billing, legal hold, and managed witnesses | Outside ATB | Belongs in Custos Ring 4 or another external product |

## atb view

`atb view` is a local, read-only review server for verified bundles. It binds to
loopback by default and protects API routes with a generated session token.
With `--sessions`, the `/sessions` surface exposes the authenticated session
index, actor grouping, schema status, anomaly summaries, and role-aware panels.
It does not provide hosted multi-tenant review.

## atb intercept

By default the capture proxy records a SHA-256 digest and byte length for
request and response bodies, not raw prompts or completions. Credential and
session-secret headers are stripped. `--capture-bodies` is an explicit privacy
tradeoff.

`--custos <url>` pushes a closed immutable bundle snapshot to a configured
Custos endpoint. `ATB_CUSTOS_TOKEN`, when set, supplies the Bearer token from
the environment. With neither option configured, interception remains local and
does not perform network custody operations.

## atb incident

`atb incident list`, `report`, and `export` operate on a bundle loaded through
the integrity gate. Reports scope a session for review, but the complete bundle
remains the authoritative hash-chained evidence object. An unsigned bundle is
reported as unsigned rather than treated as signed provenance.

## Custos and the in-repo scaffold

The supported companion product and evaluator path live in the
[Custos repository](https://github.com/pcguest/custos-product). The in-repo
`custos/` module remains a reference implementation and compatibility harness;
new custody product work does not land in ATB.

## Research and planning material

Pages under `docs/research/` are design notes, not shipped behavior unless a
page explicitly states otherwise. Pages under `docs/maintenance/` are public
maintainer records. Completed trackers remain as short historical redirects so
old links continue to resolve without presenting stale plans as current work.

## Legacy public-export tooling

`scripts/export-public-demo.sh` and its workflow remain as legacy packaging
tools for producing a narrow demonstration tree. They are not a security
boundary and do not define which ATB source is public: this repository itself
is the public source of truth.
