# ATB viewer specification

## Status

**Single-bundle viewer (`/view/`) — shipped:** local `atb view` API server with verification gate and privacy reveal flow; verification banner, timeline, trace graph, event inspector, stats strip, Profile/CAS panel, and `SessionAnomalies` banner when `--sessions` indexes additional bundles; reveal audit logging appended into `bundle.atb`.

**Cross-bundle session UI — partial:** `GET /api/v1/sessions`, `GET /api/v1/sessions/by-actor`, and `GET /api/v1/schema/status` are implemented server-side. React components `SessionList`, `ActorSessions`, `SchemaStatus`, and `RoleSelector` exist with Vitest coverage but are **not mounted** on any route; `web/app/sessions/page.tsx` is a placeholder. See `docs/maintenance/baseline-handoff.md`.

There is no hosted mode, collaborative workspace, or remote multi-tenant review surface.

## Purpose

Build a local-first review UI for one `.atb` bundle at a time that is
fast to audit and safe by default.

Auditor success bar:
- Integrity status, trace flow, and key event details must be understandable in under 30 seconds.

## CLI interface

```bash
atb view [--port 8080] [--bundle path/to/file.atb] [--profile <id-or-path>] [--sessions <glob-or-dir>]
```

Supported forms:
- `atb view run.atb/bundle.atb`
- `atb view --bundle run.atb/bundle.atb --port 9090`
- `atb view --profile atb.profile.rag_answer`
- `atb view --profile ./profiles/custom.yaml`
- `atb view --bundle ./session.atb --sessions ./sessions/`

`--profile`: optional. Evaluates the bundle against the named built-in profile or a DSL YAML
file at startup. When the chain is intact the profile/CAS summary is served immediately via
`GET /api/v1/bundle/profile`. Without `--profile` the endpoint returns 204 until triggered
via `POST /api/v1/bundle/verify` from the UI.

Additional flags:
- `--no-open`: do not auto-open browser
- `--log-reveals`: retained for CLI compatibility; reveal auditing is always on
- `--sessions`: glob or directory of additional `.atb` bundle paths used to build
  the session index alongside the primary `--bundle`. Example:
  `atb view --bundle ./session.atb --sessions ./sessions/`

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
   - `GET /api/v1/sessions` (session index across indexed bundles; see Session list surface)
   - `GET /api/v1/sessions/by-actor` (sessions grouped by actor; see Session list surface)
   - `GET /api/v1/schema/status` (declared-vs-observed event-type contract health for the loaded bundle; see Contract status surface)
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

## Session list surface

The session list extends the existing `atb view` server (same process, same
default `:8080` listen address). It indexes one or more `.atb` bundles supplied
via `--sessions` (glob or directory) together with the primary `--bundle` path.
All session routes require the same session token as other `/api/v1/*` endpoints.

### Routes

#### `GET /api/v1/sessions`

Returns a JSON object:

```json
{
  "sessions": [
    {
      "session_id": "uuid",
      "actor": {
        "display_name": "Paddy Guest",
        "email": "paddy@example.com"
      },
      "started_at": "2026-05-28T12:00:00Z",
      "closed_at": "2026-05-28T12:05:00Z",
      "exchange_count": 3,
      "inferred_profile": "atb.profile.privileged_tool_action",
      "cas_grade": "B",
      "anomaly_flags": ["tool_without_approval"],
      "bundle_path": "/absolute/path/to/session.atb"
    }
  ]
}
```

Field notes:

- `actor.display_name` and `actor.email` are taken from the first event in the
  session that carries `actor.display_name` / `actor.email` in its payload, or
  from top-level event attribution fields when present.
- `started_at` is the RFC 3339 timestamp of the first event belonging to the
  session (by `session_id` in payload or by bundle-level session markers).
- `closed_at` is the RFC 3339 timestamp of the matching `atb.session.close`
  event when present; omitted when the session has not closed.
- `exchange_count` comes from `atb.session.close` summary when present, else
  counts paired request/response exchanges inferred from session events.
- `inferred_profile` is computed by the profile inference rules below.
- `cas_grade` is the CAS grade from `atb verify` run against the session bundle
  using `inferred_profile`; empty when verify cannot run.
- `anomaly_flags` is an array of anomaly flag identifiers (see below); empty
  when no anomalies apply.
- `bundle_path` is the absolute filesystem path of the source bundle.

Returns `403` when any bundle in the index fails hash-chain verification.
Returns `200` with `"sessions": []` when no session events are found.

#### `GET /api/v1/sessions/by-actor`

Returns a JSON object grouping sessions by `actor.display_name`:

```json
{
  "actors": {
    "Paddy Guest": [ { "...SessionEntry..." } ],
    "api-key:1234": [ { "...SessionEntry..." } ]
  }
}
```

Within each actor group, sessions are sorted by `started_at` descending (most
recent first). Actor keys use `actor.display_name`; when absent, the fallback
display name `api-key:<last-4>` applies.

Returns `403` under the same integrity gate as `GET /api/v1/sessions`.

### Profile inference rules

Profile inference inspects event types present anywhere in the session (same
`session_id` within a bundle). The first matching rule wins, evaluated in this
order:

| Condition (event type present in session) | Inferred profile |
| --- | --- |
| `atb.tool.call` | `atb.profile.privileged_tool_action` |
| `atb.data.export` | `atb.profile.data_export` |
| `atb.policy.decision` | `atb.profile.policy_decision` |
| `atb.human.override` | `atb.profile.human_override` |
| `atb.retrieval.*` (any type with prefix `atb.retrieval.`) | `atb.profile.rag_answer` |
| none of the above | `atb.profile.background_automation` |

The inferred profile is advisory. It signals which built-in obligation template
best matches recorded event types; it does not certify compliance or workflow
completeness.

### Anomaly flag rules

Anomaly flags are computed per session entry:

| Flag | Rule |
| --- | --- |
| `tool_without_approval` | An `atb.tool.call` event exists in the session with no preceding `atb.human.approval` event (the canonical approval event type declared in [`spec-ai-traces.md`](./spec-ai-traces.md)) in the same `session_id` (by event sequence order within the bundle). |
| `unresolved_identity` | `actor.display_name` starts with the prefix `api-key:` |
| `session_not_closed` | The session has no `atb.session.close` event |

Multiple flags may apply to one session. The UI renders each flag independently.

### UI components (session list)

Three viewer components consume these routes (see UI components section):

- **Session list** — virtualised table of all sessions
- **Actor sessions** — expandable groups keyed by `actor.display_name`
- **Anomaly badge** — icon + tooltip per flag

Clicking a session row opens the existing single-bundle viewer for
`bundle_path` (navigate to `/view` with the bundle path parameter).

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
- Collapsible `provability_gaps` section when present: each item includes `gap`, `layer`, `mitigation`, and `closed_when` from the verify report
- "Run verify" button triggers `POST /api/v1/bundle/verify` when no startup report was computed

### Stats overview
- total events
- event family counts
- verification status

### Session list `[planned UI — API shipped]`
- virtualised table fed by `GET /api/v1/sessions`
- columns: actor (`display_name`), `started_at`, model (from session events),
  `exchange_count`, inferred profile badge, CAS grade chip, anomaly flag icons
- row click opens the single-bundle viewer for `bundle_path`

### Actor sessions `[planned UI — API shipped]`
- grouped list fed by `GET /api/v1/sessions/by-actor`
- each actor row expands to show that actor's sessions (sorted by `started_at`
  descending within the group)
- shows an unresolved-identity warning badge when any session in the group
  carries the `unresolved_identity` anomaly flag

### Anomaly badge
- one icon + tooltip per entry in `anomaly_flags`
- `tool_without_approval` → amber warning icon
- `unresolved_identity` → grey person icon
- `session_not_closed` → red clock icon

### Contract status `[planned UI — API shipped]`
- fed by `GET /api/v1/schema/status`
- summary cards: declared types, observed types, total events, incomplete events
  (incomplete count emphasised when non-zero)
- per-type table: event type, criticality, observed count, required fields, and
  a status chip — `undeclared` (red, a producer emitted a type the schema does
  not declare), `N incomplete` (amber, a required field is missing on some
  records, with the missing field names listed), `complete` (green), or
  `not observed` (grey)
- an `undeclared types observed` banner names any rogue types so contract drift
  is visible rather than hidden behind per-session detail

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
