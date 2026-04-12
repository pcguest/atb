# ATB roadmap and versioning

This document explains the historical version reset that led into the
stable series. For the current release state, use [docs/STATUS.md](./STATUS.md)
and [CHANGELOG.md](../CHANGELOG.md).

This document outlines the journey of ATB from its experimental origins to a stable 1.0 release.

## The Versioning Reset: v1.x to v0.9.0-beta

You may notice in the [Changelog](../CHANGELOG.md) that ATB previously carried versions up to `v1.8.1` before resetting to `v0.9.0-beta`. 

This reset was an intentional decision to **honestly reflect the pre-production status** of the project. While the core hash-chaining logic is robust, the higher-level specifications—including obligation profiles, CAS scoring, and the bundle export formats—are still being refined through internal pilots and community feedback. 

By moving to `v0.9.0-beta`, we are signaling that:
1.  **Breaking Spec Changes are Possible:** We may still adjust the canonical event taxonomy or profile schemas.
2.  **Pilot Readiness:** The current version is suitable for pilots, internal experiments, and portfolio-quality integration, but not yet for mission-critical production systems where API stability is a hard requirement.

## Current Status: v0.9.x (Beta)

The `v0.9.x` series is the hardening window for the local verification model, the built-in profiles, and the reviewer-facing documentation set. The Go CLI remains the authoritative implementation for verification and reporting, but the main release surfaces now exist across the repository and are being aligned for the `v1.0.0-rc` cut.

### Shipped in v0.9.x
-   [x] Six-profile verifier with YAML-backed profile templates and CAS evaluation where supported.
-   [x] `VerifierReport` and `TrustReport` JSON output shapes for `atb verify` and `atb trust-report`.
-   [x] `ComplianceManifest` export for `atb export --format compliance --json`.
-   [x] MCP integration guide in `docs/integrations/mcp.md`.
-   [x] Python and TypeScript SDK event type constants.
-   [x] Performance baseline and benchmark suite.

### Remaining limitations in beta
-   **Schema Fluidity:** Event types and field requirements may still shift before the 1.0 surface is frozen.
-   **CLI-Centric:** Python and TypeScript SDKs assist with writing and integration, but the Go CLI remains the verification source of truth.
-   **Partial CAS Coverage:** CAS scoring is not yet available for all six built-in profiles.

## v1.0.0-rc target

The `v1.0.0-rc` checkpoint is a release-readiness pass over the already shipped surfaces, plus a small set of remaining gaps that need to be tracked explicitly.

-   [ ] Add an Ed25519 signing integration test that exercises the end-to-end signing path.
-   [ ] Native MCP server mode. Post-rc.
-   [ ] Add the canonical `ai.mcp.tool_call` event type. Post-rc.
-   [ ] Extend CAS support to `data_export`, `background_automation`, `policy_decision`, and `human_override`. Post-rc.

## Path to v1.0.0

The stable `v1.0.0` milestone remains the point where ATB becomes a predictable foundation for production AI audit trails, with frozen JSON output shapes, a stable profile DSL, and clear reviewer guidance.

### Future work after rc
-   [ ] Freeze `spec-v1.0.md` and the profile DSL with a versioned migration path for later changes.
-   [ ] Ensure Python and TypeScript SDKs cover the final profile evaluation signals expected at 1.0.
-   [ ] Add signed or encrypted evidence handoff improvements to export workflows.
-   [ ] Optimise bundle scanning for very large traces (100k+ events).
-   [ ] Complete the internal security review of the 1.0 candidate.
-   [ ] Freeze the CLI command structure and JSON output formats.
-   [ ] Finalise governance guidance for CISO and auditor acceptance of ATB evidence.
