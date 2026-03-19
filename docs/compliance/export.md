# Compliance Export

ATB can package local audit evidence into a single zip archive for SOC2 and GDPR workflows.

In `v1.1.0`, run export commands from a repo checkout. The exporter reads required templates from `docs/compliance/` relative to the working directory.

## Command

```bash
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
```

```bash
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
```

```bash
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

Dry-run preview:

```bash
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip --dry-run
```

## Specifications

- [SOC 2 Type II Export Specification](./soc2.md)
- [GDPR Export Specification](./gdpr.md)

## What Gets Verified During Export

- Active bundles under `run.atb/*.atb` are loaded and verified.
- Archived bundles under `archive.atb/` are loaded and verified.
- Archive ledger integrity is verified when `archive.atb/index.ndjson` exists.

If bundle or ledger verification fails, export exits with a non-zero status.

## Evidence Package Layout

The export zip is structured under `evidence/` and includes:

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

## Operational Notes

- Use dry-run mode in CI or pre-audit checks to validate expected inclusion paths.
- Store generated evidence zips in your organization-controlled audit repository.
- Keep source bundles and archive ledger unchanged after export so evidence remains reproducible.
