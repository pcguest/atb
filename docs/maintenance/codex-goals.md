# Codex Goals playbook

This document adapts the Codex Goals operating model to ATB.

Goals should be used for:

- repository-wide cleanup;
- trust-boundary hardening;
- benchmark-driven optimisation;
- audit-profile implementation;
- SDK parity work;
- verification fixture expansion;
- release trust improvements;
- documentation alignment;
- automatic audit-capture implementation.

Goals should not be used for one-line edits or isolated cosmetic changes.

## Goal-writing principles for ATB

Every Goal should define:

1. the desired end state;
2. how success is verified;
3. what must not regress;
4. what files or systems are in scope;
5. how Codex should iterate;
6. what counts as blocked.

For ATB specifically, Goals should also:

- preserve integrity semantics;
- preserve explicit provenance boundaries;
- distinguish integrity from completeness;
- avoid introducing unverifiable behaviour;
- avoid overstating automatic capture guarantees.

## Recommended repository-wide cleanup Goal

```text
/goal Turn ATB into a coherent developer-first automatic audit-capture toolkit while preserving the existing integrity engine and trust model. Verify progress through passing tests, updated documentation, verification fixtures, benchmark output, and implementation parity across CLI, SDKs, viewer, and profiles. Preserve append-only bundle semantics, offline verification, RFC 8785 canonicalisation, explicit provenance boundaries, and local-first operation. Use only repository-local files, tests, workflows, fixtures, and generated artifacts. Between iterations, identify the highest-leverage trust, maintainability, or completeness gap remaining and implement the smallest defensible improvement that materially advances the final-state roadmap. If blocked or no valid path remains, stop with the attempted paths, evidence gathered, unresolved blocker, trust implications, and next required input.
```

## Recommended trust-hardening Goal

```text
/goal Ensure every advertised ATB signer and verification path can be verified offline with explicit failure semantics. Verify success through fixture bundles, negative verification tests, CI results, and documented signer expectations. Preserve deterministic hashing behaviour, manifest compatibility, and local-first verification. If any signer path cannot be verified safely, downgrade or document the support claim rather than leaving ambiguous behaviour.
```

## Recommended automatic audit-capture Goal

```text
/goal Implement automatic audit-capture profiles for supported CLI, Python, and TypeScript workflows while preserving explicit provenance semantics. Verify success through profile-aware verification output, fixture bundles, adapter examples, and passing tests. Preserve the distinction between integrity and completeness, and ensure imported or retrospective histories cannot masquerade as live capture. Between iterations, prioritise the workflow that provides the greatest reduction in manual instrumentation work while remaining reviewable and verifiable.
```

## Recommended release trust Goal

```text
/goal Improve ATB release trust posture through provenance, SBOM generation, workflow hardening, and release verification documentation. Verify success through CI outputs, generated provenance artifacts, documented verification instructions, and least-privilege workflow permissions. Preserve reproducibility and local verification paths.
```

## Recommended documentation-alignment Goal

```text
/goal Align ATB documentation, README positioning, SDK guidance, capture semantics, and maintenance docs with the actual current repository capabilities. Verify success through cross-linked docs, consistent terminology, and removal of contradictory or overstated claims. Preserve precision around trust boundaries and audit completeness limitations.
```

## How maintainers should evaluate Goal completion

A Goal is not complete because the repository looks cleaner.

A Goal is complete when:

- evidence exists;
- verification passes;
- trust semantics remain intact;
- documentation matches behaviour;
- unresolved uncertainty is explicitly labelled.

For ATB, the final audit surface matters as much as the implementation itself.
