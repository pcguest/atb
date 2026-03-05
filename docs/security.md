# ATB Security Model

ATB is designed for tamper evidence first and local-first operation.

## Core Security Properties

1. Integrity by default
- Event records are hash chained with SHA-256.
- Verification fails on mutation, reorder, insertion, or deletion.

2. Deterministic canonicalization
- Event payloads are canonicalized using RFC 8785 (JCS) before hashing.
- Golden tests enforce byte-identical output across Go, Python, and TypeScript implementations.

3. Local-first data ownership
- Bundles are local files (`run.atb/*.atb`) by default.
- Core workflow (`init`, `append`, `snapshot`, `verify`, `view`) does not require network access.

4. Zero custom cryptography
- Hashing and cryptographic primitives rely on standard language libraries and audited dependencies.

## Client-Side Security Flow

```mermaid
graph LR
    A["Your App"] --> B["ATB SDK/CLI"]
    B --> C["Client-Side Encryption"]
    C --> D["Encrypted Blob to R2 (Optional)"]
    D --> E["Server Storage (Ciphertext Only)"]
    style C fill:#9f9,stroke:#393
    style E fill:#f9f,stroke:#939
```

## Data Classification

ATB stores user-provided event payloads exactly as supplied.

- Public metadata: event type, sequence number, timestamp fields when present.
- Potentially sensitive: event `data` payloads (prompts, outputs, tool arguments, internal context).
- Secrets should not be embedded in event payloads unless explicitly required by your workflow.

## Threat Model (v1)

- Detect tampering of stored trace files.
- Maintain cross-language hash compatibility.
- Prevent accidental secret leakage through repository hygiene and CI checks.

Out of scope for v1:

- Multi-tenant hosted trust boundaries
- Server-side key custody
- On-device malware compromise

## Security Controls

1. Build and release integrity
- Dependency lock files/checksum manifests in Go, Python, and TypeScript packages.
- CI gating before release workflows.

2. Operational controls
- Secrets in GitHub Actions are stored only in repository secrets.
- No secrets committed to source control.
- Weekly registry health checks and maintenance checklist in `docs/maintenance/`.

3. Incident readiness
- Follow [docs/incident-response.md](./incident-response.md) for triage and containment.
- Report vulnerabilities through [../SECURITY.md](../SECURITY.md).

## SOC2-Oriented Control Mapping (Lightweight)

- Access control: GitHub repository permissions + branch protection.
- Change management: pull requests + CI status checks.
- Audit evidence: immutable git history, CI logs, release tags.
- Encryption/integrity: SHA-256 hash chain verification in all SDKs.

This mapping is a readiness baseline, not a formal SOC2 attestation.
