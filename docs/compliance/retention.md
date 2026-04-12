# Retention

ATB retention is local-first and controlled through CLI config.

## Configure retention

Set retention days in local config:

```bash
atb config retention --days 90
```

This writes `./.atb/config.json` with:

- `retention.days` (required for policy-based archive cutoff)
- `retention.archive_dir` (default: `archive.atb`)
- `retention.scope` (default: `run.atb/*.atb`)
- `retention.cutoff_basis` (current supported value: `file_mtime`)

## Run archive

Archive using configured retention days:

```bash
atb archive
```

Archive using an explicit cutoff date:

```bash
atb archive --before 2026-01-01
```

Preview actions without moving files:

```bash
atb archive --dry-run
```

## Archive behaviour

- Candidate bundles are discovered from configured `scope`.
- Bundles are verified before archiving; invalid bundles are skipped.
- Archived bundles are moved under date-partitioned paths in `archive.atb/`.
- A tamper-evident archive ledger is maintained at `archive.atb/index.ndjson`.
- If `run.atb/bundle.atb` is archived, ATB recreates an empty default bundle at `run.atb/bundle.atb`.

## Audit expectations

- Keep `archive.atb/` and `run.atb/` as local evidence stores.
- Use `atb export --format compliance --output <path.zip>` to package active bundles, archived bundles, and reports for audit review.
- Re-run `atb verify` and `atb archive --dry-run` during periodic checks to confirm integrity and retention behavior.
