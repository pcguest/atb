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
./atb append --type=milestone --data='{"name":"Project Started","status":"active"}'
./atb append --type=debug --data='{"message":"Hello ATB"}'
```

### 4. Launch Dashboard
```bash
./atb view --ui-experimental
# Opens http://localhost:18888/view
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
Edit `.atb/config.json`:

```json
{
  "pii_fields": ["email", "phone", "ssn"]
}
```

### Set Rate Limits

```json
{
  "reveal_rate_limit": {
    "burst": 10,
    "refill_per_minute": 1
  }
}
```

### Export Evidence
```bash
./atb export --output evidence.zip
# Includes manifest.json, checksums, bundle.atb
```

## Troubleshooting

### Dashboard won't load
- Check port: `lsof -i :18888`
- Rebuild UI: `cd web && npm run build && npm run export`

### Trust Score is 0
- Run `./atb verify` first
- Check bundle integrity

### Role selector not working
- Clear localStorage: `localStorage.clear()` in browser console
- Refresh page

## Next Steps
- Read `docs/api/openapi.yaml` for API details
- Explore `docs/security.md` for security model
- Join Discord for community support

Questions? Open an issue with `v1.1.0` label.
