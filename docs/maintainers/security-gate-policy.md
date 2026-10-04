# Node dependency security gate policy

This policy defines the scope of the release-blocking Node dependency
vulnerability gate (the `node-security` job in
`.github/workflows/security.yml`). It is a **gate-scope policy**, not a
suppression mechanism: it does not hide, ignore, downgrade, or whitelist any
advisory.

It is separate from `docs/maintainers/security-suppressions.md`, which is the
gosec `#nosec` register for Go and is not an npm exception register. Do not add
npm advisories to that file.

## Two signals

The gate runs two signals over `sdk/typescript` and `web`.

### Signal A — release-blocking runtime audit (fail-closed)

```
npm audit --omit=dev --audit-level=high
```

- Any **HIGH or CRITICAL** advisory in the **production/runtime** dependency
  tree fails the job.
- The command is not wrapped in `|| true`; its exit status is preserved.
- A failure to execute the audit also fails the job (see below).

### Signal B — full-tree development visibility

```
npm audit --audit-level=high     # full dependency tree, including dev/build
```

- The full dependency tree is always audited and its findings are printed in
  CI, so unresolved development/build advisories remain **visible and
  auditable**.
- Development/build HIGH/CRITICAL advisories are **not** automatically
  release-blocking, but only when **all** of the following hold:
  - the affected package is not present in the production dependency tree;
  - there is no runtime reachability;
  - the runtime audit (Signal A) passes;
  - the runtime/container scanner (Trivy) passes;
  - no patched upstream version exists, **or** remediation would require an
    unrelated high-risk toolchain migration;
  - the advisory remains tracked.

## Development advisories are not harmless

A development/build dependency is not "safe" because it is not shipped.
Development advisories can still affect:

- developer workstations;
- CI runners and build systems;
- generated artefacts and caches;
- supply-chain exposure through tooling and lockfiles.

Signal B exists precisely so these remain visible and can be scheduled for
remediation. "Not release-blocking" does not mean "accepted as risk-free".

## Execution failure is never success

`npm audit` exits with code `1` both when advisories are found and when the
audit itself cannot run (for example `ENOLOCK`, registry/network failure, or a
tool error). The gate therefore does not classify on exit code alone. It
inspects the `--json` report and distinguishes:

- `AUDIT_COMPLETED_WITH_ADVISORIES` — a well-formed report with
  `metadata.vulnerabilities`; classified by signal;
- `AUDIT_EXECUTION_FAILURE` — a report with an `error` object, unparseable
  output, or missing metadata.

An execution failure exits non-zero and fails the job. It is never reported as
a clean audit or as a harmless development advisory.

The gate is implemented in `scripts/npm-audit-gate.mjs`. Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | audit completed; no blocking condition |
| `1` | audit completed; runtime HIGH/CRITICAL found (release-blocking) |
| `2` | audit execution failure |

## Current tracked development advisories

As of this policy's introduction, the remaining HIGH/CRITICAL findings are
development/build-only:

- `GHSA-vfj7-8cjw-p6xm` — `braces <= 3.0.3`, stack-exhaustion DoS. Latest
  published `braces` is `3.0.3`; there is no patched release. Reached only
  through development tooling (`tailwindcss` → `chokidar`/`micromatch` →
  `braces`, and `eslint-config-next` → `@next/eslint-plugin-next` →
  `fast-glob` → `micromatch` → `braces`).

The production/runtime tree is clean (`npm audit --omit=dev
--audit-level=high` reports zero vulnerabilities). This advisory must be
re-checked whenever `braces` is published or the toolchain is upgraded; if a
patched release appears, upgrade through the supported dependency chain and do
not rely on this policy.

## Verifying locally

```bash
cd web   # or sdk/typescript
node ../scripts/npm-audit-gate.mjs runtime     # release-blocking signal
node ../scripts/npm-audit-gate.mjs visibility  # full-tree visibility signal
```
