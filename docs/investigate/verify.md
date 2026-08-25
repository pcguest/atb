# Verify and review evidence

An investigator can review an unknown `.atb` bundle without the producing
application or a hosted verifier.

Start with integrity and the machine-readable verification report:

```bash
atb verify --bundle evidence.atb --format json
atb trust-report --bundle evidence.atb --format markdown
```

Treat a chain failure as a hard integrity failure. A passing chain means the
presented records retain their recorded bytes and order; it does not establish
capture completeness or factual truth.

For incident evidence, discover sessions and follow each deterministic finding
to its record sequence and hash:

```bash
atb incident list --bundle evidence.atb
atb incident report --bundle evidence.atb --session <session-id>
```

If an RFC 3161 token and the required trust roots were retained, verify the
external time evidence too:

```bash
atb verify --bundle evidence.atb --with-anchor --roots tsa-roots.pem
```

Profiles evaluate whether expected evidence is present and related inside the
bundle. CAS is a profile-scoped evidence-coverage signal, not an audit opinion.
Read [`critical_failures`](../specification/verify-report.md), residual risk,
and the [CAS sub-scores](../evidence/cas.md) alongside the capture boundary.

ATB's assurance layers are cumulative:

| Layer | Evidence | What a pass establishes |
| --- | --- | --- |
| Integrity | canonical hash chain | Presented records and ordering verify |
| Workflow shape | profile rules and relations | Expected recorded evidence is present |
| Source provenance | configured signatures | A matching configured key signed the state |
| External evidence | timestamp/corroboration/custody | Separately retained evidence supports time or custody claims |
| Capture boundary | SDK/intercept/import scope | Which activity the recorder was positioned to observe |

The local viewer is useful for exploration, but CLI JSON/NDJSON reports and the
bundle itself are the portable review artefacts. See [incident
investigation](./incidents.md) and the [tampering demonstration](./tampering.md).
