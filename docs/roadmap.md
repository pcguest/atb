# ATB Roadmap & Versioning

This document outlines the journey of ATB from its experimental origins to a stable 1.0 release.

## The Versioning Reset: v1.x to v0.9.0-beta

You may notice in the [Changelog](../CHANGELOG.md) that ATB previously carried versions up to `v1.8.1` before resetting to `v0.9.0-beta`. 

This reset was an intentional decision to **honestly reflect the pre-production status** of the project. While the core hash-chaining logic is robust, the higher-level specifications—including obligation profiles, CAS scoring, and the bundle export formats—are still being refined through internal pilots and community feedback. 

By moving to `v0.9.0-beta`, we are signaling that:
1.  **Breaking Spec Changes are Possible:** We may still adjust the canonical event taxonomy or profile schemas.
2.  **Pilot Readiness:** The current version is suitable for pilots, internal experiments, and portfolio-quality integration, but not yet for mission-critical production systems where API stability is a hard requirement.

## Current Status: v0.9.x (Beta)

The `v0.9.x` series focuses on **hardening and transparency**. 

### Guarantees
-   **Integrity-First:** The SHA-256 hash-chaining and RFC 8785 canonicalization are the most stable parts of the system.
-   **Verified Core:** TSA anchoring and Ed25519 bundle signing are fully implemented and verified by the CLI.
-   **Open Specifications:** The bundle format and AI trace taxonomy are documented and open for review.

### Known Limitations
-   **Schema Fluidity:** Event types and field requirements may shift as we align with emerging AI safety standards.
-   **CLI-Centric:** While Python and TypeScript SDKs exist, the Go CLI remains the authoritative implementation for verification and reporting.

## Target: v1.0.0 (Stable)

The `v1.0.0` milestone represents the point where ATB becomes a stable, predictable foundation for production AI audit trails.

### v1.0.0 Guarantees
-   **Stable Specification:** [spec-v1.0.md](spec-v1.0.md) will be frozen. Future changes will follow a strict, versioned migration path.
-   **Certified Profiles:** The built-in obligation profiles will be fully validated against regulatory frameworks like the EU AI Act and ISO 42001.
-   **Cross-SDK Parity:** Full feature parity across Go, Python, and TypeScript for bundle writing and basic integrity checks.
-   **Support Policy:** A clear support and deprecation policy for older bundle formats and CLI versions.

---

## Milestones to 1.0

### Phase 1: Hardening (v0.9.x - Current)
-   [x] Align documentation with actual TSA/Ed25519 behavior.
-   [x] Document all built-in Obligation Profiles and CAS scoring.
-   [x] Strengthen adversarial testing for bundles and dashboard APIs.
-   [x] Document SIEM and GRC integration patterns.

### Phase 2: Refinement (v0.10.x)
-   [ ] **Spec Finalization:** Lock the `ai.*` event taxonomy and profile DSL.
-   [ ] **SDK Parity:** Ensure Python/TS SDKs support the latest profile evaluation signals.
-   [ ] **Enhanced Export:** Add support for signed evidence exports (encrypted handoff).
-   [ ] **Performance:** Optimize bundle scanning for very large traces (100k+ events).

### Phase 3: Launch (v1.0.0)
-   [ ] **Final Audit:** Internal security review of the 1.0 candidate.
-   [ ] **Stable API:** Freeze the CLI command structure and JSON output formats.
-   [ ] **Governance Docs:** Comprehensive guidance for CISOs and Auditors on accepting ATB evidence.
