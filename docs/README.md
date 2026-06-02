# ATB documentation

The canonical documentation index is in the repository [README](../README.md#docs-index).

Start with [quickstart.md](./quickstart.md) for the first-run CLI path, then follow the linked spec, security, and profile guides from the README index.

For where the product is headed and why, see the [vision](./vision.md) (the market-driven end state for ATB and Custos).

Research / design notes (forward-looking; not yet implemented unless stated) live in [research/](./research/):

- [Transparency log](./research/transparency-log.md) — Merkle inclusion proofs + signed checkpoints for Custos custody, and the honest trust model (why a single-operator log needs witnessing before it can claim equivocation resistance).
- [EU AI Act evidence-to-obligation mapping](./research/eu-ai-act-mapping.md) — what ATB/Custos evidence supports against Articles 12, 14, 17–20, and 26(6), with the residual gap stated for each. Not legal advice; not a compliance claim.
- [Auto-capture & Custos scope](./research/capture-and-custos-scope.md) — the SDK auto-capture decision (proxy vs in-process wrapper) and scope guardrails for the Custos `insights`/`oversight`/`onboarding` stubs.

Walkthroughs:

- [Agent incident forensics](./guides/agent-incident-forensics.md) — capture an agent session with `atb intercept`, then discover and review it with `atb incident list` / `atb incident report`.

Maintainers: [CONTRIBUTING.md](../CONTRIBUTING.md), [SECURITY.md](../SECURITY.md), [VERSIONING.md](../VERSIONING.md).
