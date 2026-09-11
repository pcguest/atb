# ATB View

ATB View is the static frontend embedded by the local `atb view` server. It is
a single-bundle forensic investigation surface, not a hosted dashboard.

The primary sequence is Incident → Findings → Timeline → Context →
Relationships → Evidence → Trust. Relationships are list-first and the graph
is optional. Trust keeps integrity, selected-profile coverage, and external
custody separate; it never reduces them to a health score.

## Build and verify

```bash
npm ci
npm run typecheck
npm run lint
npm test
npm run build
npm run test:e2e
npm run test:a11y
```

The app uses a static export (`output: "export"` in `next.config.js`). Do not
add Next.js `headers()` rules: the Go viewer server in `cmd/atb/view.go`
delivers runtime security headers, including CSP. A source build embeds the
contents of `web/out/`, so rebuild the frontend before building the CLI:

```bash
cd web && npm ci && npm run build && cd ..
go build -o atb ./cmd/atb
make test-embed
```

The layout avoids runtime font requests so offline builds and local review are
deterministic. Component and Vitest tests are the isolated state harness;
Cypress exercises the full mocked flow and the embedded server in Firefox.
See the [viewer specification](../docs/specification/viewer.md) and
[visual-system guidance](../docs/maintainers/visual-system.md).
