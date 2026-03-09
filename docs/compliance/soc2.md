# SOC 2 Type II Export Specification

This document defines the schema and control mapping for generating SOC 2 audit artifacts from ATB bundles. The goal is to provide auditors with cryptographically verifiable evidence of system integrity, access control, and change management.

## 1. Export Command Usage

```bash
atb export --format soc2 --bundle <path> --output <dir>
```

Output artifacts:

- `soc2_evidence_manifest.json`: Machine-readable mapping of events to Trust Services Criteria (TSC).
- `audit_trail.jsonl`: Canonicalized, filtered event log for the audit period.
- `verification_report.json`: Hash-chain validation results and integrity proofs.

## 2. Control Mapping (Trust Services Criteria)

ATB maps internal event types to specific SOC 2 Common Criteria. The export MUST include only events relevant to the selected criteria.

| TSC ID | Control Description | ATB Event Types | Evidence Requirement |
| --- | --- | --- | --- |
| CC6.1 | Logical Access Security | `auth.login`, `auth.logout`, `auth.failure`, `permission.change` | Timestamped record of who accessed what, with IP/User-Agent context. |
| CC6.6 | System Boundaries | `system.config_change`, `network.policy_update` | Before/after state hashes of configuration changes. |
| CC7.2 | System Monitoring | `alert.triggered`, `monitor.anomaly`, `health.check_fail` | Alert payload, resolution timestamp, and actor (if manual). |
| CC8.1 | Change Management | `deploy.start`, `deploy.complete`, `code.merge`, `config.promote` | Commit hash, approver ID, CI/CD run ID, deployment status. |
| CC9.1 | Risk Mitigation | `backup.start`, `backup.complete`, `restore.init` | Backup file hash, storage location ID, verification status. |

## 3. Evidence Manifest Schema (`soc2_evidence_manifest.json`)

The manifest acts as an index for the auditor, linking controls to specific line numbers or hash ranges in `audit_trail.jsonl`.

```json
{
  "audit_period": {
    "start": "2026-01-01T00:00:00Z",
    "end": "2026-03-31T23:59:59Z"
  },
  "bundle_hash": "sha256:<hash_of_source_bundle>",
  "generated_at": "2026-04-01T10:00:00Z",
  "controls": [
    {
      "control_id": "CC6.1",
      "description": "Logical Access Security",
      "evidence_count": 142,
      "sample_ids": ["evt_abc123", "evt_def456"],
      "integrity_proof": {
        "first_event_hash": "sha256:...",
        "last_event_hash": "sha256:...",
        "chain_valid": true
      }
    },
    {
      "control_id": "CC8.1",
      "description": "Change Management",
      "evidence_count": 15,
      "sample_ids": ["evt_ghi789"],
      "integrity_proof": {
        "first_event_hash": "sha256:...",
        "last_event_hash": "sha256:...",
        "chain_valid": true
      }
    }
  ],
  "verifier_signature": "ed25519:<signature_of_manifest>"
}
```

## 4. Integrity and Verification

- Hash chaining: Every event in `audit_trail.jsonl` must contain `prev_hash`. The export process MUST verify this chain before generation. If any link is broken, the export fails with exit code 1.
- Immutability: The generated `soc2_evidence_manifest.json` includes a hash of the source bundle. Any alteration to the source bundle invalidates the manifest.
- No redaction: For SOC2, internal system events are not redacted (unless they contain PII, see GDPR), as auditors require full context of system behavior.

## 5. Auditor Instructions

1. Run `atb verify <bundle>` to confirm local integrity.
2. Open `soc2_evidence_manifest.json` and select random `sample_ids`.
3. Grep `audit_trail.jsonl` for those IDs to verify raw data matches the manifest summary.
4. Recompute the hash of `audit_trail.jsonl` and compare against the `integrity_proof` in the manifest.
