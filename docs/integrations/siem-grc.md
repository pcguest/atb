# SIEM and GRC integration

ATB is a local-first system: audit traces are stored on the originating infrastructure. For teams that forward evidence into centralised monitoring or audit tools, ATB provides deterministic export formats that can be ingested into SIEM (Security Information and Event Management) and GRC (Governance, Risk, and Compliance) platforms.

## Exporting evidence

Evidence is exported as a ZIP archive using the `atb export` command. ATB supports three primary formats: `compliance`, `soc2`, and `gdpr`.

### Example commands

```bash
# General compliance export for incident review
atb export --format compliance --output review-2026-04.zip

# SOC 2 evidence for a specific bundle with a verification sidecar
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-audit.zip --with-verify

# GDPR Data Subject Request (DSR) export
atb export --format gdpr --type dsr --subject-id user_99 --output dsr-user99.zip
```

## ZIP structure and artefacts

Every export ZIP is structured under an `evidence/` directory to ensure consistent ingestion.

### Common base files
- `evidence/manifest.json`: Metadata about the export, including generated date and verification status.
- `evidence/checksums.sha256`: SHA-256 hashes for all files in the archive.
- `evidence/checksums.chain`: Metadata linking file hashes to the cryptographic head hashes of the source bundles.
- `evidence/docs/`: Core ATB security and compliance documentation.

### Format-specific artefacts

| Format | Key Artifacts | Purpose |
| :--- | :--- | :--- |
| **`compliance`** | `evidence/reports/trust-report.json` | Full completeness and assurance scoring (CAS). |
| **`soc2`** | `evidence/audit_trail.jsonl` | Filtered, security-relevant events in JSONL format. |
| **`soc2`** | `evidence/soc2_evidence_manifest.json` | Mapping of events to SOC 2 control criteria (e.g., CC6.1). |
| **`gdpr`** | `evidence/dsr_<id>.json` | Redacted subject-specific records for Article 15 requests. |
| **`gdpr`** | `evidence/ropa_summary.json` | Summary of processing activities for Article 30 (RoPA). |

### The verification sidecar (`.verify.json`)
When `--with-verify` is used, ATB writes a `<output>.verify.json` file next to the ZIP. This sidecar contains the full output of `atb verify --json`. GRC systems should ingest this file to automatically confirm the **integrity** (chain valid) and **completeness** (CAS grade) of the evidence package.

---

## Ingestion patterns

### SIEM integration (monitoring)
For security monitoring, the `soc2` export format is recommended because it produces an `audit_trail.jsonl` file filtered for high-impact events (auth, config changes, alerts).

1.  **Extract**: Automate the extraction of `evidence/audit_trail.jsonl` from the ZIP.
2.  **Ship**: Forward the JSONL lines to your SIEM collector (e.g., via Splunk HEC, Syslog-ng, or a cloud-native log ingestor).
3.  **Map**: Map the standard ATB fields (`type`, `timestamp`, `actor_id_hash`, `data`) to your SIEM’s common information model.

### GRC integration (audit and governance)
For GRC platforms (e.g., Vanta, Drata, or custom evidence lockers), the goal is to store the ZIP as an immutable artefact of proof.

1.  **Store**: Upload the ZIP archive to your GRC’s evidence repository or an S3 bucket with versioning and object lock enabled.
2.  **Verify**: Ingest the `.verify.json` sidecar. Use the `integrity.chain_valid` and `cas.overall` fields to trigger automated alerts if evidence is tampered with or falls below a required completeness threshold (e.g., CAS Grade < B).
3.  **Audit**: During a formal audit, provide the ZIP and the sidecar. The auditor can run `atb verify` against the included `.atb` bundles to independently confirm the evidence.

## Data ownership and privacy

ATB follows a local-first model. Traces remain on your infrastructure until you explicitly choose to export them.

- **Local Traces**: Raw `.atb` files are stored locally and are not automatically synced to any external service.
- **Artifact-Based Export**: Exports are discrete artefacts. You control when they are generated and where they are sent.
- **Privacy-Aware**: The `gdpr` export format automatically applies redacting or hashing to PII fields, ensuring that exported evidence complies with data minimization principles.
