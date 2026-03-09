# ATB Project Status

Last updated: 2026-03-09

## Completed

- Phase 0 (Foundation): hash chaining, canonicalization, bundle format, CI baseline.
- Phase 1 (Encryption Layer): `atb encrypt` and `atb decrypt` with cross-SDK parity.
- Phase 2 (Event Schema): optional `actor_id`, `org_id`, and `workspace_id` with backward compatibility.
- Phase 3 (Retention + Archive + Export): `atb config retention`, `atb archive`, and `atb export --format compliance`.
- Phase 4 (Compliance Exports): SOC2/GDPR exports with deterministic evidence output and schema validation coverage.

## Phase 4: Compliance Exports (SOC2/GDPR)

- **Status:** ✅ Complete (2026-03-09)
- **Commit:** `f91bdc8`
- **Deliverables:**
  - `atb export --format soc2`: maps events to Trust Services Criteria (CC6.1-CC9.1).
  - `atb export --format gdpr`: supports DSR (Article 15) and RoPA (Article 30) with PII redaction.
  - Deterministic output artifacts (stable ZIP generation with injected time).
  - Golden test fixtures for schema validation.

## Current Focus

- Post-Phase 4 hardening and docs maintenance.

## Source of Truth

- Completion log: [../.atb-agent/registry/completed-phases.md](../.atb-agent/registry/completed-phases.md)
- Active work: [../.atb-agent/registry/active-agents.md](../.atb-agent/registry/active-agents.md)
