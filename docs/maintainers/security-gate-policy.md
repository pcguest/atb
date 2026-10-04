# Dependency security gate policy

This policy defines the scope of the release-blocking dependency vulnerability
gates: the Node gate (the `node-security` job in
`.github/workflows/security.yml`) and the Python gate (the `python-security`
job). Both are **gate-scope policies**, not suppression mechanisms: they do not
hide, ignore, downgrade, or whitelist any advisory.

It is separate from `docs/maintainers/security-suppressions.md`, which is the
gosec `#nosec` register for Go and is not an npm exception register. Do not add
npm advisories to that file.

## Two signals

The gate runs two signals over `sdk/typescript` and `web`.

Both signals run through `scripts/npm-audit-gate.mjs`, which invokes
`npm audit --json` with an explicitly forced tree scope and classifies the
report.

### Signal A — release-blocking runtime audit (fail-closed)

```
npm audit --json --omit=dev
```

- Any **HIGH or CRITICAL** advisory in the **production/runtime** dependency
  tree fails the job.
- The scope is forced with `--omit=dev` and hostile `NPM_CONFIG_OMIT` /
  `NPM_CONFIG_PRODUCTION` config is cleared, so a committed `.npmrc` cannot
  make the gate skip production advisories.
- A failure to execute the audit also fails the job (see below).

### Signal B — full-tree development visibility

```
npm audit --json --include=dev
```

- The full dependency tree is always audited. CI prints aggregate counts plus,
  for every HIGH/CRITICAL finding, the package name and — where npm reports one
  — the advisory title; lower-severity findings are counted but not individually
  listed. Unresolved development/build advisories therefore remain **visible and
  auditable**.
- The scope is forced with `--include=dev`, which overrides an `omit`/`production`
  config, so the visibility signal cannot be silently reduced to production
  only.
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
  output, missing metadata, or an internally inconsistent report (the
  `vulnerabilities` object reports HIGH/CRITICAL while
  `metadata.vulnerabilities` reports none).

An execution failure exits non-zero and fails the job. It is never reported as
a clean audit or as a harmless development advisory.

## Trust boundary (residual risk)

The gate trusts the installed `npm` and the committed lockfile, as any
dependency audit must:

- a tampered lockfile that mislabels a production dependency as a development
  one can make `--omit=dev` treat it as out of scope;
- a substituted `npm` on `PATH` can report anything.

These are toolchain-integrity and code-review concerns rather than audit
classification concerns. The runtime/container scanner (Trivy) and normal PR
review of `package-lock.json` diffs remain the independent backstops.

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

## Python dependency audit (pip-audit)

The Python gate is **additive** to the Bandit source (SAST) scan and does not
replace or weaken it. It runs `scripts/pip-audit-gate.sh` from the
`python-security` job, using the pinned tooling in
`sdk/python/requirements-security.txt` (`pip-audit==2.10.1`).

It audits three fully-pinned inputs with `--no-deps --strict`:

1. **Release-tooling lock** — `sdk/python/requirements-release.txt`, the
   build/publish toolchain (`build`, `setuptools`, `twine`, `wheel` and their
   closure).
2. **SDK runtime dependency graph** — `sdk/python/requirements-runtime.txt`,
   compiled from `sdk/python/pyproject.toml` (currently `cryptography` and its
   closure).
3. **Security-tooling lock** — `sdk/python/requirements-security.txt`, so the
   audit tool's own pinned closure is itself scanned.

`--no-deps` audits exactly the pinned closure rather than re-resolving from the
network, so the checked set is reproducible. `--strict` fails closed if a
dependency cannot be audited. The gate exits `2` if `pip-audit` is unavailable,
so a missing tool is never reported as a clean audit.

### Interpreter scope

`pip-audit` requires Python `>= 3.10` and evaluates the environment markers in
`-r` inputs against the running interpreter, silently skipping entries whose
markers do not match. The runtime lock is therefore compiled at the audit
interpreter (`--python-version 3.11`) so that **every** entry is audited rather
than a marker-guarded subset. Python 3.9 remains a supported SDK runtime
(`requires-python = ">=3.9"`); its install path and SDK contract are proven by
the dedicated "Python 3.9 Runtime Compatibility" CI job, and its resolved
`cryptography` is the same audited `50.x` release (the `>=3.9.2` marker routing
selects it; `cryptography` itself excludes `3.9.0`/`3.9.1`).

The gate trusts the pinned lockfiles and the pip-audit advisory source, as any
dependency audit must; a tampered lockfile that removes a vulnerable pin is a
code-review concern, not an audit-classification one.

## Verifying locally

```bash
# Node: Web UI
cd web
node ../scripts/npm-audit-gate.mjs runtime     # release-blocking signal
node ../scripts/npm-audit-gate.mjs visibility  # full-tree visibility signal

# Node: TypeScript SDK
cd sdk/typescript
node ../../scripts/npm-audit-gate.mjs runtime
node ../../scripts/npm-audit-gate.mjs visibility

# Python: release tooling + SDK runtime graph (requires the CI-only tooling)
python3 -m pip install -r sdk/python/requirements-security.txt
bash scripts/pip-audit-gate.sh
```
