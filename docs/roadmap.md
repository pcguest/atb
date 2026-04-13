# ATB roadmap and versioning

This document explains the historical version reset that led into the
stable series. For the current release state, see [CHANGELOG.md](../CHANGELOG.md).

This document outlines the journey of ATB from its experimental origins to a stable 1.0 release.
All work described in the sections below is complete as of v1.5.0.

## The versioning reset: v1.x to v0.9.0-beta

You may notice in the [Changelog](../CHANGELOG.md) that ATB previously carried versions up to `v1.8.1` before resetting to `v0.9.0-beta`.

This reset was an intentional decision to **honestly reflect the pre-production status** of the project. While the core hash-chaining logic is robust, the higher-level specifications (obligation profiles, CAS scoring, and the bundle export formats) were still being refined through internal pilots and community feedback.

By moving to `v0.9.0-beta`, the intention was to signal that:
1. **Breaking spec changes are possible:** The canonical event taxonomy or profile schemas might still shift.
2. **Pilot readiness:** The version was suitable for pilots, internal experiments, and portfolio-quality integration, but not yet for mission-critical production systems where API stability is a hard requirement.

## v0.9.x shipped work

The `v0.9.x` series was the hardening window for the local verification model, the built-in profiles, and the reviewer-facing documentation set.

-   [x] Six-profile verifier with YAML-backed profile templates and CAS evaluation where supported.
-   [x] `VerifierReport` and `TrustReport` JSON output shapes for `atb verify` and `atb trust-report`.
-   [x] `ComplianceManifest` export for `atb export --format compliance --json`.
-   [x] MCP integration guide in `docs/integrations/mcp.md`.
-   [x] Python and TypeScript SDK event type constants.
-   [x] Performance baseline and benchmark suite.

## v1.0.0-rc target (completed)

-   [x] Ed25519 bundle signing integration test.
-   [x] Native MCP server mode (`atb mcp serve`).
-   [x] CAS support extended to `data_export`, `background_automation`, `policy_decision`, and `human_override`.

## Path to v1.0.0 and beyond (completed)

The stable `v1.0.0` milestone was the point where ATB became a predictable foundation for production AI audit trails, with frozen JSON output shapes, a stable profile DSL, and clear reviewer guidance. All of the following shipped by v1.5.0:

-   [x] `spec-v1.0.md` and the profile DSL frozen.
-   [x] Python and TypeScript SDKs cover the profile evaluation signals.
-   [x] Encrypted bundle support hardened with versioned PBKDF2-SHA256 parameters.
-   [x] CLI command structure and JSON output formats frozen.
-   [x] Governance guidance for CISO and auditor acceptance in `docs/compliance/` and `docs/security.md`.

## Forward roadmap

The following milestones are engineering intent, not committed ship dates.

### v1.6 — Bundle push and WORM export

**Status: Planned**

- `atb push` command exports a sealed bundle to a configurable external target (S3-compatible storage or a local WORM path).
- S3 target: `atb push --target s3://bucket/prefix` uses the standard AWS credential chain and writes the bundle as a single object with a content-addressed key (`sha256:<hash>.atb`).
- WORM locking: ATB sets `x-amz-object-lock-mode: COMPLIANCE` and `x-amz-object-lock-retain-until-date` when `--lock-until` is supplied and the bucket has Object Lock enabled. ATB does not enforce WORM — the bucket policy does.
- Remote verification: `atb verify --remote s3://bucket/sha256:<hash>.atb` verifies a remotely stored bundle without downloading it in full (streaming verify).

See [`docs/spec/bundle-push.md`](./spec/bundle-push.md) for the full design-intent specification, including the current manual workaround.

### v1.7 — LangChain native callback integration

**Status: Planned**

- `ATBCallbackHandler` for LangChain (Python) attaches to any LLM, Chain, or Agent and emits `ai.llm.call`, `ai.tool.exec`, and `ai.chain.run` events automatically.
- Zero-config mode: `ATBCallbackHandler()` with no arguments uses the active bundle in the current working directory.
- See [`docs/integrations/langchain.md`](./integrations/langchain.md) for the current integration guide.

### v1.8 — Profile DSL v1

**Status: Planned**

- YAML-defined custom profiles: operators can define their own obligation profiles without modifying Go code.
- Profile validation tooling: `atb profile validate ./my-profile.yaml` checks a custom profile schema for correctness.
- Profile registry: `atb profile list` shows built-in and user-defined profiles.
