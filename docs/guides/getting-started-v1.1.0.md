<!-- Archived release-planning draft. Not maintained. -->
# Archived getting-started draft for v0.9.0-beta

This document is retained for historical reference only. It describes a
beta evaluation flow from the pre-release series and is not current
operator guidance. Use [docs/quickstart.md](../quickstart.md) for the
current quickstart.

## Prerequisites
- Go 1.25.0
- Node.js 20.9+ if you need to rebuild the embedded web export
- npm 10+ if you need to rebuild the embedded web export

## Quick start (5 minutes)

### 1. Build ATB
```bash
git clone https://github.com/pcguest/atb.git
cd atb
git checkout main
go build -o atb ./cmd/atb
```

### 2. Initialize Bundle
```bash
./atb init
# Creates ./run.atb/bundle.atb
```

### 3. Append Events
```bash
./atb append milestone --data='{"name":"Project Started","status":"active"}'
./atb append debug --data='{"message":"Hello ATB"}'
```

### 4. Launch Dashboard
```bash
./atb view --ui-experimental
# Opens http://localhost:8080/view/
```

Plain `./atb view` still serves the default local viewer. The role-based dashboard remains behind `--ui-experimental` in the current beta.

### 5. Verify Integrity
```bash
./atb verify --format json
# Output: {"status":"valid","chain_length":2,...}
```

## Role-based views

### Engineer view
- See raw event payloads
- Debug information
- Full hash chain visualization

### Auditor view
- Compliance statistics
- Export summary button for a local JSON evidence snapshot
- No raw PII (backend-filtered)

### Executive view
- Trust Score (0-100)
- High-level trends
- Summary cards only

## Advanced usage

### Configure PII masking
ATB ships with bundled default PII masking rules. To override them, point `ATB_PII_FIELDS_PATH` to your own JSON file:

```json
{
  "pii_fields": ["email", "phone", "ssn"]
}
```

Privacy reveal auditing is always appended to `bundle.atb` in the current beta. The `--log-reveals` flag is retained for CLI compatibility but is no longer required.

### Export evidence

```bash
./atb export --format soc2 --bundle run.atb/bundle.atb --output evidence.zip
# Includes manifest, checksums, and bundle evidence
```

## Troubleshooting

### Dashboard won't load
- Check port: `lsof -i :8080`
- Rebuild UI: `cd web && npm run build`

### Trust score is 0
- Run `./atb verify` first
- Check bundle integrity

## Next steps
- Read `docs/api/openapi.yaml` for API details
- Explore `docs/security.md` for security model
- Read `docs/use-cases/internal-audit-privacy-review.md` for buyer-facing review workflow
- Open an issue if you find a problem
