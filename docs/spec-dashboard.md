# ATB viewer specification

## Status

Dashboard implementation is complete:
- local `atb view` API server with verification gate and privacy reveal flow
- `/view` viewer UI with verification banner, timeline, graph, inspector, and stats
- reveal audit logging appended into `bundle.atb`

This is the only supported visual UI in the current release. There is no
hosted mode, collaborative workspace, or multi-bundle review surface.

## Purpose

Build a local-first review UI for one `.atb` bundle at a time that is
fast to audit and safe by default.

Auditor success bar:
- Integrity status, trace flow, and key event details must be understandable in under 30 seconds.

## CLI interface

```bash
atb view [--port 8080] [--bundle path/to/file.atb] [--profile <id-or-path>]
```

Supported forms:
- `atb view run.atb/bundle.atb`
- `atb view --bundle run.atb/bundle.atb --port 9090`
- `atb view --profile atb.profile.rag_answer`
- `atb view --profile ./profiles/custom.yaml`

`--profile`: optional. Evaluates the bundle against the named built-in profile or a DSL YAML
file at startup. When the chain is intact the profile/CAS summary is served immediately via
`GET /api/v1/bundle/profile`. Without `--profile` the endpoint returns 204 until triggered
via `POST /api/v1/bundle/verify` from the UI.

Additional flags:
- `--no-open`: do not auto-open browser
- `--log-reveals`: retained for CLI compatibility; reveal auditing is always on

Builds produced by `go install` from the module proxy do not include the embedded review UI.
Running `atb view` in that case shows a minimal install-guidance page. Build from source to
use the visual interface.

Security note: `atb view` binds to `127.0.0.1` by default. All API endpoints require a
session token generated at startup and delivered to the browser in the URL fragment. Do not
expose the viewer on a non-loopback interface.

## Architecture

Chosen flow: **Go API server + Next.js viewer UI**.

1. `atb view` resolves and loads bundle.
2. Go verifies hash chain before serving event data.
3. Go serves:
   - `GET /` (viewer shell; tamper page if invalid)
   - `GET /api/v1/verification`
   - `GET /api/v1/bundle/meta`
   - `GET /api/v1/bundle/events?offset=&limit=`
   - `GET /api/v1/bundle/graph`
   - `POST /api/v1/privacy/reveal`
   - `GET /api/v1/bundle/profile` (204 if no report; 200 + ProfileReportSummary if computed)
   - `POST /api/v1/bundle/verify` (runs verify with stored profile path; returns fresh ProfileReportSummary)
4. Next.js viewer UI (`/view`) consumes local API JSON.

Rationale:
- file access and integrity verification stay in one trusted local process
- no cloud dependency required
- pagination and masking happen server-side for performance/privacy

## Security rules

### Verification gate

- `bundle.Verify()` runs before event APIs are available.
- If verification fails:
  - root page shows full-width red warning: `TAMPER DETECTED`
  - `GET /api/v1/verification` returns `status=invalid`
  - data endpoints return `403`

### Privacy defaults

- sensitive fields are masked by default in event payload API responses
- reveal requires explicit `POST /api/v1/privacy/reveal`
- reveal response returns only requested field
- reveal actions are append-logged to `bundle.atb`

## API DTO summary

- `VerificationResponse`: status, message, bundle_path, chain_length, head_hash
- `BundleMetaResponse`: event_count, type_counts, timestamp bounds, verification summary
- `BundleEventsResponse`: paginated event list with sanitized data
- `BundleGraphResponse`: nodes and edges for trace/span graph rendering
- `PrivacyRevealRequest` / `PrivacyRevealResponse`: single field reveal flow
- `ProfileReportSummary`: profile_id, pass, chain_valid, anchor_status, cas_score, cas_grade,
  sub_scores, critical_failures (array of `{kind, detail}`), warnings
  - `GET /api/v1/bundle/profile` returns 204 No Content when no report has been computed yet.
  - `POST /api/v1/bundle/verify` runs (or re-runs) verify and returns a fresh summary.
  - `POST /api/v1/bundle/verify` returns 422 if the stored profile path cannot be loaded.
  - Both endpoints return 403 when the bundle fails hash-chain verification (tamper mode).

## UI components

### Verification banner
- Green lock style when valid.
- Red warning banner when invalid.
- Invalid state disables interactive panels.

### Timeline view
- virtualized vertical list
- color-coded event families (`llm`, `tool`, `chain`, default)
- default page size from API: 200

### Graph view
- React Flow node/edge model from `/api/v1/bundle/graph`
- parent span and sequence relationships visualized

### Inspector panel
- selected event detail
- masked fields by default
- `Click to Reveal` triggers reveal endpoint

### Profile/CAS summary panel (when `--profile` supplied or after "Verify" button)
- Profile ID
- Pass/fail badge
- Completeness (CAS) score and grade: labeled "completeness (CAS)", not "compliance"
- `corroboration_bonus` and `effective_score` when a `CorroborationPolicy` is applied; grade derives from `effective_score`
- Chain/anchor status (one line)
- List of critical obligation failures with `kind` and `detail`
- Collapsible warnings section
- "Run verify" button triggers `POST /api/v1/bundle/verify` when no startup report was computed

### Stats overview
- total events
- event family counts
- verification status

## Performance requirements

- handle 10k+ events smoothly
- required mechanisms:
  - paginated API
  - timeline virtualization
  - lazy detail rendering

## Error handling

- missing bundle path/file: clear load error
- manifest-only or zero-event bundle: viewer loads, integrity remains valid, and profile summary stays empty until the user runs verify or selects a profile
- parse errors: return explicit error JSON
- port in use: surface actionable server startup error
- verification failure: show tamper-first UX and block data endpoints
