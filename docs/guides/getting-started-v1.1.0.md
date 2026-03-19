# Getting Started with ATB v1.1.0

## Prerequisites
- Go 1.21+
- Node.js 18+
- npm 9+

## Quick Start (5 minutes)

### 1. Build ATB
```bash
git clone https://github.com/pcguest/atb.git
cd atb
git checkout v1.1.0
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

### 5. Verify Integrity
```bash
./atb verify --format json
# Output: {"status":"valid","chain_length":2,...}
```

## Role-Based Views

### Engineer View
- See raw event payloads
- Debug information
- Full hash chain visualization

### Auditor View
- Compliance statistics
- Export evidence button (SOC2/GDPR)
- No raw PII (backend-filtered)

### Executive View
- Trust Score (0-100)
- High-level trends
- Summary cards only

## Advanced Usage

### Configure PII Masking
Point `ATB_PII_FIELDS_PATH` to a JSON file or update `docs/compliance/pii-fields.json`:

```json
{
  "pii_fields": ["email", "phone", "ssn"]
}
```

Privacy reveal auditing is always appended to `bundle.atb` in v1.1.0. The `--log-reveals` flag is retained for CLI compatibility but is no longer required.

### Export Evidence
```bash
./atb export --format soc2 --output evidence.zip
# Includes manifest, checksums, and bundle evidence
```

## Troubleshooting

### Dashboard won't load
- Check port: `lsof -i :8080`
- Rebuild UI: `cd web && npm run build && npm run export`

### Trust Score is 0
- Run `./atb verify` first
- Check bundle integrity

## Next Steps
- Read `docs/api/openapi.yaml` for API details
- Explore `docs/security.md` for security model
- Open an issue with the `v1.1.0` label if you find a problem
