# SOC2 Type II Evidence Bundle

This template provides a standard structure for SOC2 evidence generated from ATB bundles.

## Control Criteria

- **CC6.1:** Logical Access Controls
- **CC7.2:** System Operations
- **CC8.1:** Change Management

## Evidence Requirements

The exporter should include the following ATB-derived evidence:

- Event Log: Complete chronological list of appended events for the selected period.
- Integrity Proof: Hash-chain verification result and algorithm metadata.
- Actor Attribution: Unique actor IDs involved in traced actions.
- Export Metadata: Bundle source path, generated timestamp, and export format.

## Instructions for Auditors

1. Confirm the source bundle was verified before evidence generation.
2. Review the event log for completeness across the audit period.
3. Confirm the integrity section indicates a valid hash chain.
4. Sample-check actor-attributed events against control criteria.

## Auditor Notes

This bundle was generated automatically by ATB v1.1.0. Verify the
`hash_chain_verified` field in export metadata before proceeding with attestation.
