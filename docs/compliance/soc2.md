# SOC 2 Type II evidence bundle

This template provides a standard structure for SOC2 evidence generated from ATB bundles.

ATB provides tamper-evident records of the events that were appended during the workflow. Completeness against the SOC2 control is evaluated per the declared obligation profile, not by universal capture.

This package is evidence support for a SOC 2 review. It is not a SOC 2
attestation, control certification, or substitute for auditor judgement.

## Control criteria

- **CC6.1:** Logical Access Security
- **CC6.6:** System Boundaries
- **CC7.2:** System Monitoring
- **CC8.1:** Change Management
- **CC9.1:** Risk Mitigation

## Evidence requirements

The exporter should include the following ATB-derived evidence:

- Event Log: Chronological list of appended SOC2-relevant events for the selected period.
- Integrity Proof: Hash-chain verification result and algorithm metadata.
- Actor Attribution: Unique actor IDs involved in traced actions.
- Export Metadata: Bundle source path, generated timestamp, and export format.

The SOC2 export produces a zip containing:

- `evidence/bundles/active/run.atb/bundle.atb` (the verified event bundle at the selected bundle path)
- `evidence/soc2_evidence_manifest.json` (control mapping for CC6.1, CC6.6, CC7.2, CC8.1, and CC9.1)
- `evidence/audit_trail.jsonl` (SOC2-relevant event trail)
- `evidence/verification_report.json` (bundle verification summary)
- `evidence/checksums.sha256` (artefact integrity manifest)

## Instructions for auditors

1. Confirm the source bundle was verified before evidence generation.
2. Review the event log against the declared obligation profile and selected audit period.
3. Confirm the integrity section indicates a valid hash chain.
4. Sample-check actor-attributed events against control criteria.

## Auditor notes

This bundle was generated automatically by the current release. Verify the
`chain_valid` field in `evidence/verification_report.json` before relying on the
evidence in an audit review.
