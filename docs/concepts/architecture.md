# ATB architecture

ATB turns externally observed AI-agent activity into portable evidence that can
be verified and investigated without trusting the system that produced it.
The file boundary, shared semantic contract, and capture limits are the
load-bearing parts of the architecture.

## Evidence flow

```text
AI agent / application / framework
              │
              ▼
           CAPTURE
   ┌──────────┼──────────┐
   │          │          │
  SDK      intercept    import
   │          │          │
   └──────────┼──────────┘
              ▼
       canonical ATB event
              │
              ▼
       RFC 8785 canonicalise
              │
              ▼
         SHA-256 chain
              │
              ▼
          .atb bundle
              │
       ┌──────┼─────────────┐
       ▼      ▼             ▼
    verify  incident      view
              │
              ▼
        evidence pack
              │ optional
              ▼
           Mortise
        custody boundary
```

Signing, RFC 3161 timestamp evidence, and encryption are optional operations
around the bundle. Signing records configured key provenance, timestamping
adds independently verifiable time evidence, and encryption protects
confidentiality. None proves capture completeness.

## Independent implementation model

ATB does not route every SDK call through one shared Go engine. The three
implementations share a semantic contract and prove agreement with deterministic
vectors:

```text
                 shared schemas
                 shared event vocabulary
                 shared golden vectors
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
   Go implementation  Python SDK    TypeScript SDK
          │               │               │
          └──────── byte-compatible ──────┘
                    bundle semantics
```

Go is the reference implementation for the CLI and compatibility review.
Python and TypeScript implement bundle creation, canonicalisation,
verification, and supported cryptographic semantics independently. A failure
in any language's golden-vector tests blocks release across all three.

## Bundle file boundary

An event is semantic activity submitted to ATB. A record is the canonicalised
event plus its SHA-256 chain hash stored as one NDJSON line in a `.atb` bundle.
The previous record hash is part of the next canonical hash input, beginning
with the all-zero genesis value.

Bundle mutations use append-only record semantics. Go writes the resulting
local file via advisory locking, a same-directory temporary file, `fsync`, and
atomic rename. This prevents ordinary concurrent writers and torn writes from
silently corrupting a bundle. It does not make the file immutable: a process
with filesystem control can replace or roll back the entire file before
external anchoring or custody.

Integrity-sensitive readers use `LoadVerified`, which requires a manifest and
verifies the complete chain before returning evidence. Inspection and legacy
compatibility paths may use the deliberately non-validating `Load` parser.

## Capture surfaces

- Go, Python, and TypeScript SDKs append events explicitly.
- `atb capture run` supplies capture context to a child process; it does not
  auto-instrument arbitrary runtimes.
- `atb intercept` observes only provider traffic routed through its loopback
  HTTPS proxy and only configured target hosts.
- `atb import chatlog` maps supported generic JSONL records.
- `atb import otel` decodes the implemented OTLP/JSON subset and maps
  attributable spans through `pkg/otel`.
- OTel attributes outside the recognised semantic map remain attached under
  `otel_attributes`; payload-shaped or credential-like values are retained as
  canonical SHA-256 digests rather than raw content.
- `atb mcp serve` is a beta stdio bridge with a deliberately limited evidence
  tool surface; it does not instrument unrelated MCP servers.

Every surface has blind spots. A capture-scope event can record what an
integration claims it could observe, but neither that claim nor the resulting
bundle proves that all relevant real-world activity passed through the
integration.

## Verification, profiles, and incident findings

`atb verify` recomputes the chain before evaluating optional signature,
timestamp, and profile evidence. Profiles are ATB-defined declarative evidence
obligations. CAS is the Completeness Assurance Score: a local estimate of
profile-scoped evidence coverage within the recorded bundle, not universal
capture completeness or compliance certification.

Incident analysis deterministically derives observations from verified
records. Findings retain record sequence and hash references. For example,
`tool_without_approval` means no matching approval is present in the preceding
recorded evidence; it does not assert that no approval existed outside ATB's
capture boundary.

## Viewer and authentication boundary

`atb view` is a single-user, loopback-only review server. API routes fail
closed and accept a generated session token or explicitly paired OIDC issuer
and audience settings. Viewer authentication protects local API access; it does
not independently validate caller-asserted actor fields stored in evidence.

A `go install` build uses the `noembed` path and serves installation guidance.
The full static review UI is embedded only when built from a checkout after the
web production build.

## External integration and custody boundary

Explicit network integrations include configured provider traffic, TSA
requests, OIDC/JWKS retrieval, corroboration adapters, remote signers, S3
pushes, and optional Mortise endpoints. Core capture, verification, incident
analysis, profile evaluation, packs, and local review require none of them.

Mortise is the optional commercial custody and organisational layer for ATB
evidence. The ATB repository contains an integration client and conformance
surface, not the Mortise runtime. Operator-controlled S3/Object Lock is another
explicit custody option; an accepted upload request is evidence of API
acceptance, not proof of continuing storage enforcement.

## Resource and trust limits

ATB caps individual bundle records, total bundle bytes, record count,
chatlog/OTel imports, proxy bodies, Mortise responses, and TSA responses.
Treat bundles as untrusted input and see the [trust model](./trust-model.md) for
the current hardening classification.

See the [bundle specification](../specification/bundle-v1.md) for byte-level semantics and the
[glossary](./glossary.md) for canonical terminology.
