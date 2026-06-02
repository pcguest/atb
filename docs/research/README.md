# Research & design notes

Forward-looking research and design notes. These are **not** product claims or
shipped features unless a note says otherwise — they exist to think a problem
through (and, for the cryptographic and regulatory items, to lock the trust model
and the honesty bounds) *before* any code is written.

- [transparency-log.md](./transparency-log.md) — Custos transparency log: Merkle
  inclusion proofs, signed checkpoints (C2SP note format), and the honest trust
  model — why a single-operator log needs witness cosignatures before it can claim
  resistance to equivocation (split-view). Grounded in RFC 6962 and the
  Sigstore/Rekor + C2SP witnessing model.
- [eu-ai-act-mapping.md](./eu-ai-act-mapping.md) — a defensible mapping of what
  ATB/Custos evidence supports against EU AI Act Articles 12, 14, 17–20, and 26(6),
  with the residual gap stated for each. Not legal advice; not a compliance claim.
- [capture-and-custos-scope.md](./capture-and-custos-scope.md) — the SDK
  auto-capture decision (universal proxy vs opt-in in-process wrapper) and scope
  guardrails for the Custos `insights` / `oversight` / `onboarding` stubs, so they
  do not drift into generative judgement or hosted multi-tenant workflow.
