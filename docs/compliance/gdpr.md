# GDPR Export Specification (Article 15 and 30)

This document defines the schema for Data Subject Requests (DSR) and Records of Processing Activities (RoPA). It enforces strict PII handling, legal basis declaration, and retention compliance.

## 1. Export Command Usage

Data Subject Request (Portability/Access):

```bash
atb export --format gdpr --type dsr --subject-id <user_uuid> --bundle <path> --output <dir>
```

Records of Processing (RoPA):

```bash
atb export --format gdpr --type ropa --bundle <path> --output <dir>
```

## 2. PII Classification and Redaction Rules

Before export, the CLI MUST scan all event payloads for fields marked as sensitive in the schema definition.

| Field Category | Action in DSR Export | Action in RoPA Export |
| --- | --- | --- |
| Direct Identifiers (`email`, `ip`, `user_id`) | Include plain (if matching subject) | Hash (SHA-256 with salt) |
| Sensitive Data (`payment`, `health`, `bio`) | Include plain (if matching subject) | Redact (replace with `[REDACTED]`) |
| System Metadata (`timestamps`, `hashes`) | Include plain | Include plain |
| Third-Party Data (other users' IDs) | Redact (unless linked to subject) | Hash |

Implementation note: Redaction is deterministic. The same input always yields the same redacted output to allow verification of the redaction logic itself.

## 3. Data Subject Request (DSR) Schema

Output: `dsr_<subject_id>.json`

```json
{
  "request_type": "GDPR_ARTICLE_15",
  "subject_id": "usr_9f8e7d6c",
  "export_date": "2026-03-06T12:00:00Z",
  "legal_basis": "contract_performance",
  "data_categories": [
    {
      "category": "access_logs",
      "record_count": 50,
      "retention_rule": "delete_after_2y",
      "records": [
        {
          "event_id": "evt_111",
          "timestamp": "2026-01-10T09:00:00Z",
          "action": "login",
          "context": {
            "ip": "192.168.1.1",
            "user_agent": "Mozilla/5.0..."
          }
        }
      ]
    }
  ],
  "provenance": {
    "source_bundle_hash": "sha256:...",
    "extraction_signature": "ed25519:..."
  }
}
```

## 4. Records of Processing Activities (RoPA) Schema

Output: `ropa_summary.json`

This aggregates processing activities without exposing individual user data.

```json
{
  "controller_id": "org_xyz",
  "reporting_period": "2026-Q1",
  "processing_activities": [
    {
      "purpose": "service_delivery",
      "legal_basis": "contract_performance",
      "data_categories": ["identity", "usage_metrics"],
      "recipient_categories": ["internal_engineering", "cloud_provider_aws"],
      "retention_schedule": "24_months_from_last_activity",
      "security_measures": [
        "AES-256-GCM encryption at rest",
        "Hash-chained audit logs",
        "Role-based access control"
      ],
      "event_volume": 150042
    },
    {
      "purpose": "fraud_detection",
      "legal_basis": "legitimate_interest",
      "data_categories": ["ip_address", "device_fingerprint"],
      "retention_schedule": "90_days",
      "security_measures": [
        "Automated anomaly detection",
        "Manual review workflow"
      ],
      "event_volume": 402
    }
  ]
}
```

## 5. Compliance Checks

The export command MUST fail if:

- The `--subject-id` does not exist in the bundle (for DSR).
- Any event in the chain fails hash verification.
- The bundle retention policy has expired for the requested data range.

## 6. Right to Erasure (Article 17) Note

ATB does not support physical deletion of events from an immutable hash-chained ledger. Instead, erasure is implemented via cryptographic redaction:

- A new tombstone event is appended stating the subject's data is logically deleted.
- Future exports for this subject return an empty dataset with a reference to the tombstone event.
- The original raw data remains in the archive for legal hold (if configured) but is inaccessible via standard export keys.
