# Codex execution brief

Purpose: provide implementation-oriented guidance for future Codex sessions and contributors.

## High-level objective

Move ATB from:

- a strong integrity-focused audit engine;

into:

- a developer-first automatic audit-capture toolkit with explicit completeness semantics.

Do not destroy or dilute the integrity model while expanding intake automation.

## Architecture priorities

Preserve:

- append-only bundle semantics;
- RFC 8785 canonicalisation;
- offline verification;
- deterministic hashing behaviour;
- local-first operation;
- explicit provenance metadata;
- bundle portability.

Avoid:

- hidden hosted dependencies;
- unverifiable shortcuts;
- ambiguous provenance;
- silent verification fallback behaviour;
- feature work that bypasses the bundle engine.

## Required implementation streams

### Stream 1: repository stabilisation

Goals:

- align support matrix;
- reduce configuration drift;
- improve maintainer clarity;
- add baseline benchmarks and coverage reporting.

Expected outputs:

- support matrix doc;
- version parity CI checks;
- benchmark suite;
- coverage visibility.

### Stream 2: trust boundary hardening

Goals:

- ensure every supported signer path verifies offline;
- fail closed on unsupported algorithms;
- sanitise sensitive signer diagnostics.

Expected outputs:

- ECDSA-P256 verification support if advertised;
- signer fixture corpus;
- negative verification fixtures;
- hardened remote signer handling.

### Stream 3: audit profile engine

Goals:

- define named capture profiles;
- verify completeness expectations;
- separate integrity from completeness.

Expected outputs:

- profile schema;
- required event definitions;
- profile-aware verification mode;
- missing-event diagnostics.

### Stream 4: automatic intake adapters

Goals:

- minimise instrumentation work;
- make supported workflows auditable by default.

Expected outputs:

Python:

- context manager;
- decorator;
- OpenAI SDK wrapper;
- agent framework adapter.

TypeScript:

- runtime wrapper;
- OpenAI SDK wrapper;
- agent framework adapter.

CLI:

- process capture wrapper;
- stdout/stderr hashing;
- environment redaction support.

### Stream 5: release trust

Goals:

- make release artifacts trustworthy enough for a security-sensitive audit product.

Expected outputs:

- SBOM generation;
- release provenance;
- release verification docs;
- workflow permission review;
- pinned action references.

### Stream 6: user experience and onboarding

Goals:

- reduce cognitive load;
- produce a coherent first-run experience.

Expected outputs:

- quickstart;
- troubleshooting;
- example workflows;
- architecture diagrams;
- terminology consistency.

## Guardrails

### Never imply omniscience

Do not market or document ATB as if it captures all possible behaviour.

Always distinguish:

- recorded evidence;
- inferred evidence;
- imported evidence;
- unverifiable external behaviour.

### Retrospective provenance must remain explicit

Imported histories must not appear identical to live capture.

### Do not bypass verification semantics

Every adapter or import route must converge through the same bundle integrity model.

### Prefer explicit failure

Unknown algorithms, malformed bundles, missing required events, unsupported profile versions, and invalid manifests should fail loudly.

## Recommended PR cadence

Keep PRs small and reviewable.

Suggested order:

1. docs and roadmap
2. support matrix and CI parity
3. signer verification hardening
4. fixture expansion
5. audit profile engine
6. CLI capture improvements
7. Python adapter
8. TypeScript adapter
9. viewer alignment
10. release provenance

## Definition of done

ATB reaches a strong developer-first release state when:

- supported workflows can be captured automatically;
- completeness expectations are machine-verifiable;
- bundles verify offline;
- imports are provenance-aware;
- releases have a documented trust story;
- docs are aligned with reality;
- users understand exactly what ATB proves and what it does not prove.
