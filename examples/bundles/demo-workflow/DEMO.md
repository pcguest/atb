# Demo narration script (~5 minutes)

Run from the repository root unless noted. Requires a pre-built `./atb` with embedded viewer (`make build` once).

---

## 1. Setup (15 s)

> "This is a support escalation workflow — an AI agent handles a refund request, policy blocks the auto-refund, a supervisor approves store credit instead, and every step is recorded locally in a tamper-evident audit trail."

---

## 2. Capture story (45 s)

Regenerate the bundle (optional — committed copy works):

```bash
cd examples/bundles/demo-workflow
./generate.sh
cd ../..
```

Show bundle size and event count:

```bash
wc -c examples/bundles/demo-workflow/demo-workflow.atb
wc -l examples/bundles/demo-workflow/demo-workflow.atb
```

**Expected:**

```
    8844 examples/bundles/demo-workflow/demo-workflow.atb
      22 examples/bundles/demo-workflow/demo-workflow.atb
```

> "Twenty events captured locally — no backend, no cloud — in an append-only hash chain. The whole bundle is under nine kilobytes."

---

## 3. Verify moment (60 s)

```bash
./atb verify --bundle examples/bundles/demo-workflow/demo-workflow.atb
```

No `--profile` flag is needed: the bundle declares its workflow class
(`policy_decision`) via the `purpose_tag` on `ai.request.received`, so `verify`
auto-selects the right obligation profile.

**Expected (excerpt):**

```
Profile: atb.profile.policy_decision (Policy Decision)

Summary

Integrity: PASS
Profile:   PASS
CAS:       0.83 (Medium)
...
Verification: PASS
```

Point out:

- **Integrity PASS** — hash chain intact
- **CAS 0.83 (Medium)** — completeness score with XC corroboration credit
- **Obligations** — profile requirements met (warnings on optional L3 signatures are OK)

---

## 4. Tamper moment (90 s) — the kicker

```bash
cd examples/bundles/demo-workflow
cp demo-workflow.atb demo-tampered.atb
./tamper.sh demo-workflow.atb demo-tampered.atb
cd ../..
./atb verify --bundle examples/bundles/demo-workflow/demo-tampered.atb
```

**Expected (excerpt):**

```
Summary

Integrity: FAIL
Profile:   FAIL
CAS:       0.00 (Insufficient)

Issues
✗ bundle: verify: tamper detected at event 11 (seq 11): expected <hash>…, got <hash>…: bundle: tampered
...
Verification: FAIL
```

> "One byte changed in a middle event. The stored hash no longer matches the recomputed hash — tamper detected at sequence 11. This is cryptographic proof the record was altered after capture."

---

## 5. Viewer story (90 s)

**Green state** (valid bundle):

```bash
./atb view --bundle examples/bundles/demo-workflow/demo-workflow.atb \
  --profile atb.profile.policy_decision --port 18890
```

Open the URL printed on stderr (e.g. `http://127.0.0.1:18890/view/#session=…`).

Walk through:

1. **Verification banner** — green chain, profile PASS
2. **Stats** — 22 events, trust score
3. **Timeline** — click `ai.policy.decision`, `ai.human.approval`, `atb.corroboration.external`
4. **Inspector** — event payload (do not click Reveal on the committed bundle)
5. **Profile & CAS** — expand panel, CAS breakdown, obligations

**Tampered state:**

```bash
./atb view --bundle examples/bundles/demo-workflow/demo-tampered.atb \
  --profile atb.profile.policy_decision --port 18891
```

**Expected:** Red **TAMPER DETECTED — HASH CHAIN VERIFICATION FAILED** page. Event data is blocked by design when integrity fails.

Compare with `screenshots/dashboard-overview.png` (green) vs `screenshots/dashboard-tampered.png` (red).

---

## 6. Article 12 close (15 s)

> "Under the EU AI Act, high-risk AI systems need technical logs that support post-market monitoring and incident investigation — Article 12 enforcement begins August 2026. ATB gives you a local-first, tamper-evident chain you can verify independently, without trusting the vendor's dashboard."

---

## Quick reference

| Artifact | Path |
|----------|------|
| Bundle | `examples/bundles/demo-workflow/demo-workflow.atb` |
| Tamper helper | `examples/bundles/demo-workflow/tamper.sh` |
| Green screenshot | `examples/bundles/demo-workflow/screenshots/dashboard-overview.png` |
| Tampered screenshot | `examples/bundles/demo-workflow/screenshots/dashboard-tampered.png` |
| SDK demo scripts | `examples/demo/profile_workflows_demo.{py,ts}` |

**Caveat:** The viewer's privacy Reveal feature writes `privacy.reveal` events
to a separate `<bundle>.reveals` sidecar. The authoritative demo bundle is not
mutated, but the sidecar may contain sensitive reveal audit metadata.
