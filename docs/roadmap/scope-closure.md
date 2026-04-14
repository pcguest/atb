# ATB scope closure — v1.6 and beyond

Working checklist for closing the product scope. This is an internal reference, not marketing copy. Items are phrased as concrete implementation tasks. "Committed" means the behaviour is specified and will ship; "consideration" means it depends on demand or capacity.

Current stable release: v1.5.1. All items below are post-v1.5.1.

---

## Core runtime and CLI

These items close gaps in the local runtime before or alongside v1.6.

- [ ] Expose `atb push` event type in the canonical event registry (`atb events`) once the push command ships, so profile authors and auditors can reference it.
- [ ] Add `atb view` as a stable (non-experimental) command once the embedded UI reaches a formally specified API surface. Currently `--ui-experimental` is excluded from the stable CLI contract.
- [ ] Produce a verified performance baseline for bundles exceeding 100k events and document tuning parameters in `docs/performance.md`.
- [ ] Harden the `atb verify --trace` diagnostic output: currently debug-only stderr; consider a structured flag for machine-readable hash-step output in a future minor.

---

## SDKs and integrations

### LangChain (v1.7 milestone)

- [ ] Publish `ATBCallbackHandler` as a first-class supported integration in the Python SDK with stable import path and version-pinned example.
- [ ] Implement zero-config mode: `ATBCallbackHandler()` with no arguments uses the active bundle in the current working directory.
- [ ] Add integration test covering LangChain ≥ 0.3 callback interface.

### TypeScript SDK and Vercel AI

- [ ] Stabilise `atbMiddleware` API in the TypeScript SDK; it is currently undocumented in the main README.
- [ ] Add integration test for the Vercel AI SDK middleware path.

### MCP bridge

- [ ] Confirm MCP bridge tool schema against MCP protocol version shipped in Claude Desktop and document the version pinning.
- [ ] Add `rag_answer_record` tool as a convenience alongside existing `rag_index_record` and `rag_retrieval_record`, or document the recommended multi-call pattern.

### SIEM and GRC export

- [ ] Write a reference Splunk pipeline (HEC ingestor config + field extractions) for `atb export --format soc2` JSONL output.
- [ ] Write a reference Elastic pipeline (ingest pipeline + index mapping) for the same JSONL output.
- [ ] Write a reference Datadog log forwarding config for the same JSONL output.
- [ ] Document the GRC evidence-locker upload pattern (ZIP + `.verify.json` sidecar) for Vanta and Drata in `docs/integrations/siem-grc.md`.

### Profile DSL (v1.8 milestone)

- [ ] Implement YAML-defined custom profiles: operators define obligation profiles without modifying Go code.
- [ ] Ship `atb profile validate ./my-profile.yaml` to check a custom profile schema for correctness.
- [ ] Ship `atb profile list` showing built-in and user-defined profiles.

---

## Storage and durability

Canonical spec: [`docs/spec/bundle-push.md`](../spec/bundle-push.md).
Integration guide: [`docs/integrations/worm-s3.md`](../integrations/worm-s3.md).

### S3 WORM push (v1.6 — committed)

CLI stub is wired (`cmd/atb/push.go`). Full implementation remaining:

- [ ] Implement S3 upload in `atb push <s3://bucket/prefix>`: open bundle, compute head hash, PUT as `<prefix>/sha256-<head-hash>.atb` using AWS SDK.
- [ ] Set `x-amz-object-lock-mode: COMPLIANCE` and `x-amz-object-lock-retain-until-date: <YYYY-MM-DD>T00:00:00Z` when `--lock-until` is supplied. No-op if bucket does not have Object Lock enabled (surface a warning, not an error).
- [ ] Record `atb.bundle.pushed` event in the local bundle after a successful upload, including target URI and object key.
- [ ] Implement `--dry-run`: validate args, resolve the object key, print the key and lock headers that would be sent; no upload.
- [ ] Implement `atb verify --remote s3://bucket/prefix/sha256-<hash>.atb`: stream-verify a remotely stored bundle without a full download. The hash in the object key is checked against the computed head hash.
- [ ] Write integration tests for the push path covering: successful upload, `--lock-until` header presence, `--dry-run` no-op, credential failure exit code.
- [ ] Register `ATB_WORM_S3_ACCESS_KEY_ID` and `ATB_WORM_S3_SECRET_ACCESS_KEY` in CI before the v1.6 release workflow runs (see `docs/release/secrets.md`). Map to `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in the workflow environment.

### Other WORM-capable targets (post-v1.6 consideration)

- [ ] Evaluate Azure Blob Storage immutable storage as an `az://` target. Depends on demand; spec placeholder in `docs/spec/bundle-push.md`.
- [ ] Evaluate Google Cloud Storage object lock as a `gcs://` target. Depends on demand; spec placeholder in `docs/spec/bundle-push.md`.

### Local backup

- [ ] Document a recommended local backup pattern (e.g., periodic `atb export --format bundle` to a separate encrypted volume) for operators who cannot use cloud WORM.

---

## Observability and telemetry (opt-in only)

- [ ] Design opt-in telemetry: disabled by default, enabled only via `ATB_TELEMETRY=1` or an explicit flag.
- [ ] Scope strictly to anonymised, aggregated usage metrics (command invocations, profile IDs used, bundle sizes). No event payload data.
- [ ] Publish the telemetry design doc and data dictionary before any collection is enabled.

---

## Docs and compliance mapping

- [ ] Add a "Limits" section to every compliance doc that does not already have one, stating: ATB does not ensure recording completeness; it does not prove model correctness or risk controls; it proves integrity of what was recorded.
- [ ] Add reference SIEM pipeline examples to `docs/integrations/siem-grc.md` (see SIEM items above).
- [ ] Expand `docs/compliance/retention.md` to cover WORM retain-until date recommendations once `atb push` ships.
- [ ] Write a key rotation guide in `docs/key-management.md` covering Ed25519 revocation and handoff for regulated environments.
