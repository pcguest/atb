# ATB Web Dashboard Notes

## Static Export and Security Headers

The web app is configured for static export (`output: "export"` in `next.config.js`).
Next.js warns that `headers()` rules are not applied to static exports. This is expected in ATB.

Runtime security headers (including CSP) are enforced by the Go viewer server in `cmd/atb/view.go`.
Source builds of the Go CLI embed whatever is present under `web/out/`, so run a fresh web build before testing `atb view` from a checkout:

```bash
cd web
npm ci
npm run build
cd ..
go build -o atb ./cmd/atb
```

Validate header delivery through the embed flow:

```bash
make test-embed
```

## Font behaviour in CI

The dashboard avoids runtime Google Fonts fetches in layout code to keep CI and offline builds reliable.

## Current Dashboard Readiness

### Completed

- [x] Role-based rendering (Engineer/Auditor/Executive)
- [x] Real-time polling (5s interval)
- [x] Trust Score widget + calculation logic
- [x] Bundle Meta panel with copyable hashes
- [x] Zod schema validation for all API responses
- [x] React Query + Zustand state management
- [x] WCAG 2.1 AA accessibility (0 axe violations in mock-mode E2E)
- [x] E2E tests (4/4 passing with mock API)
- [x] Lighthouse targets documented (A11y >= 100, Perf >= 90)

### Test Commands

```bash
# Unit tests
npm run test

# E2E tests (mock mode for CI)
CYPRESS_MOCK_API=true npm run test:e2e

# Accessibility audit
npm run test:a11y

# Build validation
npm run build
```

See [docs/roadmap.md](../docs/roadmap.md) for tracked follow-on work.
