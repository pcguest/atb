# ATB

[![Release](https://img.shields.io/badge/release-v1.11.0-blue.svg)](CHANGELOG.md) [![Go Reference](https://pkg.go.dev/badge/github.com/pcguest/atb.svg)](https://pkg.go.dev/github.com/pcguest/atb) [![CI](https://github.com/pcguest/atb/actions/workflows/ci.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/ci.yml) [![Security](https://github.com/pcguest/atb/actions/workflows/security.yml/badge.svg)](https://github.com/pcguest/atb/actions/workflows/security.yml) [![Licence](https://img.shields.io/badge/licence-MIT-green.svg)](LICENSE)

ATB is a local-first tamper-evident audit trail for AI and agent workflows. It records events into an append-only, hash-chained bundle on local disk so teams can verify later whether the recorded sequence was altered, without a hosted backend and without routing payload data to external infrastructure by default.

ATB is not an observability platform, a model evaluator, or a compliance certification service. It does not stream events, score model quality, or replace your existing logging pipeline.

Integrity primitive: SHA-256 hash chaining over RFC 8785 canonical JSON, with optional Ed25519 or ECDSA P-256 bundle signatures and optional RFC 3161 TSA anchoring. ATB proves integrity of what was recorded; it does not prove recording completeness, model correctness, or that risk controls were applied.

Current release: [`v1.11.0`](CHANGELOG.md)

## Is ATB right for me?

### ATB is a good fit if

- You need a portable evidence artefact for an incident review, customer handoff, or internal audit.
- You need to prove that recorded AI workflow events were not altered after capture.
- Your team wants local-first review without sending raw traces to a hosted backend by default.
- You can define the workflow evidence you expect and instrument it with the CLI or SDKs.
- You need structured verifier output (`pass`, CAS, `provability_gaps`, profile obligations) you can attach to a review pack.

### ATB is not the right tool if

- You need real-time monitoring, alerting, or a shared debugging dashboard across many services.
- You need model evaluation, benchmarking, prompt management, or broader AI operations tooling.
- You need a SaaS control plane or hosted multi-user workspace.
- You need ATB itself to certify compliance, prove model correctness, or prove capture completeness.
- You mainly need general application performance or distributed tracing.

### Trust model

ATB proves that a bundle was not altered after recording. It does not prove that recording was complete, that every relevant event was captured, or that the bundle file has not been replaced wholesale by an attacker with write access before export. For regulated deployments, pair ATB with filesystem integrity monitoring or export to a WORM-capable store before relying on bundles as primary evidence.

Bundle writes use temp-file, fsync, atomic rename, and advisory locking. Integrity-sensitive paths use `LoadVerified`; `Load` is non-validating for inspection only. See [`docs/security.md`](docs/security.md) for the full threat model.

## Quickstart

Record and verify a support-triage escalation decision:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
atb bundle new
atb append ai.request.received \
  --data '{"request_id":"req-1042","actor_id_hash":"sha256-user-1042","purpose_tag":"support-triage"}'
atb append ai.action.precommit \
  --data '{"action_id":"act-1042","action_type":"route_case","action_parameters_digest":"sha256-route-tier-2","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision \
  --data '{"policy_id":"pol-severity-routing","policy_version":"2026-04","decision":"deny","decision_reason_codes":["sev2_requires_manual_review"],"subject_id_hash":"sha256-user-1042","action_id":"act-1042"}'
atb snapshot incident_review_failed
atb verify --profile atb.profile.policy_decision --format json
```

The `.atb` file holds the hash-chained event sequence. A passing result means the chain is intact and the selected profile's required evidence was found; it does not mean recording was complete.

For concurrent CI writers, set `ATB_LOCK_WAIT=10s` or pass `--lock-wait 10s` on `append`, `snapshot`, `capture run`, or `sign` so lock contention retries before exit code 9.

Inspect locally:

```bash
atb inspect --bundle run.atb/bundle.atb
atb view --bundle run.atb/bundle.atb
```

`atb view` requires a build with the embedded UI; use a [GitHub Releases](https://github.com/pcguest/atb/releases) binary if `go install` serves install guidance only.

Further paths: [`docs/quickstart.md`](docs/quickstart.md), [`docs/guides/capture-quickstart.md`](docs/guides/capture-quickstart.md) (`atb capture run`, `atb import chatlog`), [`docs/guides/instrumentation-checklist.md`](docs/guides/instrumentation-checklist.md), [`docs/guides/agent.md`](docs/guides/agent.md) — **recommended for local desktop usage** (optional Agent + workspace capture).

## ATB Agent (optional)

The **ATB Agent** is an optional local background service for installed
workflows. It is not required: the CLI, SDKs, `atb capture run`, and
`atb view` work without it. For multiple session bundles on one machine,
see [`docs/guides/agent.md`](docs/guides/agent.md) (first-run behaviour and configuration).

Start the Agent:

```bash
atb agent run
```

**Current capabilities:** loopback HTTP only (default `127.0.0.1:6180`):

- `GET /healthz` — readiness check (`{"status":"ok"}`).
- `GET /v1/info` — ATB version, build metadata when available, and config
  summary (`listen_addr`, `data_dir`).
- Capture and workspace index APIs — session open/append/close and
  `GET /v1/workspace/bundles` (read-only bundle list).

Default workspace data directory: `~/.atb/agent` (override with
`ATB_AGENT_DATA_DIR` or `agent.data_dir` in `~/.atb/config.json`).

**Planned (high level):** multi-bundle viewer integration; MCP hosted by the
Agent. See [`docs/guides/agent.md`](docs/guides/agent.md).

## Exit codes

| Constant | Code | Meaning |
| --- | ---: | --- |
| `exitSuccess` | 0 | Command completed successfully. |
| `exitUserError` | 1 | Bad flags, missing local files, invalid input, or another operator-correctable error. |
| `exitIntegrityFailure` | 2 | Bundle integrity verification failed. |
| `exitVerifyFailure` | 3 | Profile or verification policy failed. |
| `exitSystemError` | 3 | Runtime or system failure; shares code `3` for compatibility. |
| `exitLockContention` | 9 | Another writer holds the bundle lock; retry after a short delay. |

## Install

Download the platform binary from [GitHub Releases](https://github.com/pcguest/atb/releases), or:

```bash
go install github.com/pcguest/atb/cmd/atb@latest
```

Requires Go 1.26.3+. Build from source with the embedded viewer:

```bash
cd web && npm ci && npm run build && cd .. && go build -o atb ./cmd/atb
```

`atb view` binds to `127.0.0.1` by default and uses a session token in the URL fragment. Do not expose the viewer on a non-loopback interface. See [SECURITY.md](SECURITY.md).

## SDKs and integrations

| Surface | Entry |
| --- | --- |
| Python SDK | [`sdk/python/README.md`](sdk/python/README.md) — LangChain callback, in-process `Bundle` |
| TypeScript SDK | [`sdk/typescript/README.md`](sdk/typescript/README.md) |
| LangChain | [`docs/integrations/langchain.md`](docs/integrations/langchain.md) |
| Chatlog import | [`docs/integrations/chatlog-import.md`](docs/integrations/chatlog-import.md) |
| MCP stdio bridge | [`docs/integrations/mcp.md`](docs/integrations/mcp.md) |
| ATB Agent (local service) | [`docs/guides/agent.md`](docs/guides/agent.md) — optional; health and info endpoints today |
| WORM / S3 export | [`docs/integrations/worm-s3.md`](docs/integrations/worm-s3.md) |
| Queue push transport | [`docs/integrations/push-transports.md`](docs/integrations/push-transports.md) |
| SIEM / GRC export patterns | [`docs/integrations/siem-grc.md`](docs/integrations/siem-grc.md) |

Six built-in obligation profiles (`atb.profile.*`) define required event sets for concrete workflows. See [`docs/profiles.md`](docs/profiles.md) and [`docs/cas-guide.md`](docs/cas-guide.md) for verifier semantics and CAS interpretation.

## Docs index

- [ATB specification v1.0](docs/spec-v1.0.md)
- [AI trace event specification](docs/spec-ai-traces.md)
- [Architecture](docs/architecture.md)
- [Security model](docs/security.md)
- [Quickstart](docs/quickstart.md)
- [Verification profiles](docs/profiles.md)
- [CAS guide](docs/cas-guide.md)
- [Provability ladder](docs/provability-ladder.md)
- [LangChain integration](docs/integrations/langchain.md)
- [Chatlog import](docs/integrations/chatlog-import.md)
- [MCP integration](docs/integrations/mcp.md)
- [WORM/S3 export](docs/integrations/worm-s3.md)
- [Push transports](docs/integrations/push-transports.md)
- [SIEM and GRC integration](docs/integrations/siem-grc.md)
- [Capture quickstart](docs/guides/capture-quickstart.md)
- [ATB Agent (local service)](docs/guides/agent.md)
- [Incident review workflow](docs/guides/incident-review-workflow.md)
- [Customer handoff workflow](docs/guides/customer-handoff-workflow.md)
- [Compliance export](docs/compliance/export.md)
- [EU AI Act mapping](docs/compliance/eu-ai-act.md)
- [NIST AI RMF mapping](docs/compliance/nist-ai-rmf.md)
- [OpenTelemetry comparison](docs/comparisons/opentelemetry.md)
- [CLI flag reference](docs/config.md)
- [Key management](docs/key-management.md)
- [Roadmap](docs/roadmap.md)
- [Custos development handoff](docs/custos-handoff.md)
- [Changelog](CHANGELOG.md)

## Attribution

ATB uses [PageIndex](https://github.com/VectifyAI/PageIndex) by Vectify AI under the MIT Licence in the Python SDK. See [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES) for the full licence text.

## Licence

MIT. See [LICENSE](LICENSE).
Copyright (c) 2026 Patrick Guest.
