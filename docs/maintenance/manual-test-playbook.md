# Manual ATB/Mortise test playbook

Use this playbook from a source checkout when you want to verify that the
private trunk works end-to-end before extraction, demo, or release work. It
does not require a hosted service. Mortise is optional and runs on loopback.

## 1. Build and run the core ATB checks

```bash
make build
go test ./... -count=1
make check-generated
make test-golden
make test-embed
make test-integration
```

Expected result: all commands exit 0. `make test-golden` must pass across Go,
Python, and TypeScript; that is the guard that bundle hashing and RFC 8785
canonicalisation still agree across implementations.

## 2. Generate a realistic incident bundle

```bash
PYTHONPATH=sdk/python ATB_BIN=./atb python examples/python/agent_incident_demo.py
```

Expected result: `run.atb/agent-incident-demo.atb` is created, verified, and
reported. The demo intentionally records a denied policy decision, an
unapproved privileged action, a failed action, human override evidence, and a
closed session. The workflow outcome is bad; the evidence should still verify.

## 3. Exercise the ATB CLI happy path

```bash
./atb verify --bundle run.atb/agent-incident-demo.atb --profile atb.profile.policy_decision --format json
./atb trust-report run.atb/agent-incident-demo.atb --profile atb.profile.policy_decision --format markdown
./atb incident list --bundle run.atb/agent-incident-demo.atb
./atb incident report --bundle run.atb/agent-incident-demo.atb --session sess-laptop-demo --format markdown
./atb incident export --bundle run.atb/agent-incident-demo.atb --session sess-laptop-demo --out /tmp/atb-incident-evidence.zip
./atb compliance pack --bundle run.atb/agent-incident-demo.atb --profile atb.profile.policy_decision --regime eu-ai-act --out /tmp/atb-eu-ai-act-pack.zip
```

Expected result:

- `verify` reports a valid hash chain and the selected profile result.
- `trust-report` explains chain integrity, CAS/completeness, and residual risk.
- `incident list` shows the captured session and anomaly flags.
- `incident report` renders a session-scoped forensic timeline.
- `incident export` writes a zip containing the bundle, report, manifest, and checksums.
- `compliance pack` writes a deterministic offline pack with profile/CAS and EU AI Act mappings.

## 4. Open the ATB UI surfaces

Build from source first so the UI is embedded:

```bash
cd web
npm ci
npm test
npm run lint
npm run typecheck
npm run build
cd ..
make build
```

Open the single-bundle review UI:

```bash
./atb view --bundle run.atb/agent-incident-demo.atb --profile atb.profile.policy_decision
```

Expected UI: a local, dark, work-focused three-pane trust dashboard.

- Left rail: bundle path, event timeline, selected sequence, and profile/CAS summary.
- Center: role selector, session anomaly strip, trace graph, integrity/stat cards, and role-specific summaries.
- Right rail: event inspector with masked fields and explicit reveal controls.
- Roles: `Engineer` sees raw event data and can reveal masked fields; `Auditor` sees compliance evidence; `Executive` sees the summary view.

Open workspace/session pages only when the viewer is served with workspace
support:

```bash
./atb view --sessions "run.atb/*.atb"
```

Expected UI:

- `/sessions`: actor/session overview, schema coverage, and grouped sessions.
- `/workspace`: read-only index of closed session bundles when served by the ATB Agent; in single-bundle mode it explains that no workspace index is exposed.
- `/`: the documentation/marketing site with hero, code demo, feature list, scope boundary, and footer. It positions ATB as the MIT local evidence core.

## 5. Exercise Mortise locally

Terminal 1:

```bash
export MORTISE_AUTH_TOKEN="$(openssl rand -hex 32)"
cd ~/mortise
go run ./cmd/mortise-ingestd \
  --addr 127.0.0.1:9090 \
  --worm-dir /tmp/atb-mortise-worm \
  --receipt-dir /tmp/atb-mortise-receipts
```

Terminal 2:

```bash
export MORTISE_AUTH_TOKEN="<copy-from-terminal-1>"
curl -fsS -X POST \
  -H "Authorization: Bearer ${MORTISE_AUTH_TOKEN}" \
  --data-binary @run.atb/agent-incident-demo.atb \
  http://127.0.0.1:9090/ingest

curl -fsS \
  -H "Authorization: Bearer ${MORTISE_AUTH_TOKEN}" \
  http://127.0.0.1:9090/receipts

curl -fsS http://127.0.0.1:9090/healthz
curl -fsS http://127.0.0.1:9090/custody/key
```

Expected result:

- `/ingest` returns a signed receipt for the bundle.
- `/receipts` lists retained receipts.
- `/healthz` remains unauthenticated and returns `ok`.
- `/custody/key` publishes the receipt signing public key so receipt holders can verify attestations out of band.

## Current product fit

ATB is in a good private-trunk position for the MIT core: the local bundle
format is frozen, CLI/SDK verification is cross-language tested, incident
forensics and compliance packs are cohesive, and the viewer now gives distinct
engineer/auditor/executive review modes without requiring a hosted service.

Mortise is useful as the private companion layer: custody, WORM/S3 storage,
signed receipts, retention policy, and organisation-scoped API keys. Keep it separate from
the public MIT repo. The public story should remain: ATB proves local evidence
integrity offline; Mortise improves custody and operational assurance when an
operator chooses to run it.
