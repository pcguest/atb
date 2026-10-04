---
name: security-review
description: "Use when reviewing a change for security, trust-boundary, or hostile-input weaknesses. Trigger on: import/parsing untrusted input, checkpoint files, paths/symlinks, secrets, CLI args, raw-source reveal, model/Jev authority, permission or tool expansion, before a material commit."
---

# Security Review

## Purpose

Attempt to falsify the change under hostile conditions. This is a review
procedure for the adversarial-reviewer agent (a different model family from the
implementer). Output is ADVISORY_ONLY; deterministic gates decide.

## When to invoke

- Before accepting a material change (new parser, importer, file I/O, CLI flag)
- When untrusted input reaches ATB (JSONL, OTLP, chatlog, bundle files)
- When adding viewer/API exposure of raw source or secrets-adjacent data
- When changing permissions, tool access, or model routing

## Bounded procedure

1. Enumerate untrusted inputs and trust boundaries touched.
2. Probe: malicious JSONL; path traversal; symlink source; huge source;
   source-ID collision; same ID / different payload; digest confusion;
   malicious timestamp; checkpoint tampering/staleness; arbitrary local file
   disclosure; raw-source reveal bypass; secret leakage; adapter metadata
   spoofing; CLI argument injection.
3. Probe epistemic boundary: does any path let Jev or a model act as authority?
4. Probe routing: can a read-only agent gain write permission?
5. Classify each finding BLOCKING / NON_BLOCKING / QUESTION with a concrete
   exploit path and impact. Distinguish theoretical from reachable.

## Stop conditions

- Stop and escalate if a reachable exploit writes to a bundle or leaks a secret.
- Stop if the only fix is to weaken an invariant; escalate instead.

## Canonical references

- SECURITY.md
- docs/evidence/key-management.md
- docs/maintainers/security-suppressions.md

## Tests / checks

- `go test ./...` (includes adversarial_test.go in pkg/api/v1)
- `make security-scan`, `make govuln-scan` where available
- Targeted negative tests for the specific weakness

## Prohibited interpretations

- A clean scan is not proof of security.
- Severity must be evidenced, not asserted.
