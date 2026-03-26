# ATB Web Dashboard Notes

## Static Export and Security Headers

The web app is configured for static export (`output: "export"` in `next.config.js`).
Next.js warns that `headers()` rules are not applied to static exports. This is expected in ATB.

Runtime security headers (including CSP) are enforced by the Go viewer server in `cmd/atb/view.go`.
Validate header delivery through the embed flow:

```bash
make test-embed
```

## Font Behavior in CI

The dashboard avoids runtime Google Fonts fetches in layout code to keep CI and offline builds reliable.

## v1.1.0 Gold Release Readiness

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

# Build + export validation
npm run build && npm run export
```

See [docs/roadmap/web-testing.md](../docs/roadmap/web-testing.md) for planned web testing improvements.
