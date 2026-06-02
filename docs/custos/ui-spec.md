# Custos UI/UX Specification

> **Status (June 2026):** This spec describes the target Custos product UI. The in-repo
> `custos/` module ships ingest, receipts, attestation verify, and auth — not these views.
> Do not treat this document as shipped behaviour. See `docs/custos-handoff.md` and
> `docs/public-surface.md`.

## Onboarding / Setup Wizard

- Purpose: collect organisation setup, team contact mode, AI service access, allowed tool list, and automatic signing policy.
- Key fields: org name; work or personal email mode per team; Claude, OpenAI, Gemini, GitHub, and custom MCP API keys; accepted AI applications and websites.
- Signing fields: local key file path or KMS reference, rotation schedule, and TSA anchoring requirement.
- Primary actions: save organisation profile, add or remove service keys, edit accepted tools, validate signing configuration, complete setup.
- Human touchpoint: org admin confirms tool allow-list and signing policy; no manual bundle-signing action exists.

## AI Tool Discovery View

- Purpose: show authorised network scan results and decide which discovered AI tools may produce ingested evidence.
- Key fields: tool name, vendor, kind, endpoint, first-seen timestamp, authorisation state, and review flag.
- Primary actions: authorise, ignore, flag for review, bulk authorise by vendor, refresh scan results.
- Human touchpoint: operators authorise, ignore, or flag tools; signing starts automatically once a tool is authorised and capture occurs.

## Active Sessions Dashboard

- Purpose: provide a live operational view of AI sessions by user, team, tool, signing state, and review state.
- Key fields: user, team, bundle ID, tool used, event count, CAS score, signing status, and human review status.
- Signing status values: auto-signed, unsigned, or key missing.
- Review status values: pending, approved, or rejected.
- Primary actions: filter by team, date, tool, or signing status; open bundle details; open related review item.
- Human touchpoint: users triage sessions with pending review or missing signing configuration.

## Human Review Queue

- Purpose: handle post-facto review of flagged events after bundle signing and ingestion.
- Key fields: review summary, bundle ID link, event ID, tool name, CAS score, auto-sign status, SLA countdown, and assignee.
- Primary actions: approve, reject, enter mandatory reason text of at least 10 characters, escalate to org admin, open resolved archive.
- Human touchpoint: reviewers decide whether flagged evidence is accepted, rejected, or escalated; they do not sign bundles.

## Auditable Work Tree

- Purpose: show per-bundle lineage from events through corroboration links to handoff points.
- Key fields: event nodes, corroboration sources, handoff points, pitfall markers, signing status per node, and TSA anchor status.
- Signing colours: green for auto-signed and TSA anchored, amber for signed only, red for unsigned.
- Primary actions: inspect node detail, follow corroboration link, open handoff record, export `verify.report.v1` JSON.
- Human touchpoint: reviewers inspect highlighted pitfalls and use exports as evidence records.

## Insights & Takeaways

- Purpose: summarise recurring workflow issues and evidence quality by team and tool.
- Key fields: pitfall frequency by team, top workflow patterns by tool, unsigned event count per team, suggested remediations, and date range.
- Primary actions: filter by team or tool, inspect pitfall examples, export PDF evidence pack, export JSON evidence pack.
- Human touchpoint: team leads review repeated pitfalls, assign remediation, and track unsigned-event risk.

## Organisation & Team Management

- Purpose: manage users, team membership, tool permissions, signing references, retention policy, and push destinations.
- Key fields: user list, team IDs, per-team AI tool allow-list, per-team auto-signing key reference, retention duration, and bundle push destination.
- Retention policy: EU AI Act Article 12 minimum is enforced; admins cannot set a value below the minimum.
- Push destination values: S3 or WORM target.
- Primary actions: add or remove users, assign teams, edit allow-lists, update signing key reference, update retention, configure push destination.
- Human touchpoint: org admins maintain team permissions, signing key references, and retention controls.
