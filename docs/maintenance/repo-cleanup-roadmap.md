# ATB repository cleanup roadmap

Status: planning baseline for a developer-first, automatic audit-capture release candidate.
Scope: the whole repository, not only the automated capture layer.
Audience: maintainers, Codex sessions, reviewers, and future contributors.

## Executive position

ATB already has a credible integrity core: append-only bundles, canonical hashing, verification, signing, export paths, SDK surfaces, a local viewer, and CI/security workflows. The repository is not merely a prototype. It is closer to an advanced engine with a partially productised shell.

The next phase should not be random feature work. It should be a deliberate cleanup and closure programme that turns ATB into a coherent developer-first audit toolkit:

- every supported intake path is explicit;
- every recorded event is tamper-evident;
- every supported signing path can be verified offline;
- every capture mode can explain what was captured and what was not;
- every release has a trust story;
- every new contributor or Codex agent can understand the architecture without reverse-engineering the repository.

The central product distinction must remain honest: ATB can prove integrity of what was recorded. Automatic capture and audit-completeness profiles are the work required to move from integrity-only evidence to higher-confidence audit coverage.

## Current infrastructure map

| Area | Current infrastructure | Cleanup direction |
| --- | --- | --- |
| Core bundle model | Append-only bundle, manifest, hash chain, verification semantics, signing and export concepts | Preserve the core. Reduce ambiguity around versioning, provenance fields, and supported algorithms. |
| AI trace model | Dedicated AI trace specification and event taxonomy | Convert taxonomy into first-party adapters and profile checks. |
| CLI | Broad command surface across init, capture/import, verify, sign/export/view style workflows | Clarify command grouping, help output, examples, and failure messages. |
| SDKs | Python and TypeScript SDK surfaces exist | Treat SDKs as first-class intake routes. Add adapter examples and parity tests. |
| Viewer | Local dashboard exists and is verification-aware | Stabilise local API contract, failure diagnostics, filters, and evidence export flows. |
| CI/CD | CI, security, release-oriented workflows, cross-platform testing and SDK checks | Add clearer support matrix, coverage/benchmark gates, CodeQL/SBOM/provenance targets, and workflow ownership docs. |
| Security | Scanner baseline and cryptographic design discipline | Close signer verification gaps, sanitise sensitive errors, document key and remote signer threat models. |
| Docs | Strong specs and architecture docs | Add maintainer-facing roadmap, automatic capture definition, and Codex execution briefs. |
| Release posture | Versioned release materials and automation exist | Move towards signed provenance, reproducible artifacts, and consumer verification instructions. |

## Cleanup principles

1. Trust boundaries first. Anything touching hashing, signing, canonicalisation, verification, timestamping, encryption, or remote signers must be treated as release-blocking and heavily tested.
2. Product honesty over marketing. Do not imply complete audit coverage until ATB can prove capture completeness for a supported profile.
3. One core engine, many intake routes. CLI, SDKs, adapters, imports, capture wrappers, and MCP flows should converge through the same bundle semantics.
4. Fail closed for evidence. Verification failures, missing audit events, unknown algorithms, and incomplete manifests should be surfaced explicitly.
5. Keep the local-first thesis. Avoid prematurely adding hosted multi-tenant platform scope.
6. Make Codex work reviewable. Prefer small PRs with exact acceptance criteria over broad speculative rewrites.

## Priority roadmap

### P0: Stabilise the repository baseline

Objective: remove avoidable friction before deeper implementation.

Actions:

- Align Go, Python, Node, pnpm/npm, and CI versions across `go.mod`, workflow files, SDK manifests, docs, and release notes.
- Add or update a single support matrix covering operating systems, Go version, Python version, Node version, package managers, and browser support for the viewer.
- Add a version-drift check that fails CI when declared versions disagree.
- Add a lightweight benchmark suite for bundle append, verify, export, and viewer/API load paths.
- Add coverage reporting for Go and SDK tests, even if thresholds start advisory.
- Ensure repository docs describe the current release state without overstating automatic capture.

Acceptance criteria:

- A new contributor can identify supported tool versions from one file.
- CI enforces or reports version parity.
- Baseline benchmark numbers are produced in CI or a maintainer command.
- README and architecture docs do not claim automatic completeness unless profile checks support it.

### P0: Close trust and verification gaps

Objective: all supported evidence-producing paths must be verifiable offline.

Actions:

- Implement offline verification for every advertised signature algorithm and backend, especially ECDSA-P256 remote/KMS paths if they remain supported.
- Add fixture bundles for local Ed25519, remote signer, KMS-style ECDSA, timestamped bundles, encrypted bundles, snapshots, and corrupted examples.
- Sanitise remote signer error handling so response bodies or sensitive fragments are not leaked by default.
- Document signer provenance semantics, including local, remote, KMS, backend identifier, key identifier, algorithm, and manifest version expectations.
- Add tests proving unknown algorithms fail closed with clear diagnostics.

Acceptance criteria:

- `atb verify` can verify every supported signer path offline.
- Unsupported algorithms fail with typed errors and useful operator guidance.
- Remote signer failures do not print sensitive response bodies by default.
- Fixture corpus covers positive and negative evidence cases.

### P0: Define automatic audit capture precisely

Objective: establish what automatic capture means for ATB and what it does not mean.

Actions:

- Add first-party capture profiles: `cli-basic`, `python-agent`, `typescript-agent`, and later `mcp-session`.
- Define required events for each profile: session start, process metadata, prompt/input, model request metadata, model response metadata, tool call, tool output hash, approval/denial, error, session end, and final bundle manifest.
- Add `atb verify --profile audit` or equivalent profile checks that report missing required events.
- Add `atb audit init` or `atb instrument` to configure capture defaults and print framework-specific integration snippets.
- Ensure automatic capture reports its own blind spots: network traffic not intercepted, provider-side logs not guaranteed, external systems only represented by captured inputs/outputs, and imported histories marked as retrospective.

Acceptance criteria:

- Automatic capture has a written definition and profile schema.
- A supported workflow can be captured with minimal manual code.
- Verification can distinguish tamper evidence from capture completeness.
- Reports say exactly what was captured, what was inferred, and what was outside ATB's visibility.

### P1: Improve developer experience

Objective: make ATB usable by someone who did not write the repository.

Actions:

- Create a linear quickstart: install, initialise, capture, verify, view, export.
- Add one Python agent example and one TypeScript agent example that generate verifiable bundles.
- Improve CLI help text around capture limitations and audit profiles.
- Add troubleshooting docs for common verification, signer, timestamp, and viewer failures.
- Add architecture diagrams that show intake routes, core engine, bundle boundary, verification, viewer, export, and remote storage.

Acceptance criteria:

- A new user can produce and verify a bundle in under 10 minutes from docs alone.
- Examples are executed in CI or at least smoke-tested.
- Viewer and CLI terminology matches the specs.

### P1: Release trust and supply-chain posture

Objective: make the release itself trustworthy enough for an evidence product.

Actions:

- Add CodeQL if it is not already active.
- Generate SBOMs for release artifacts.
- Add signed provenance/attestation for releases, initially targeting a pragmatic SLSA Build L2 consumer story.
- Document how a user verifies release artifacts.
- Pin or justify GitHub Actions versions and review workflow permissions.
- Add a release checklist that includes spec parity, fixture verification, SDK parity, security scan status, and provenance publication.

Acceptance criteria:

- Release artifacts include provenance or a documented transition plan.
- Workflow permissions are least-privilege where practical.
- Security docs explain vulnerability reporting and supported versions.

### P1: Repository structure and maintainability

Objective: reduce cognitive load and prepare for more contributors.

Actions:

- Group docs into clear zones: `docs/specs`, `docs/guides`, `docs/operations`, `docs/maintenance`, if a migration is worthwhile.
- Add an index page for maintainers that links specs, architecture, CI, release, security, and audit capture docs.
- Audit naming across packages and commands for consistent terms: bundle, event, trace, capture, import, verify, sign, anchor, export, profile.
- Add issue templates for bug, security-hardening task, adapter request, capture profile, and release checklist.
- Add labels aligned with the roadmap: `p0-trust`, `p0-audit-capture`, `p1-dx`, `p1-release`, `p2-adapter`, `docs`, `security`, `ci`.

Acceptance criteria:

- A maintainer can locate the canonical document for each subsystem.
- GitHub issues map cleanly to roadmap workstreams.
- Terminology is consistent across README, CLI, specs, and dashboard copy.

### P2: Adapter and intake expansion

Objective: make automatic capture real for practical agent workflows.

Actions:

- Python: add generic context manager/decorator, OpenAI SDK wrapper, LangChain/LangGraph adapter where feasible.
- TypeScript: add generic wrapper, OpenAI SDK wrapper, LangChain/LangGraph adapter where feasible.
- CLI: ensure `atb capture run -- <command>` records process metadata, command hash, stdout/stderr hashes, exit status, start/end events, and environment redaction policy.
- MCP: define whether MCP server operations are intake, control-plane operations, or both.
- Imports: distinguish live capture from retrospective import with explicit provenance fields.

Acceptance criteria:

- Each supported adapter has an example, test fixture, and audit profile.
- Adapter output verifies through the same core engine as CLI events.
- Retrospective imports cannot masquerade as live capture.

## Non-goals for this cleanup phase

- Hosted multi-tenant SaaS.
- Organisation RBAC.
- Broad observability dashboards.
- Model quality scoring.
- Claiming legal admissibility.
- Capturing traffic or data that the runtime cannot see.
- Replacing provider logs, SIEM, tracing, or compliance systems.

## Suggested Codex PR sequence

1. `docs: add maintenance roadmap and audit-capture definition`
2. `ci: align support matrix and version drift checks`
3. `security: harden remote signer diagnostics`
4. `verify: add signer fixture corpus and ECDSA-P256 verification`
5. `audit: add profile schema and completeness checks`
6. `capture: improve cli capture session events`
7. `sdk-python: add automatic audit capture wrapper`
8. `sdk-typescript: add automatic audit capture wrapper`
9. `docs: add end-to-end quickstarts and troubleshooting`
10. `release: add provenance, SBOM, and release verification docs`

## Final-state definition

ATB is finalised for a developer-first release when it can:

- ingest supported AI/agent workflow activity through documented live capture routes;
- write events through the same append-only, hash-chained bundle boundary;
- sign, timestamp, export, and verify those bundles offline;
- explain capture completeness through named audit profiles;
- present evidence through CLI, SDK, and local dashboard flows;
- distinguish live capture from retrospective import;
- ship releases with a clear supply-chain trust story;
- document limitations precisely enough that a reviewer can understand what ATB proves and what it does not prove.
