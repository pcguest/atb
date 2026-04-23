# Compliance export

ATB can package local audit evidence into zip archives for `compliance`, `soc2`, and `gdpr` workflows.

## Command

```bash
atb export --format compliance --output compliance-evidence.zip
```

```bash
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
```

```bash
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
```

```bash
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

With full verify sidecar:

```bash
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip --with-verify
# Writes soc2-evidence.zip and soc2-evidence.zip.verify.json
```

Dry-run preview:

```bash
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip --dry-run
```

## Specifications

- [SOC 2 Type II Export Specification](./soc2.md)
- [GDPR Export Specification](./gdpr.md)

## What gets verified during export

- Active bundles under `run.atb/*.atb` are loaded and verified.
- Archived bundles under `archive.atb/` are loaded and verified.
- Archive ledger integrity is verified when `archive.atb/index.ndjson` exists.

If bundle or ledger verification fails, export exits with a non-zero status.

## Format scope

- `compliance` produces a general-purpose evidence zip for incident review, audit follow-up, and cross-functional handoff.
- `soc2` produces a zip with SOC2-specific evidence files and control-oriented structure.
- `gdpr` produces a zip with GDPR-specific structure for either DSR (`--type dsr`) or RoPA (`--type ropa`) output.

These exports are local archives. They are convenient evidence packs, not
immutable storage on their own. If you need retained evidence outside the
local trust boundary, store the exported zip or the pushed bundle in
an organisation-controlled WORM-capable system.

## Evidence package layout

The `compliance` export zip is structured under `evidence/` and includes:

- `evidence/manifest.json`
- `evidence/checksums.sha256`
- `evidence/checksums.chain`
- `evidence/bundles/active/...` (active `.atb` files)
- `evidence/bundles/archived/...` (archived `.atb` files and ledger when present)
- `evidence/reports/verify.json`
- `evidence/reports/trust-report.json`
- `evidence/reports/archive-ledger.json`
- `evidence/config/atb-config.json` when local config exists
- `evidence/docs/...` with core docs and compliance docs when present

The `soc2` and `gdpr` formats also write zip archives under `evidence/`, but each adds format-specific files:

- `soc2` adds `evidence/soc2_evidence_manifest.json`, `evidence/audit_trail.jsonl`, and `evidence/verification_report.json`.
- `gdpr --type dsr` adds `evidence/dsr_<subject_id>.json`.
- `gdpr --type ropa` adds `evidence/ropa_summary.json`.

## Verify sidecar

Use `--with-verify` when you want the full verify report JSON written next to the export zip.

- `--with-verify` writes `<output>.verify.json`.
- The sidecar contains the full `atb verify --json` report for the exported bundle.
- Dry-run mode previews the export and does not write the zip or the sidecar.

## Operational notes

- Use dry-run mode in CI or pre-audit checks to validate expected inclusion paths.
- Store generated evidence zips in your organisation-controlled audit repository.
- Keep source bundles and archive ledger unchanged after export so evidence remains reproducible.
