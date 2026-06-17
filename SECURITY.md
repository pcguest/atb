# Security Policy

This file covers vulnerability reporting and supported release policy. For the product threat
model and control boundaries, see [docs/security.md](docs/security.md).

## Supported versions

Security fixes are shipped on the current release tag only.

| Version | Supported |
| --- | --- |
| `v1.15.0` | Yes |
| `v1.14.5` and older | No |

## Reporting a vulnerability

Report security issues by email to **patrickcguest@proton.me** with the subject line
`[ATB Security] <brief description>`. Do not open a public GitHub issue for an unpatched
vulnerability.

Include:

- A clear description of the issue and its impact.
- Reproduction steps or proof of concept.
- Affected versions, commits, and environment details.

**Disclosure timeline:**

- Initial acknowledgment: within 5 business days of receipt.
- Triage and remediation plan: after confirming reproducibility.
- Coordinated disclosure: after a fix is available and tested.

## Scope

This policy applies to:

- Go CLI (`cmd/`, `internal/`)
- SDKs (`sdk/python`, `sdk/typescript`)
- Local viewer UI (`web/`, `pkg/api/v1/`)
- CI/CD and release workflows (`.github/workflows/`)

Trivy vulnerability scanning runs on a weekly schedule via GitHub Actions.

For incident handling after report intake, see [docs/incident-response.md](docs/incident-response.md).

## Local viewer threat model

`atb view` starts a short-lived HTTP server. Its intended use is single-operator, single-machine
bundle inspection. The threat model is narrow by design.

**What is protected:**

- The server binds to `127.0.0.1` (loopback only) by default. It does not accept connections
  from other machines on the network unless `--host` is overridden. Do not override `--host` to
  expose the viewer on a network interface.
- All read API endpoints (`/api/v1/*`) require a session token (`X-ATB-Session-Token` header or
  `?session_token=` query parameter). The session token is a 32-byte CSPRNG value generated at
  startup. It is delivered to the browser in the URL fragment (`#session=<token>`) so it is never
  automatically forwarded to the server by the browser. For `--no-open` usage, the full URL
  (including fragment) is printed to stdout.
- Privacy reveal operations require a separate, independently generated reveal token delivered
  via cookie, bearer header, or legacy header. Reveals are rate-limited (10 per minute per
  source address by default) and every successful reveal is appended as an auditable
  `privacy.reveal` event to the bundle chain.
- Bundle data is never served when hash-chain verification fails. The tamper warning page
  contains no event payloads.
- HTTP response headers include `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, a strict `Content-Security-Policy`, and
  `Cross-Origin-Resource-Policy: same-origin`. Inline scripts and styles are required by the
  embedded Next.js viewer UI and are constrained with `'unsafe-inline'` within an otherwise
  restrictive policy.

**What is not protected:**

- An attacker with local filesystem write access can replace the bundle file wholesale before
  `atb view` loads it. ATB detects tampering within a loaded chain but cannot detect pre-load
  replacement. Export to WORM-capable storage before relying on bundles as primary evidence in
  adversarial environments.
- The session token is printed to the terminal and included in the browser URL bar. Anyone
  with read access to the terminal session or browser history can see it. This is acceptable for
  the intended single-operator local inspection use case.
- Bundle data is only served via the session-gated JSON API (`/api/v1/*`). The embedded
  Next.js viewer UI at `/view/` reads data exclusively through those API calls. There is no
  server-rendered HTML path that exposes event data outside the session-token model.
- No TLS. The viewer uses plain HTTP because TLS would require certificate management on
  loopback, which adds friction without meaningful security benefit for a local-only service.
  Do not reverse-proxy or expose the viewer over a network.

## Session token model

- Generated: one 32-byte CSPRNG value (hex-encoded, 64 characters) at server startup.
- Override: `--session-token <hex>` supplies a fixed token (useful for scripting or
  automated testing).
- Delivery: appended as a URL fragment (`#session=<token>`) to the startup URL printed to stdout
  and used to open the browser. The fragment is never sent in HTTP requests by the browser. When
  the embedded review UI is available the URL is `http://127.0.0.1:<port>/view/#session=<token>`;
  for builds without the embedded review UI it is `http://127.0.0.1:<port>/#session=<token>`.
- Enforcement: all `/api/v1/` endpoints return HTTP 401 if the token is missing or incorrect.
  Comparison uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.
- Scope: protects the current viewer session on the local machine. Does not persist across
  restarts — a new token is generated each time `atb view` is invoked.

## Reveal token model

- Generated: one 32-byte CSPRNG value at server startup, independently of the session token.
- Delivery: set as an `HttpOnly; SameSite=Strict` cookie (`atb_reveal_token`). Accepted
  additionally via `X-ATB-Viewer-Token` header, `X-ATB-Reveal-Token` header (legacy), or
  `Authorization: Bearer <token>`.
- Enforcement: present only if the session token check passes first. Returns HTTP 401 on
  mismatch.
- Rate limiting: 10 reveals per minute per source address + token combination. Returns
  HTTP 429 with `Retry-After: 60` when exceeded.
- Audit: every reveal is appended to the bundle chain as a `privacy.reveal` event before
  the revealed value is returned. If the append fails, the reveal is aborted with HTTP 500.

## MCP and `atb serve` exposure

`atb mcp serve` and `atb serve` (if present) expose a different surface to the local process
or stdin/stdout bridge. They are not the same as the viewer server and have separate
exposure characteristics. Review the relevant command documentation before exposing these
to external processes or network connections.

## Safe harbour

Good-faith security research is authorised when you avoid privacy violations, data destruction,
service disruption, and social engineering. Scope is limited to issues reproducible against
your own ATB installation.
