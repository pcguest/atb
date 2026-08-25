# ATB LOCAL PUBLIC-READINESS REPORT

> Local convergence evidence captured on 2026-08-24. This is release-preparation
> material, not a claim that the GitHub repository or packages are currently
> public or current. The accepted implementation/documentation state is commit
> `a4c5179`; the report-only commit containing this file is expected to be its
> child and does not change runtime or release behaviour.

## Repository truth map

Implementation and tests outrank prose in this matrix. “Desired wording” is
the canonical wording for current human-facing documentation.

| Concept | Code reality | Test reality | Current docs | Desired canonical wording | Action |
| --- | --- | --- | --- | --- | --- |
| Product identity | The Go module, CLI, bundle format, SDK packages, profiles, viewer, and exports are all ATB. There is no Tenon runtime package. | CLI and package tests exercise ATB without Tenon or Mortise. | The README mostly has the right hierarchy; some deep pages unnecessarily lead with Tenon or Mortise. | “Tenon is the umbrella product identity. ATB and Mortise are separate products within it.” | Keep Tenon at the product-family/documentation boundary; remove it from low-level ATB explanations. |
| ATB definition | ATB captures or imports events, writes canonical hash-chained records, verifies them, evaluates profiles, derives incident findings, exports evidence, and serves a local viewer. | End-to-end and command-contract tests cover these paths. | “Audit trail”, “logging”, “evidence core”, and compliance-led descriptions compete. | “ATB is the open-source, local-first evidence core for independently verifiable records of AI-agent behaviour.” | Use one evidence-led opening in README, docs hub, architecture, and public-surface docs. |
| Tenon | No runtime dependency or hosted requirement exists. | No test requires Tenon. | Correct in the README but over-represented in some release material. | Umbrella product family only: `Tenon → ATB; Tenon → Mortise`. | Confine Tenon to product relationship sections. |
| Mortise | `internal/mortise` and CLI endpoint flags are optional HTTP integration clients. Local capture, verify, incident, view, profile, and pack paths do not require them. | Mortise client/conformance tests use explicit configured endpoints; core tests remain local. | Several current pages call Mortise a “framework” or make the companion repository central. | “Mortise is the optional commercial custody and organisational layer for ATB evidence.” | Describe Mortise as an integration/custody boundary; do not imply it is an ATB runtime component. |
| Bundle | A `.atb` file is NDJSON containing canonical events and their chain hashes. Manifest v1 is the default; v2 is readable and opt-in. | Bundle, malformed-input, adversarial, manifest-version, and golden tests pin the format. | Generally accurate, though “bundle store” and directory/file language vary. | “A bundle is a portable `.atb` evidence record containing canonicalised, hash-chained records.” | Add glossary and link to the format specification. |
| Event | Callers submit semantic activity with type, data, and optional attribution/trace context. | Go/Python/TypeScript append and schema tests cover event shapes. | “Event” and “record” are sometimes interchangeable. | “An event is semantic activity submitted to ATB.” | Standardise the distinction. |
| Record | `bundle.Record` is the canonical event plus its stored chain hash. | Hash and tamper tests recompute every record. | Often called an event after storage. | “A record is the canonicalised, chained representation stored in a bundle.” | Standardise the distinction. |
| Append-only / tamper-evident | Mutations append semantically, then atomically replace the local file under an advisory lock. A filesystem actor can replace or roll back the whole file before external custody. | Locking, atomic-write, mutation, reorder, insertion, deletion, and truncation tests cover these properties. | Some pages use “append-only” or “immutable” without the local-file qualification. | “Append-only record semantics and tamper-evident bundles; not inherently immutable.” | Remove unqualified immutability claims except for correctly configured external WORM/Git history contexts. |
| Capture completeness | ATB can only preserve activity submitted through SDK, wrapper, intercept, import, MCP, OTel, or explicit append paths. | Profile-gap and incident tests reason only over recorded evidence. | Core security docs are accurate; CAS and incident wording sometimes sounds universal. | “ATB proves integrity of recorded evidence. It does not prove completeness or truth of capture.” | Repeat this exact boundary at high-value claim sites. |
| Offline verification | Local bundles, hash-chain verification, signatures, anchors with supplied/system roots, profiles, incident analysis, packs, and the viewer can operate without a hosted verifier. Explicit remote/custody/network integrations are separate. | Installed-binary, verify, profile, incident, and export tests cover local operation. | Generally accurate; some diagrams route everything through a single core or Mortise. | “Verification and investigation are local and independently runnable; configured external evidence may require separately retained trust material.” | Correct diagrams and cross-links. |
| Signatures | Go supports Ed25519 local signing and configured KMS/remote ECDSA provenance; SDKs inspect/verify supported signature records. Signatures authenticate configured keys, not real-world identity. | Local, KMS contract, signature provenance, and cross-SDK tests cover supported algorithms. | Mostly accurate but sometimes compresses provenance into identity. | “Configured signature provenance proves a matching key signed the recorded bundle state.” | Qualify identity claims. |
| Anchoring | ATB requests and verifies RFC 3161 timestamp evidence. Verification checks imprint, signer signature, and certificate chain. TSA responses are capped at 4 MiB. | Anchor parsing, imprint, signature, chain, response-limit, and CLI tests exist. | Security docs describe verification accurately. | “Optional RFC 3161 evidence can corroborate that a bundle state existed by the TSA time; it does not make local storage immutable.” | Keep the trust qualification and bounded response documented. |
| Encryption | CLI and SDK paths use AES-256-GCM; the Go CLI uses a versioned PBKDF2-SHA256 envelope. Encryption protects confidentiality, not capture completeness or custody. | Go and cross-language encryption parity tests pin the wire semantics. | Generally accurate. | “Optional AES-256-GCM encryption protects bundle confidentiality in storage or transit.” | Keep separate from integrity and custody claims. |
| SDK architecture | Go, Python, and TypeScript independently implement bundle/canonicalisation semantics. They share schema, event vocabulary, and deterministic golden vectors; Python/TypeScript do not call a Go core. | Cross-language canonicalisation, encryption, schema, and signed-bundle tests enforce agreement. | `docs/architecture.md` incorrectly routes both SDKs through one “Core Engine”; README calls Go “the implementation” in a way that can imply delegation. | “Independent implementations agree against shared schemas and deterministic vectors.” | Redraw architecture and describe Go as the reference contract implementation, not a runtime dependency of other SDKs. |
| OpenTelemetry | `pkg/otel` decodes supported OTLP/JSON trace payloads and maps attributable spans to ATB events; `atb import otel` is bounded input. It is not a collector or generic tracing backend, and mapping coverage is partial. | Decoder, receiver, translator, command, and input-limit tests cover the implemented subset. | Some package comments still call it a scaffold while current docs imply broader support. | “OpenTelemetry is an input/integration surface for mapping relevant telemetry into ATB evidence.” | State the exact implemented OTLP/JSON subset and backlog modern GenAI semantic-convention mapping. |
| MCP | `atb mcp serve` is a beta local stdio bridge with a deliberately limited tool surface. It does not auto-instrument third-party MCP servers. | Command-surface and MCP server tests cover the shipped tools. | The MCP guide is detailed and largely honest. | “MCP is an optional semantic capture boundary, not ATB’s architecture or a generic MCP observability layer.” | Keep beta status and precise limitations. |
| Viewer | The full static viewer is embedded only in a source build after `web` is built. `go install` uses the `noembed` path and serves install guidance. The viewer is loopback-only and authenticated by a generated session token or paired OIDC settings. | Host, auth, fail-closed API, installed-binary, and viewer tests cover the distinction. | README mentions the distinction; quickstart and install docs must make it unavoidable. | “`go install` provides the CLI and a minimal view guidance page; `make build` provides the full embedded local viewer.” | Preserve rather than disguise the packaging distinction. |
| Authentication | Viewer APIs fail closed. Session tokens are local; optional OIDC resolves discovery/JWKS over an operator-supplied issuer with time and discovery-body bounds. Actor fields in evidence remain caller asserted. | JWT lifecycle, timeout, rotation, RBAC, viewer, and handler tests cover configured behaviour. | Security docs accurately separate viewer auth from evidence identity, but network-boundary limits need explicit treatment. | “Viewer authentication controls local API access; it does not independently prove event actor identity.” | Clarify OIDC/JWKS network trust and SSRF/operator configuration boundary. |
| Incident analysis | `atb incident list/report/export` deterministically derives findings from bundle records and points findings back to sequences/hashes. | Incident command, review workflow, and demo tests cover missing approval and failed action findings. | Strong but some wording turns “not recorded” into “did not exist”. | “Incident findings are deterministic observations about recorded evidence.” | Replace universal absence claims with “no matching approval is present in the recorded evidence”. |
| CAS | CAS means “Completeness Assurance Score”. It is a local weighted score over evidence visible within a selected ATB profile; integrity failure forces the overall score to zero. It cannot measure universal capture completeness. | Profile-specific, sub-score, integrity-failure, and custom-DSL tests cover it. | The CAS guide contains both precise caveats and shorthand such as “how much of the workflow was recorded”. | “CAS estimates profile-scoped evidence coverage within the recorded bundle.” | Keep the repository’s existing expansion; tighten shorthand and never present CAS as external assurance. |
| Profiles | Built-in and YAML DSL profiles declare evidence obligations and can produce deterministic pass/fail/warning results over recorded events. | Registry, DSL, obligation, gap, and all-profile tests cover them. | Usually accurate; compliance-led pages can overstate outcomes. | “A profile is a declarative ATB evidence-obligation model evaluated against recorded evidence.” | Use “ATB-defined profile”, not certification. |
| Evidence/compliance packs | CLI exports deterministic archives containing bundle, verification/profile reports, mapped artefacts, manifests, and checksums; optional Mortise receipts can be included. | Determinism, manifest, checksum, and customer-handoff tests cover exports. | “Compliance pack” is sometimes treated as an outcome rather than a packaging label. | “A portable evidence pack; `compliance pack` is the CLI’s mapping-oriented package name, not legal certification.” | Lead with evidence pack and retain the command name precisely. |
| Custody | Local storage remains operator-controlled. ATB can explicitly push to S3/Object Lock or an optional Mortise endpoint, recording request evidence without proving continuing external enforcement. | Push, retention, receipt, queue, and export tests cover explicit integrations. | Some wording calls accepted retention “immutable”. | “Custody is external preservation and control after evidence creation.” | Separate creation/integrity from independently operated custody. |
| Installation | Source build, Go install/noembed, Python packaging, TypeScript packaging, and Docker release paths are distinct. No cloud account or external database is required. | Release-contract and installed-binary smoke tests cover package surfaces. | Commands exist but the public-user path is fragmented. | “CLI-first five-minute path, followed by explicit SDK and full-viewer installation paths.” | Consolidate installation truth in quickstart/support matrix. |
| Security resource limits | Individual NDJSON records are capped at 16 MiB; chatlog/OTel imports, proxy bodies, and TSA responses are bounded. Total bundle bytes and record count are not capped in `LoadReader`. | Per-record, import/body, and TSA-response limit tests exist; total bundle/count limit tests do not. | Current security docs enumerate the bounded and accepted unbounded surfaces. | “Treat bundles and configured network endpoints as untrusted inputs; current bounded and unbounded surfaces are explicit.” | Classify total bundle limits for a versioned hardening release; do not silently change bundle compatibility in this pass. |
| Public/open-source status | Source carries an MIT licence and public-ready metadata, but `pcguest/atb` is currently PRIVATE. | Licence/release contract tests inspect shipped metadata. | Current prose often says “open” as intended product framing and links to URLs unavailable to unauthenticated users. | “ATB is MIT-licensed and prepared for open-source publication; the remote remains private until the controlled publication pass.” | Keep public-facing prose ready, but record the current remote state only in release-preparation material. |
| Current version | Source markers and local tag are `1.15.2`; `main` and `v1.15.2` are commit `362b43c`. The current branch is nine commits ahead at the starting point. | Version scripts require marker agreement. | README calls v1.15.2 current. | “v1.15.2 is the tagged source baseline, not a fully published multi-registry release.” | Preserve history and recommend a new patch release only after gates pass. |
| Release state | GitHub’s latest Release is v1.15.0; no GitHub Release exists for v1.15.2; PyPI and npm latest are 1.14.5. The repository is private. | Release-contract tests validate local artefacts but cannot establish registry publication. | Historical launch/runbook/checklist material mixes v1.15.0, v1.15.1, and v1.15.2. | “Release truth is per surface; skipped or partial releases remain historical facts.” | Mark historical docs, make operational docs version-neutral, and target v1.15.3 if convergence remains patch-compatible. |

## Document classification

### Current

- `README.md`, `SECURITY.md`, `VERSIONING.md`, `CONTRIBUTING.md`
- `docs/README.md`, `docs/architecture.md`, `docs/security.md`,
  `docs/public-surface.md`, `docs/roadmap.md`, `docs/quickstart.md`, and current
  guides/specifications

### Historical

- `ACCEPTANCE.md` — v1.15.0 acceptance evidence
- `RELEASE-REVIEW-v1.15.0.md` when present — v1.15.0 release review
- version-specific review/acceptance evidence under the release documentation

Historical documents retain their original claims and commands. They receive
only an unmistakable historical-status banner where needed.

### Release-specific

- `CHANGELOG.md` — cumulative release history plus the next candidate entry
- `LAUNCH-COLLATERAL.md` — historical v1.15.1 collateral
- `PRE-FLIGHT-CHECKLIST.md` — historical preflight unless replaced by a
  version-neutral checklist
- `LAUNCH-RUNBOOK.md` — historical Tenon/Mortise launch plan unless replaced
  by the current controlled publication sequence
- `docs/release.md` and `docs/release/*` — current version-neutral release
  process and provenance guidance

## A. Repository identity

```text
branch:                   fix/ci-gate-scanners
accepted source HEAD:     a4c5179 (report-only commit follows)
base:                     main at 362b43c
working tree:             clean after the report commit
current source version:   1.15.2
recommended next version: 1.15.3
```

The convergence pass started at `6958f8f508c83b48ab1f23515a6878778bd6ff24`
on `fix/ci-gate-scanners`, nine commits ahead of `main`. The accepted source
state is sixteen commits ahead of `main`; the report-only commit makes the final
branch seventeen commits ahead. That is the original nine commits plus the
eight bounded convergence commits listed below. There is no divergence in the
other direction.

`main` and the peeled annotated tag `v1.15.2^{}` both resolve to
`362b43c46190742b66313a431b9b23898b33cdb6`. The current branch is therefore a
strict descendant of the tagged source baseline. No tag was moved, recreated,
or deleted.

Starting repository evidence was recorded before edits:

```text
pwd:        /Users/paddyguest/atb
branch:     fix/ci-gate-scanners
remote:     origin https://github.com/pcguest/atb.git
visibility: PRIVATE when checked on 2026-08-24
latest GitHub Release: v1.15.0
PyPI atb-sdk latest:   1.14.5
npm @pcguest/atb-sdk:  1.14.5
```

The local launcher was Go 1.26.3; all authoritative repository pins and test
runs used the supported patched Go 1.26.7 toolchain. Other principal local
tools were Python 3.11, Node.js 22.23.1, and npm 10.9.8. Docker CLI was present
but its daemon was unavailable. The CodeQL CLI was not installed.

Release truth is surface-specific: `v1.15.2` is a valid tagged source baseline,
but it was not a complete GitHub/PyPI/npm release. Because `v1.15.1` and
`v1.15.2` already exist as tags, the converged public release must not reuse or
rewrite either. The changes preserve the `.atb` format and public APIs, so the
next appropriate SemVer is `v1.15.3`, after merge and remote release gates.

Bounded convergence commits:

1. `df90c5b fix(ci): restore portable release and security gates`
2. `b976e56 fix(security): bound remote inputs and trust claims`
3. `0c99a77 docs(architecture): align implemented evidence trust model`
4. `467a4df docs(product): converge ATB Tenon and Mortise framing`
5. `3e6be7a docs(release): reconcile current and historical release truth`
6. `868ac0f fix(ci): isolate scanners from ignored local caches`
7. `a4c5179 docs(release): make automated publication authoritative`
8. the report-only commit containing this document

## B. Product framing

**Tenon:** Tenon is the umbrella product identity for ATB and Mortise. It is not
an ATB runtime component, verification dependency, hosted requirement, or
infrastructure service required to use ATB.

**ATB:** ATB is the open-source, local-first evidence core for independently
verifiable records of AI-agent behaviour. It captures agent and tool activity
into portable, tamper-evident `.atb` bundles that can be verified and
investigated offline without a cloud account, hosted verifier, external
database, Tenon service, or Mortise.

**Mortise:** Mortise is the optional commercial custody and organisational
layer for ATB evidence. It may provide durable custody, WORM retention, signed
custody receipts, transparency/witness services, fleet views, organisational
access controls, and operational support; it is not required for ATB capture,
verification, incident analysis, local review, profile evaluation, or evidence
pack generation.

Approved relationship:

```text
Tenon
├── ATB      open-source, local-first evidence core
│    │
│    ├─ capture / import / intercept
│    ├─ canonical, hash-chained .atb bundles
│    ├─ offline verification and incident analysis
│    ├─ profiles and evidence packs
│    └─ local viewer
│
└── Mortise  optional commercial custody and organisational layer
```

Approved evidence flow:

```text
AI agent / application / framework
              |
              v
           CAPTURE
      SDK / intercept / import
              |
              v
       canonical ATB event
              |
              v
       RFC 8785 canonicalise
              |
              v
        SHA-256 record chain
              |
              v
          .atb bundle
       /         |          \
  verify      incident      view
                  |
                  v
          evidence / pack
                  |
                  | optional
                  v
              Mortise
          custody boundary
```

Go, Python, and TypeScript are independent implementations of the same bundle
semantic contract. They agree through shared schemas and deterministic golden
vectors; Python and TypeScript do not call a hidden Go engine. Optional
Ed25519/ECDSA signing, RFC 3161 timestamp evidence, and AES-256-GCM encryption
add configured evidence properties without changing the basic product model.

## C. Trust statement

Canonical wording:

> **ATB proves the integrity of what was recorded: the integrity and recorded
> ordering of presented records, detection of post-recording mutation,
> configured signature provenance, configured timestamp evidence, and whether
> the recorded evidence satisfies an ATB-defined obligation profile.**

> **ATB does not prove that every relevant event was captured, that an actor
> was honest before capture, that model output is true, that an AI decision is
> correct, that a claimed actor is a real-world person, that omitted events do
> not exist, that legal compliance has been achieved, or that local evidence
> has immutable external custody.**

Short form: **ATB proves integrity of recorded evidence. ATB does not prove
completeness or truth of capture.**

All absence findings must remain scoped to the bundle: for example, “no
matching approval is present in the recorded evidence,” never “no approval
existed anywhere.” “Tamper-evident” is not silently upgraded to “immutable,”
and an evidence/compliance pack is not a legal certification.

## D. Technical state

The following are the final local outcomes. `BLOCKED` means the check could not
run in this environment, not that it was treated as green.

| Gate | State | Evidence |
| --- | --- | --- |
| `go vet` | PASS | All first-party packages under Go 1.26.7. |
| Go tests | PASS | Full repository suite, including integration/release/schema packages. |
| Go race tests | PASS | Full first-party package set. |
| Go staticcheck | PASS | Pinned/local static analysis. |
| Go coverage threshold | PASS | 80.2% aggregate coverage; threshold 80%. |
| Cross-language golden vectors | PASS | Canonicalisation, schema, signature, and encryption parity. |
| Python tests | PASS | 136 tests. |
| Python build/smoke | PASS | sdist and wheel built, checked, and imported in isolation. |
| TypeScript tests | PASS | 108 tests. |
| TypeScript typecheck | PASS | `tsc --noEmit`. |
| TypeScript lint | PASS | Configured lint target. |
| TypeScript build/package smoke | PASS | Build and isolated tarball import. |
| Viewer tests | PASS | 67 tests. |
| Viewer typecheck | PASS | Configured TypeScript check. |
| Viewer lint | PASS | ESLint with zero warnings. |
| Viewer production build | PASS | Next.js static production export. |
| Viewer live E2E | PASS | Firefox, live embedded ATB server, 2/2. |
| Viewer accessibility | PASS | Strict Firefox accessibility audit, 1/1. |
| Source build | PASS | Full embedded-viewer CLI. |
| `go install`/`noembed` | PASS | CLI and explicit minimal viewer-guidance path. |
| CLI quickstart | PASS | Bundle generation and local flow. |
| `atb verify` | PASS | Quickstart and artifact smoke paths. |
| `atb incident` | PASS | Deterministic report/export and incident demo. |
| `atb view` | PASS | Loopback, session auth, embedded/noembed distinction. |
| `actionlint` | PASS | All workflows. |
| Version consistency | PASS | Go/Python/TypeScript/web at 1.15.2. |
| Support-matrix consistency | PASS | Go 1.26.7, Python and Node support markers. |
| Generated bindings | PASS | Schema-generated files unchanged. |
| Documentation/internal links | PASS | Current local Markdown targets resolved. |
| Bash syntax/portability review | PASS | Bash 3-compatible package arrays on macOS jobs. |
| `git diff --check` | PASS | No whitespace errors. |
| Release preflight | PASS | All seven stages; Docker stage explicitly skipped and reported separately. |
| Exact `make gate-gold-release` | PASS | Full hygiene, race, coverage, scanners, E2E, and accessibility. |
| Native release artifact build | PASS | Five CLI binaries, checksums, Python/npm/web artifacts. |
| Artifact smoke/version checks | PASS | Artifacts report 1.15.2 and import/run as expected. |
| SBOM/provenance preparation | PASS | Local SPDX structure and provenance/checksum manifest prepared. |
| Gitleaks working/publishable tree | PASS | No secrets. |
| Gitleaks reachable history | PASS | 1,598 commits, 14.22 MiB scanned, no secrets. |
| gosec | PASS | 165 files, 35,626 lines, 0 issues. |
| govulncheck | PASS | Strict run reported no reachable vulnerabilities. |
| npm audit (viewer) | PASS | 0 vulnerabilities. |
| npm audit (TypeScript SDK) | PASS | 0 vulnerabilities. |
| Trivy publishable filesystem | PASS | 0 HIGH/CRITICAL findings. |
| Bandit HIGH/HIGH | PASS | 0 findings. |
| Docker image build/smoke | BLOCKED | Docker daemon unavailable locally. Required remote workflow remains enabled. |
| Trivy built-image scan | BLOCKED | No local Docker daemon. Required remote job remains enabled. |
| Local CodeQL-equivalent CLI run | BLOCKED | CodeQL CLI unavailable. Required GitHub workflow remains enabled. |
| Third-party notice completeness | FAIL | Existing notice covers PageIndex but is not yet a generated/exhaustive distribution inventory. |
| Supplemental Ruff scan | FAIL | 73 findings; Ruff is not a configured authoritative repository gate. |
| Supplemental mypy scan | FAIL | 9 findings; mypy is not a configured authoritative repository gate. |

The scanner gate is not green by omission. Trivy excludes only ignored local
build/tool caches and installed `node_modules`; it still scans tracked source
and manifests, while both npm trees are separately audited from their exact
lockfiles. Gitleaks, gosec, govulncheck, Bandit, npm audit, and Trivy all fail
closed on real findings.

## E. Changes made

Each path changed between the recorded starting HEAD and the accepted source
state is listed once:

- `.githooks/pre-commit` — removed warning-only vulnerability handling and kept required checks fail-closed.
- `.github/CODEOWNERS` — added realistic solo-maintainer ownership and future enforcement guidance.
- `.github/workflows/ci.yml` — aligned Go pins and made cross-platform package selection portable.
- `.github/workflows/codeql.yml` — aligned the supported Go toolchain and least-privilege workflow posture.
- `.github/workflows/gold-release.yml` — pinned current scanner/tool versions and preserved the full gold gate.
- `.github/workflows/release-gates.yml` — made Go package enumeration compatible with default macOS Bash.
- `.github/workflows/release.yml` — aligned toolchain/scanners and retained verified, ordered publication gates.
- `.github/workflows/security.yml` — granted Gitleaks only `contents: read` and updated bounded scanner inputs.
- `ACCEPTANCE.md` — marked v1.15.0 acceptance evidence unmistakably historical.
- `CHANGELOG.md` — recorded the convergence/security/release corrections without rewriting prior releases.
- `CONTRIBUTING.md` — aligned contribution and product-boundary language with current ATB.
- `Dockerfile` — updated the authoritative patched Go builder pin.
- `LAUNCH-COLLATERAL.md` — marked the v1.15.1 material historical rather than current launch truth.
- `LAUNCH-RUNBOOK.md` — classified the former Tenon/Mortise launch plan as historical.
- `Makefile` — fixed portable package selection, strict gates, and scanner isolation from ignored caches.
- `README.md` — replaced compliance-led framing with the evidence-led five-minute product/trust path.
- `RELEASE-REVIEW-v1.15.0.md` — marked v1.15.0 review evidence historical.
- `SECURITY.md` — converged integrity/completeness, custody, raw-content, and network-boundary claims.
- `cmd/atb/incident.go` — scoped incident absence findings to recorded evidence.
- `cmd/atb/intercept.go` — added an explicit sensitive raw-content capture warning.
- `cmd/atb/intercept_test.go` — asserted the raw-content warning.
- `docs/README.md` — made the documentation hub the canonical navigation layer.
- `docs/architecture.md` — replaced the fictional shared engine with independent SDK implementations and vectors.
- `docs/cas-guide.md` — scoped Completeness Assurance Score to profile-visible recorded evidence.
- `docs/ciso-acceptance-guide.md` — removed certification/immutability overclaims.
- `docs/compliance/article-12-mapping.md` — retained regulatory mapping as evidence support, not certification.
- `docs/glossary.md` — established canonical bundle/event/record/capture/profile/CAS/custody terminology.
- `docs/guides/incident-forensics.md` — scoped deterministic findings and emphasized hash-addressed evidence.
- `docs/guides/rbac-configuration.md` — separated viewer access control from evidence identity/custody.
- `docs/integrations/README.md` — framed integrations as evidence inputs/outputs rather than ATB's identity.
- `docs/integrations/siem-grc.md` — distinguished observability telemetry from independently verifiable evidence.
- `docs/maintenance/README.md` — classified current versus historical maintenance material.
- `docs/maintenance/history-rewrite-plan.md` — marked the obsolete rewrite proposal historical and non-operative.
- `docs/maintenance/manual-test-playbook.md` — marked the former private-trunk playbook historical.
- `docs/maintenance/public-mit-extraction-checklist.md` — replaced extraction assumptions with current public-readiness checks.
- `docs/mortise-handoff.md` — described Mortise as an optional integration/custody boundary.
- `docs/mortise-storage.md` — separated ATB evidence creation from optional custody storage.
- `docs/profiles.md` — described profiles as ATB evidence obligations, not compliance certification.
- `docs/provability-ladder.md` — tightened integrity, corroboration, and external-custody claim levels.
- `docs/public-surface.md` — aligned intended public state with the currently private remote.
- `docs/quickstart.md` — clarified the CLI-first path and full-source viewer distinction.
- `docs/release.md` — made version truth and the automated release workflow authoritative.
- `docs/release/README.md` — separated current release process from historical release records.
- `docs/release/provenance.md` — aligned build, verification, signing, and publication ordering.
- `docs/roadmap.md` — moved OTel/MCP breadth and bounded hardening to post-convergence work.
- `docs/security.md` — documented implemented bounds, accepted limits, and honest trust claims.
- `docs/support-matrix.md` — updated supported Go and installation/package distinctions.
- `examples/bundles/demo-workflow/generate.sh` — kept the demo deterministic under current bundle semantics.
- `examples/bundles/incident-capture/README.md` — turned the existing primitives into a believable incident proof.
- `examples/bundles/incident-capture/application-log.txt` — added the intentionally incomplete application-log contrast.
- `examples/go/README.md` — aligned Go examples with the local-first product boundary.
- `go.mod` — set Go 1.26.7 and the bounded patched `x/net` dependency.
- `go.sum` — recorded the compatible Go dependency update.
- `internal/agent/first_run.go` — corrected first-run product/custody language.
- `internal/anchor/anchor.go` — capped RFC 3161 responses at 4 MiB.
- `internal/anchor/anchor_test.go` — covered the TSA response limit.
- `internal/incident/incident.go` — made derived observations explicitly bundle-scoped.
- `internal/proxy/recorder.go` — retained bounded, privacy-aware provider recording.
- `internal/proxy/session.go` — aligned session capture warnings and trust wording.
- `pkg/auth/jwt.go` — bounded OIDC/JWKS refetch network time.
- `pkg/otel/otel.go` — described the implemented OTLP/JSON subset rather than a generic tracing core.
- `pkg/otel/receiver.go` — aligned receiver comments with bounded input behaviour.
- `pkg/otel/translator.go` — documented partial semantic mapping precisely.
- `scripts/check-support-matrix.sh` — enforced the Go 1.26.7 support truth.
- `scripts/govulncheck.sh` — made vulnerability scanning strict and reproducible.
- `scripts/quality-evidence.sh` — excluded installed dependency trees without excluding first-party packages.
- `scripts/release-check.sh` — made clean release preflight package selection portable and reproducible.
- `sdk/typescript/package-lock.json` — applied the minimal compatible transitive vulnerability fix.
- `test/golden/encrypt_parity_test.go` — made Python environment discovery reproducible.
- `test/golden/schema_parity_test.go` — made cross-language parity use the intended local environment.
- `tools/go.mod` — updated pinned security/build tool dependencies.
- `tools/go.sum` — recorded the patched tool dependency graph.
- `web/out/placeholder.txt` — clarified clean-checkout/go-install versus full embedded-viewer builds.
- `web/package-lock.json` — applied the minimal compatible transitive vulnerability fix.
- `LOCAL-PUBLIC-READINESS.md` — records the truth map, acceptance evidence, blockers, and controlled publication plan.

No change altered the `.atb` manifest format, RFC 8785 profile, chain input,
event schema, or cross-language golden semantics.

## F. Documentation convergence

**Contradictions fixed:** evidence versus logging; integrity versus capture
completeness; tamper-evident versus immutable; evidence support versus legal
certification; profile-scoped CAS versus universal completeness; Tenon family
versus runtime; Mortise integration versus required infrastructure; independent
SDKs versus a fictional shared Go engine; OTel input versus generic tracing;
current source tag versus registry release; `go install` viewer guidance versus
full source-build viewer; manual versus workflow-owned GitHub Release creation.

**Historical documents preserved:** v1.15.0 acceptance/review evidence,
v1.15.1 launch collateral, earlier launch/runbook material, and obsolete
history-rewrite/private-trunk plans retain their historical statements and now
carry explicit non-current context.

**Terminology standardised:** bundle, event, record, capture, verification,
profile, Completeness Assurance Score (CAS), incident finding, evidence pack,
compliance pack, custody, ATB, Mortise, and Tenon are defined in the glossary
and used consistently in current entry points.

**Remaining intentional distinctions:**

- `compliance pack` remains the established CLI command/package label; it is an evidence package, not certification.
- Go is the reference contract implementation; Python and TypeScript independently implement compatible semantics.
- `go install` supplies the full CLI but only viewer installation guidance; `make build` embeds the complete viewer.
- OTel accepts a documented OTLP/JSON subset; modern GenAI/MCP mapping breadth is backlog, not implied support.
- `v1.15.2` remains historical source/tag truth even though registry surfaces stopped at 1.14.5.
- Public-facing prose is prepared for the intended open-source state; this report alone records that the remote is still private.

## G. Public exposure audit

| Area | State | Result |
| --- | --- | --- |
| Secret scan | PASS | Publishable tree and reachable history contain no Gitleaks findings. |
| History scan | PASS | All reachable commits/tags scanned; no credential finding or rewrite requirement. |
| Personal/private artifacts | PASS | No tracked credential, `.env`, private endpoint, or signing key. Historical personal paths are labelled context, not runtime configuration. |
| Local ignored material | PASS | Ignored local signing-key/demo material exists with restrictive permissions but has never been tracked or made reachable. It will not be uploaded by Git; do not archive or screen-share it. |
| Oversized/generated files | PASS | No oversized binary in reachable publishable history; largest reachable blob is approximately 543 KiB. Build outputs remain ignored/controlled. |
| Licensing | PASS | ATB source is MIT-licensed; no obvious incompatible direct distribution licence was identified. |
| Third-party notices | FAIL | `THIRD_PARTY_NOTICES` is not yet exhaustive for release binaries and the bundled web application. |
| Security policy | PASS | `SECURITY.md` gives disclosure and trust-boundary guidance. |
| Contribution policy | PASS | `CONTRIBUTING.md` is current and product-aligned. |
| CODEOWNERS | PASS | Solo-maintainer routing exists without pretending a second reviewer exists. |
| Issue/PR templates | PASS | Bug, feature, integration, security, and PR templates are present. |
| Release artifacts | PASS | Local native artifacts, checksums, package smoke tests, SPDX structure, and provenance preparation succeeded. Docker artifact remains separately BLOCKED. |

An unreachable historical local pack object of about 46 MiB was observed, but
it is not reachable from any branch or tag in the publishable graph. It is not
a reason to rewrite public history. The ignored local key material is a clone
hygiene concern, not repository content; it must never be force-added.

## H. Remaining blockers

### BLOCKS PUBLIC REPOSITORY

None found in the publishable Git graph or working source state.

### BLOCKS RELEASE

1. Generate and review an exhaustive third-party notice/licence inventory for
   the exact CLI, Python, npm, and bundled-web artifacts; update
   `THIRD_PARTY_NOTICES` and include it in the release artifacts.
2. Push the branch while the repository is still private and require PASS from
   hosted CodeQL and the Docker build/Trivy image scan. They were locally
   BLOCKED, not waived.
3. After merge, prepare `v1.15.3` in a release PR, rerun the manual non-publishing
   artifact workflow, and confirm trusted PyPI/npm/signing configuration before
   pushing the tag. Do not reuse `v1.15.2`.

### DOES NOT BLOCK PUBLICATION

- Supplemental Ruff (73 findings) and mypy (9 findings) are honest FAIL results
  but are not configured support gates; triage them in a bounded Python-quality
  pass instead of silently making them release policy mid-convergence.
- The remote is still private by design until the controlled publication pass.
- Registry versions at 1.14.5 and source tag 1.15.2 are documented historical
  facts; the next release resolves them with 1.15.3 rather than rewritten tags.
- Ignored local key/demo material is absent from Git and does not travel with a
  push. It remains a local operational hygiene item.
- A local Docker daemon and CodeQL CLI are not required to expose source once
  their mandatory hosted checks have passed on the private PR.

### POST-RELEASE

- Add a total bundle-byte and total record-count policy before the next
  hardening release. Current 16 MiB per-record and import/body bounds are
  documented; whole-bundle memory growth is a **SHOULD FIX FOR NEXT RELEASE**.
- Consider issuer/JWKS destination allowlisting or a custom dial policy for
  deployments accepting untrusted OIDC configuration. Today it is a
  **DOCUMENTED ACCEPTED LIMIT** at the trusted-operator configuration boundary;
  time and discovery-body bounds are implemented.
- Expand modern OTel GenAI/MCP semantic compatibility only as an evidence-input
  mapping; do not turn ATB into a generic observability platform.

Bounded P2 classification:

| Finding | Classification | Disposition |
| --- | --- | --- |
| TSA response size | MUST FIX BEFORE PUBLIC | Fixed at 4 MiB and tested. |
| Sensitive raw-content capture warning | MUST FIX BEFORE PUBLIC | Fixed and tested. |
| Retention/immutability wording | MUST FIX BEFORE PUBLIC | Corrected across current docs. |
| Total bundle bytes/record count | SHOULD FIX FOR NEXT RELEASE | Documented accepted current limit; no format change made. |
| Untrusted bundle memory exhaustion | SHOULD FIX FOR NEXT RELEASE | Same bounded loader work item as total limits. |
| OIDC/JWKS destination control | DOCUMENTED ACCEPTED LIMIT | Operator-supplied trust boundary; network timeout now bounded. |

## I. GitHub publication plan

Do not execute any publication step until the local report commit is clean.
The exact ordered plan is:

### 1. Git operations while the repository is private

```bash
git status --short --branch
git log --oneline main..fix/ci-gate-scanners
git push -u origin fix/ci-gate-scanners

gh pr create --repo pcguest/atb \
  --base main \
  --head fix/ci-gate-scanners \
  --title "chore: converge ATB for public release" \
  --body-file LOCAL-PUBLIC-READINESS.md
```

Wait for every required PR check below. Review the public diff and rendered
README/security/release documents unauthenticated via a clean clone or archive.
Then preserve the bounded commits with a merge commit:

```bash
gh pr merge --repo pcguest/atb --merge --delete-branch=false
git switch main
git pull --ff-only origin main
git status --short --branch
```

### 2. Solo-maintainer ruleset/branch protection

Create a `main` ruleset before visibility changes:

- target `main`; active enforcement;
- require a pull request, but set required approving reviews to **0** while
  there is only one maintainer;
- do not require CODEOWNERS approval until a second active maintainer exists;
- require conversation resolution and dismiss stale approvals once approvals
  become required;
- block force pushes and branch deletion;
- restrict direct updates to the PR merge path; keep only a documented
  repository-owner emergency bypass;
- require branches to be up to date before merge;
- require these observed check contexts after the private PR has created them:
  `CI / Core Validation (Layers A/B)` for Ubuntu, macOS, and Windows;
  `CI / Golden Parity (Layers A/B)`;
  `CI / Profile Parity Gate (Layer C)`;
  `Release gates / verify`;
  `Security Gate / Layer D - Secret Scan`;
  `Security Gate / Layer D - Go Security Scan`;
  `Security Gate / Layer D - Python Security Scan`;
  `Security Gate / Layer D - Node Security Scan`;
  `Security Gate / Layer D - Trivy Filesystem Scan`;
  `Security Gate / Layer D - Trivy Docker Image Scan`;
  `CodeQL / analyze`; and `Version gate / check-versions` when version paths
  change.

Do not require the tag-only Gold/Publish jobs on ordinary PRs. Require signed
commits or signed tags only after a reliable signing/recovery process is in
place; an unenforceable key policy is worse than an explicit unsigned history.

When two or more active maintainers exist, raise required approvals to two,
require CODEOWNERS review, retain stale-approval dismissal, and require approval
of the most recent push by someone other than its author.

### 3. Security settings while private

Enable and verify dependency graph, Dependabot alerts/security updates, secret
scanning, push protection, private vulnerability reporting, and the existing
advanced CodeQL workflow. Confirm Actions is restricted to the pinned actions
already present. Run all workflows once and record green results.

### 4. GitHub visibility

After the private PR, ruleset, and security checks are accepted:

```bash
gh repo edit pcguest/atb \
  --visibility public \
  --accept-visibility-change-consequences
```

This is the first and only visibility-changing command. Verify the repository,
README links, licence, security policy, default branch, topics, issue templates,
and clone access from a logged-out browser and a clean unauthenticated clone.

### 5. Prepare the `v1.15.3` release

First complete the third-party notice blocker and receive green hosted
CodeQL/Docker-image results. Then follow `docs/release.md` from updated `main`:

```bash
git switch main
git pull --ff-only origin main
git switch -c release/v1.15.3

# Edit CHANGELOG.md and every version marker listed in docs/release.md.
ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh
bash scripts/check-support-matrix.sh
SKIP_DOCKER=1 scripts/release-check.sh
PATH=/tmp/atb-tools/bin:$PATH make gate-gold-release

git add CHANGELOG.md cmd/atb/main.go sdk/python/pyproject.toml \
  sdk/python/atb/__init__.py sdk/typescript/package.json \
  sdk/typescript/package-lock.json web/package.json web/package-lock.json \
  README.md THIRD_PARTY_NOTICES
git commit -m "chore(release): prepare v1.15.3"
git push -u origin release/v1.15.3
gh pr create --repo pcguest/atb --base main --head release/v1.15.3 \
  --title "chore(release): prepare v1.15.3" \
  --body "Prepare the converged public release v1.15.3."
```

Merge only after all private/public PR gates pass. Manually dispatch
`release.yml` on the release commit or merged `main`; because it is not running
on a tag, it builds and verifies but cannot publish:

```bash
gh workflow run release.yml --repo pcguest/atb --ref main
gh run list --repo pcguest/atb --workflow release.yml --limit 3
```

### 6. Tag, release creation, PyPI, and npm

Confirm the PyPI trusted publisher, package ownership, npm granular token/2FA,
and `ATB_SIGNING_KEY_PEM` before the tag exists. Then:

```bash
git switch main
git pull --ff-only origin main
git tag -a v1.15.3 -m "v1.15.3"
git push origin v1.15.3
```

Use a signed tag instead of `-a` only if the signing key and recovery process
have been tested. The tag-triggered workflow is authoritative: it creates a
draft GitHub Release, publishes the verified PyPI and npm artifacts, signs and
attaches the final evidence bundle, and finally makes the Release public. Do
not also run `gh release create` or manual `twine`/`npm publish` commands.

Monitor each surface separately:

```bash
gh run list --repo pcguest/atb --workflow version-gate.yml --limit 3
gh run list --repo pcguest/atb --workflow gold-release.yml --limit 3
gh run list --repo pcguest/atb --workflow docker-publish.yml --limit 3
gh run list --repo pcguest/atb --workflow release.yml --limit 3
gh release view v1.15.3 --repo pcguest/atb
npm view @pcguest/atb-sdk@1.15.3 version dist.integrity
python -m pip index versions atb-sdk
```

Finally download every GitHub asset into a clean directory, verify checksums,
SBOM/provenance, CLI versions, Python/npm installs, Docker labels, the signed
ATB release bundle, and unauthenticated documentation links.

## J. Rollback plan

**Public exposure reveals an issue:** immediately disable affected features or
make the repository private if the issue is material and platform policy permits;
revoke/rotate any exposed credential first, open a private advisory, determine
whether the data is merely in the working tree or reachable history, and only
then plan a coordinated history rewrite if genuinely necessary. A normal bug
uses a revert/fix PR; never rewrite tags for ordinary defects.

**Package publication fails:** keep the GitHub Release in draft. Preserve logs,
checksums, and already-published bytes. Correct credentials/workflow state and
retry only steps that are byte-identical and registry-safe. If the candidate's
bytes must change, advance to `v1.15.4`; do not overwrite `v1.15.3`.

**One registry publishes and another does not:** record the release as partial,
do not mark the GitHub Release latest, and do not delete the successful registry
version. If the remaining registry can receive the already verified identical
artifact, resume it; otherwise publish a coordinated next patch and clearly
document the partial version on every release surface.

**CI fails after push:** leave the repository public but do not tag or release.
Revert through a PR or add a bounded fix commit on the branch, rerun all required
checks, and merge only when green. A failed check is never converted to allowed
failure to unblock publication.

**Release artifacts disagree:** leave or return the GitHub Release to draft,
stop remaining publication jobs where safe, retain all evidence for diagnosis,
and rebuild every artifact from one clean commit. Compare hashes and embedded
versions before resuming. If any registry already received different bytes,
use a new patch version and document the abandoned/partial release.

## K. Go / no-go decision

The local publishable Git graph is technically green for source exposure, its
trust/product language is coherent, and no public-repository blocker remains.
This decision authorises only the separate controlled visibility pass after the
private PR and hosted checks. It does **not** authorise a `v1.15.3` package or
GitHub Release until the three release blockers in section H are closed.

GO — READY TO MAKE ATB PUBLIC
