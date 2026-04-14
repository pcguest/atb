# ATB roadmap

## Stable baseline

v1.5.1 is the current stable release. The following are non-negotiable
invariants that all forward work must preserve:

- **Six schema-locked obligation profiles** — `atb.profile.rag_answer`,
  `atb.profile.data_export`, `atb.profile.privileged_tool_action`,
  `atb.profile.policy_decision`, `atb.profile.human_override`,
  `atb.profile.background_automation`. Profile IDs, required event
  sets, and CAS sub-score semantics are frozen at v1.0 and will not
  change without a major version bump.
- **Local-first, hash-chained bundle guarantee** — SHA-256 hash
  chaining over RFC 8785 canonical JSON is the integrity primitive.
  No core verification path requires a network connection. Remote
  exports are always opt-in and complement, not replace, the local
  guarantee.

## Near-term (v1.6 – v1.8)

### v1.6 — Bundle push and WORM export

**Status: Planned**

- `atb push` exports a sealed bundle to a configurable external target
  (S3-compatible storage or a local WORM path).
- S3 target: `atb push --target s3://bucket/prefix` uses the standard
  AWS credential chain and writes the bundle as a single object with a
  content-addressed key (`sha256:<hash>.atb`).
- WORM locking: ATB sets `x-amz-object-lock-mode: COMPLIANCE` and
  `x-amz-object-lock-retain-until-date` when `--lock-until` is supplied
  and the bucket has Object Lock enabled. ATB does not enforce WORM —
  the bucket policy does.
- Remote verification: `atb verify --remote s3://bucket/sha256:<hash>.atb`
  verifies a remotely stored bundle without downloading it in full
  (streaming verify).

See [`docs/spec/bundle-push.md`](./spec/bundle-push.md) for the full
design-intent specification, including the current manual workaround.

### v1.7 — LangChain native callback integration

**Status: Planned**

- `ATBCallbackHandler` for LangChain (Python) attaches to any LLM,
  Chain, or Agent and emits `ai.llm.call`, `ai.tool.exec`, and
  `ai.chain.run` events automatically.
- Zero-config mode: `ATBCallbackHandler()` with no arguments uses the
  active bundle in the current working directory.
- See [`docs/integrations/langchain.md`](./integrations/langchain.md)
  for the current integration guide.

### v1.8 — Profile DSL v1

**Status: Planned**

- YAML-defined custom profiles: operators can define their own
  obligation profiles without modifying Go code.
- Profile validation tooling: `atb profile validate ./my-profile.yaml`
  checks a custom profile schema for correctness.
- Profile registry: `atb profile list` shows built-in and user-defined
  profiles.

## Medium-term

### SIEM and GRC integration guides

- Structured export guides for Splunk, Elastic, and Datadog: how to
  forward `atb verify --format json` output into a SIEM pipeline.
- GRC mapping docs: how ATB findings map to specific control families
  (EU AI Act, NIST AI RMF, ISO 42001, SOC 2 CC6/CC7).
- See [`docs/integrations/siem-grc.md`](./integrations/siem-grc.md)
  for the current starting point.

### Enterprise hardening

- Multi-bundle correlation: tooling to join bundles from parallel
  workflow branches into a reviewable audit graph.
- Key management workflow docs: expanded guidance on Ed25519 key
  rotation, revocation, and handoff for bundle signing in regulated
  environments. See [`docs/key-management.md`](../docs/key-management.md).
- Performance tuning: verified baseline and tuning guide for very large
  bundles (>100k events).

### Opt-in telemetry

- Disabled by default; no data is collected unless explicitly enabled
  by a flag or environment variable (`ATB_TELEMETRY=1`).
- Scope limited to anonymised, aggregated usage metrics (command
  invocations, profile IDs used, bundle sizes). No event payload data
  is ever included.
- The telemetry design doc and data dictionary will ship before any
  collection is enabled.

## Historical context

For the complete version history and what shipped in each release, see
[CHANGELOG.md](../CHANGELOG.md).

### The versioning reset: v1.x to v0.9.0-beta

You may notice in the [Changelog](../CHANGELOG.md) that ATB previously
carried versions up to `v1.8.1` before resetting to `v0.9.0-beta`.

This reset was an intentional decision to honestly reflect the
pre-production status of the project at that point. While the core
hash-chaining logic was robust, the higher-level specifications
(obligation profiles, CAS scoring, and bundle export formats) were
still being refined through internal pilots and community feedback.

By moving to `v0.9.0-beta`, the intention was to signal that:

1. **Breaking spec changes are possible:** The canonical event taxonomy
   or profile schemas might still shift.
2. **Pilot readiness:** The version was suitable for pilots, internal
   experiments, and portfolio-quality integration, but not yet for
   production systems where API stability is a hard requirement.

### v0.9.x and v1.0 completed work

All of the following shipped by v1.5.0:

- [x] Six-profile verifier with YAML-backed profile templates and CAS
      evaluation where supported.
- [x] `VerifierReport` and `TrustReport` JSON output shapes for
      `atb verify` and `atb trust-report`.
- [x] `ComplianceManifest` export for
      `atb export --format compliance --json`.
- [x] MCP integration guide in `docs/integrations/mcp.md`.
- [x] Python and TypeScript SDK event type constants.
- [x] Performance baseline and benchmark suite.
- [x] Ed25519 bundle signing integration test.
- [x] Native MCP server mode (`atb mcp serve`).
- [x] CAS support extended to all six profiles.
- [x] `spec-v1.0.md` and the profile DSL frozen.
- [x] Encrypted bundle support hardened with versioned PBKDF2-SHA256
      parameters.
- [x] CLI command structure and JSON output formats frozen.
- [x] Governance guidance for CISO and auditor acceptance in
      `docs/compliance/` and `docs/security.md`.
