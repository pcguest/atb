# Tenon pilot walkthrough

This walkthrough is the local, credential-free pilot path for ATB. It uses a
synthetic fixture so evaluators can test the evidence loop without contacting a
model provider, executing tools, or submitting to custody by default.

The scenario: a support agent considers two privileged actions in one session.
One destructive action (`delete_user_records`) is attempted without approval and
fails. A separate action (`issue_store_credit`) is approved and completes.

## 1. Build the complete CLI

```bash
make build
```

The generated fixture and reports work with a normal CLI build. The embedded
viewer requires the source build above.

## 2. Generate the synthetic session

```bash
go run ./examples/bundles/tenon-pilot/
```

This writes `examples/bundles/tenon-pilot/tenon-pilot.atb`. The fixture is
deterministic and records stable timestamps, IDs, and digests for CI and
review repeatability.

## 3. Verify locally

```bash
./atb verify \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --profile atb.profile.privileged_tool_action \
  --format json
```

Expected result:

- integrity chain valid;
- privileged-tool profile passes for the approved action path;
- CAS/profile output is present;
- no vendor account or custody service is required.

## 4. Investigate the incident

```bash
./atb incident list \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb

./atb incident report \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --session sess-tenon-pilot-0001
```

Expected result:

- the session is closed;
- `tool_without_approval` is present for `delete_user_records`;
- `action_failed` is present for the refused destructive action;
- the approved `issue_store_credit` action is visible after approval;
- every report row is bound to the authoritative bundle by sequence and record hash.

## 5. Inspect in the local viewer

```bash
./atb view \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --profile atb.profile.privileged_tool_action
```

The viewer stays local. It must not be exposed outside loopback unless the
operator has separately assessed the risk.

## 6. Optional Mortise custody handoff

Only run this step when a compatible Mortise instance is explicitly available:

```bash
ATB_MORTISE_TOKEN=<token> \
./atb incident export \
  --bundle examples/bundles/tenon-pilot/tenon-pilot.atb \
  --session sess-tenon-pilot-0001 \
  --mortise-endpoint http://127.0.0.1:8088
```

Without `--mortise-endpoint`, all evidence remains local. With a Mortise
endpoint, ATB submits the authoritative bundle bytes and prints the custody
receipt returned by Mortise.

## Honest limits

This is synthetic demonstration evidence. It does not prove live capture
completeness, real operator identity, model correctness, or legal compliance.
It demonstrates the product path: create representative evidence, verify it
offline, investigate an anomalous privileged action, and optionally hand the
authoritative bundle to custody.
