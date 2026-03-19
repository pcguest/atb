# ATB Dashboard Specification

## Status

Dashboard implementation is complete:
- local `atb view` API server with verification gate and privacy reveal flow
- `/view` dashboard UI with verification banner, timeline, graph, inspector, and stats
- reveal audit logging appended into `bundle.atb`

## Purpose

Build a local-first visual dashboard for `.atb` bundles that is fast to audit and safe by default.

Auditor success bar:
- Integrity status, trace flow, and key event details must be understandable in under 30 seconds.

## CLI Interface

```bash
atb view [--port 8080] [--bundle path/to/file.atb]
```

Supported compatibility forms:
- `atb view run.atb/bundle.atb`
- `atb view --bundle run.atb/bundle.atb --port 9090`

Additional safety flags:
- `--no-open`: do not auto-open browser
- `--log-reveals`: retained for CLI compatibility; reveal auditing is always on in v1.1.0

## Architecture

Chosen flow: **Go API server + Next.js dashboard**.

1. `atb view` resolves and loads bundle.
2. Go verifies hash chain before serving event data.
3. Go serves:
   - `GET /` (viewer shell; tamper page if invalid)
   - `GET /api/v1/verification`
   - `GET /api/v1/bundle/meta`
   - `GET /api/v1/bundle/events?offset=&limit=`
   - `GET /api/v1/bundle/graph`
   - `POST /api/v1/privacy/reveal`
4. Next.js dashboard (`/view`) consumes local API JSON.

Rationale:
- file access and integrity verification stay in one trusted local process
- no cloud dependency required
- pagination and masking happen server-side for performance/privacy

## Security Rules

### Verification Gate

- `bundle.Verify()` runs before event APIs are available.
- If verification fails:
  - root page shows full-width red warning: `TAMPER DETECTED`
  - `GET /api/v1/verification` returns `status=invalid`
  - data endpoints return `403`

### Privacy Defaults

- sensitive fields are masked by default in event payload API responses
- reveal requires explicit `POST /api/v1/privacy/reveal`
- reveal response returns only requested field
- reveal actions are append-logged to `bundle.atb`

## API DTO Summary

- `VerificationResponse`: status, message, bundle_path, chain_length, head_hash
- `BundleMetaResponse`: event_count, type_counts, timestamp bounds, verification summary
- `BundleEventsResponse`: paginated event list with sanitized data
- `BundleGraphResponse`: nodes and edges for trace/span graph rendering
- `PrivacyRevealRequest` / `PrivacyRevealResponse`: single field reveal flow

## UI Components

### Verification Banner
- Green lock style when valid.
- Red warning banner when invalid.
- Invalid state disables interactive panels.

### Timeline View
- virtualized vertical list
- color-coded event families (`llm`, `tool`, `chain`, default)
- default page size from API: 200

### Graph View
- React Flow node/edge model from `/api/v1/bundle/graph`
- parent span and sequence relationships visualized

### Inspector Panel
- selected event detail
- masked fields by default
- `Click to Reveal` triggers reveal endpoint

### Stats Overview
- total events
- event family counts
- verification status

## Performance Requirements

- handle 10k+ events smoothly
- required mechanisms:
  - paginated API
  - timeline virtualization
  - lazy detail rendering

## Error Handling

- missing bundle path/file: clear load error
- parse errors: return explicit error JSON
- port in use: surface actionable server startup error
- verification failure: show tamper-first UX and block data endpoints
