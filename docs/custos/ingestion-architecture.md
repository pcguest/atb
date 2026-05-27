# Custos Ingestion Architecture Map

| Component | Existing path | Status |
| --- | --- | --- |
| Inbound event receipt from AI tools (agents, LLMs, MCP, RAG) | `internal/agent.Server` capture handlers; `pkg/otel.Receiver`; `cmd/atb capture/import` | partial reuse |
| OTel span/trace -> ATB event translation | `pkg/otel.Translator`, `pkg/otel.DefaultTranslator`, `pkg/otel.Receiver` | scaffold |
| Bundle write + hash-chain append | `internal/bundle.Bundle.AppendWithOptions`, `internal/hash.Compute` | existing |
| Ed25519 signing | `internal/bundle.SignToWithSigner`, `internal/signer.NewLocalSigner` | existing |
| AES-256-GCM encryption at rest | `internal/encrypt.Encrypt`, `internal/encrypt.Decrypt` | existing |
| WORM / push to object store | `internal/push.S3Pusher`, `internal/push.S3Uploader`, `cmd/atb push` | existing |
| Corroboration (external verification of events) | `pkg/corroborate/github.Corroborator`, `pkg/corroborate/langchain.Corroborator`, `internal/corroboration` | scaffold/partial |
| Human-in-the-loop approval gating | net-new | net-new |
| Organisation / team segmentation (multi-tenancy) | `internal/bundle.AppendOptions` has `OrgID` and `WorkspaceID`; no tenancy model | partial |
| AI application / website recognition (known-tool registry) | net-new | net-new |
| Onboarding flow (email, API key provisioning) | net-new | net-new |
| Auditable work tree (handoff + lineage display) | `pkg/api/v1.BundleGraphResponse`, `pkg/api/v1.EventRecordDTO`; no handoff lineage product model | partial |
| Insight extraction (pitfall detection, workflow takeaways) | net-new | net-new |
| UI / dashboard layer | `web/` viewer and `pkg/api/v1.APIServer`; no Custos dashboard | partial |
