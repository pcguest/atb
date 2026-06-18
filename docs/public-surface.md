# Public repository surface

ATB is a public, MIT-licensed open-source repository. The source tree,
specifications, SDKs, tests, fixtures, examples, local viewer, release tooling,
and maintainer documentation are world-readable and should be written on that
basis.

Mortise is a separate proprietary companion product published for evaluation and
audit at [github.com/pcguest/custos-product](https://github.com/pcguest/custos-product).
ATB owns local capture, the `.atb` format, integrity verification, profiles,
CAS, and offline review. Mortise owns durable custody, signed receipts,
transparency-log evidence, and auditor access. Tenon is the umbrella name for
both: ATB the open core, Mortise the commercial framework. Neither product
certifies compliance or proves capture completeness.

## Compatibility-sensitive public contracts

The following surfaces require extra review because external users and Mortise
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
| Reviewer identity evidence | Shipped, optional | Hash-chains caller-provided IdP/assertion digests; ATB is not an IdP and does not validate the assertion |
| Retention operations bundle | Shipped | Records policy changes, local archive outcomes, and accepted Object Lock requests; does not prove continuing remote enforcement |
| `atb compliance pack` | Shipped | Deterministic offline bundle/profile review package; not a conformity assessment |
| In-repo `custos/` Go module | Reference scaffold | Kept for contract tests and compatibility; not the Mortise product |
| Managed storage (filesystem, S3-compatible) | In-repo reference | Mortise provides managed storage for receipts and bundles, with retention policies |
| Hosted custody, SSO, billing, legal hold, and managed witnesses | Outside ATB | Belongs in Mortise Ring 4 or another external product |

## Role-Based Access Control (RBAC)

ATB and Mortise now support optional role-based access control for their HTTP APIs. This allows operators to define granular permissions for different users or services interacting with the systems.

### Roles

The following roles are defined:

*   **`viewer`**: Read-only access to view data and reports.
*   **`auditor`**: Read-only access to view data and generate reports (same as viewer for now, but can be extended).
*   **`operator`**: Read-write access to manage data and configurations (e.g., ingest events/bundles, trigger privacy reveals, re-run profile verification).
*   **`admin`**: Full administrative access.

### Authentication Mechanisms

Both `custosd` and `atb view` support:

*   **Shared Secret Token**: A simple bearer token (e.g., `CUSTOS_AUTH_TOKEN` for `custosd`, `X-ATB-Session-Token` for `atb view`). This grants `admin` privileges.
*   **OIDC/JWT**: JSON Web Tokens issued by an OpenID Connect provider. Roles are extracted from JWT claims (e.g., `role` or `roles` claims). If no valid role claim is found, a default role (e.g., `viewer`) is applied.

### `custosd` RBAC

`custosd` enforces RBAC on its HTTP endpoints:

*   `POST /ingest`: Requires `operator` or `admin` role.
*   `GET /receipts`, `GET /receipts/by-hash`, `GET /receipts/:id`, `GET /receipts/:id/verify`, `GET /receipts/:id/attestation`: Require `viewer` or higher role.
*   `GET /health`, `GET /custody/key`: Publicly accessible (no authentication required).

### `atb view` RBAC

`atb view` enforces RBAC on its API routes (`/api/v1/*`):

*   `GET` endpoints (e.g., `/api/v1/verification`, `/api/v1/bundle/meta`, `/api/v1/bundle/events`, `/api/v1/sessions`, `/api/v1/schema/status`): Require `viewer` or higher role.
*   `POST /api/v1/bundle/verify`, `POST /api/v1/privacy/reveal`: Require `operator` or `admin` role.

### Configuration

Refer to `docs/custos-storage.md` for `custosd` configuration and `atb view --help` for viewer configuration.

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

`--custos-endpoint <url>` lodges the completed bundle with a configured Mortise
endpoint when a session closes. `ATB_CUSTOS_TOKEN`, when set, supplies the Bearer
token from the environment. With neither option configured, interception remains
local and does not perform network custody operations. Mortise ingests whole
bundles, verifies them, and returns a signed receipt; it does not accept
individual events.

## atb incident

`atb incident list`, `report`, and `export` operate on a bundle loaded through
the integrity gate. Reports scope a session for review, but the complete bundle
remains the authoritative hash-chained evidence object. An unsigned bundle is
reported as unsigned rather than treated as signed provenance.

## Identity and retention evidence

Oversight events can carry digest-only `identity_evidence`. Trust and incident
reports surface the provider, subject, assertion type, and digest as
caller-provided evidence. Deployments remain responsible for IdP/JWKS/PKI
verification and for retaining the original assertion under their own access
controls.

Retention events live in `.atb/operations.atb`, separate from the workflow
bundle. Compliance packs include relevant operations evidence when available.
An accepted S3 Object Lock request is not represented as independent proof of
bucket configuration, legal hold, or future object availability.

## Mortise and the in-repo scaffold

The supported companion product and evaluator path live in the
[Mortise repository](https://github.com/pcguest/custos-product). The in-repo
`custos/` module remains a reference implementation and compatibility harness;
new custody product work does not land in ATB.

## Research and planning material

Pages under `docs/maintenance/` are operator recovery notes. Shipped behavior is
defined by the current release tag and specification documents, not by the
roadmap alone.

## Legacy public-export tooling

`scripts/export-public-demo.sh` and its workflow remain as legacy packaging
tools for producing a narrow demonstration tree. They are not a security
boundary and do not define which ATB source is public: this repository itself
is the public source of truth.
