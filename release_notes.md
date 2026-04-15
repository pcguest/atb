### Pre-launch surface polish

- `atb view` now accepts `--profile <id-or-path>`: evaluates the bundle against the named built-in profile or a DSL YAML file at startup and exposes the result via `GET /api/v1/bundle/profile`. Without `--profile`, a "Run verify" button triggers `POST /api/v1/bundle/verify` to compute and cache a fresh `ProfileReportSummary`.
- Added `GET /api/v1/bundle/profile` (204 No Content when no report; 200 + `ProfileReportSummary` when computed) and `POST /api/v1/bundle/verify` to the dashboard API server.
- Rewrote README above-the-fold: explicit Why ATB bullets, What ATB does not do sub-section, `atb push` WORM export added to multi-surface list, demo asset hooks.
- Updated `docs/spec-dashboard.md` with new API routes, `--profile` CLI interface, and Profile/CAS summary panel spec; updated `docs/quickstart.md` section 4 to cover `--profile` usage and the verify report screenshot reference.
- Added `docs/launch/assets/` canonical path table to `docs/launch/README.md` with regeneration reminders for demo GIF and verify-report screenshot.

### Key Features in v1.6.0
- **Bundle push / WORM**: Export sealed bundles to S3-compatible storage with optional compliance locking.
- **Profile DSL v1**: Define custom obligation profiles using YAML.
- **atb view UI polish**: Enhanced dashboard with CAS and profile summary integration.
