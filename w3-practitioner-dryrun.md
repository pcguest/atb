# W3 Practitioner Evaluation — Internal Dogfood Dry-Run

**Mode:** INTERNAL DOGFOOD / DRY-RUN

**Participant status:** AGENT-EXECUTED — NO HUMAN PARTICIPANT

**Purpose:** Exercise the documented practitioner path and record executable
behaviour without making claims about human comprehension or usability.

Raw output is retained in `.local/runs/`. The local ledger records command,
exit code, elapsed time, key output, and log path. Results below use
`EXECUTED`, `FAILED`, or `NOT RE-RUN`; they do not represent an observed human
study.

## Job A — Obtain

### A1 Go CLI installation

Command: `go install github.com/pcguest/atb/cmd/atb@v1.16.0`

Result: **EXECUTED — exit 0**

Evidence: `.local/runs/track-b-stranger-install.log`

Observed output: the install completed and the installed CLI reported version
`1.16.0` in the clean-room workflow.

### A2 Source build

Command: `make build` in the clean-room clone

Result: **EXECUTED — exit 0**

Evidence: `.local/runs/track-b-make-build.log`

Observed output: `Built ./atb with embedded viewer`. The build invoked npm before
the Go build; separate probes recorded failure when npm was absent or unusable.

### A3 Python SDK installation

Command: `pip install atb-sdk==1.16.0`

Result: **EXECUTED — exit 0**

Evidence: `.local/runs/track-b-sdk-installs.log`

### A4 TypeScript SDK installation

Command: `npm install @pcguest/atb-sdk@1.16.0`

Result: **EXECUTED — exit 0**

Evidence: `.local/runs/track-b-sdk-installs.log`

### A5 Demo incident bundle creation

Command: `make demo-incident`

Result: **FAILED — exit 2 on both recorded runs**

Evidence: `.local/runs/track-b-demo-incident-1.log` and
`.local/runs/track-b-demo-incident-2.log`

Observed output: the example raised `RuntimeError: expected
tool_without_approval finding was not reported`; the Markdown output contained
the escaped form `tool\_without\_approval`. Both runs recorded the same bundle
SHA-256, so the generated bundle was deterministic even though the harness
assertion failed.

## Job B — Verify

### B1 Intact bundle

Command: `atb verify --profile atb.profile.policy_decision <bundle>`

Result: **EXECUTED — exit 0**

Evidence: `.local/runs/track-b-handoff.log` and
`.local/runs/track-c-outputs.log`

Observed output included `Integrity: PASS`, `Profile: PASS`, and `Verification:
PASS`. These strings describe integrity and profile evaluation of the supplied
bundle; they do not establish completeness or truth.

### B2 Tampered bundles

Procedure: verify copies with a modified, reordered, or removed record

Result: **EXECUTED — each case rejected with exit 2**

Evidence: `.local/runs/track-b-tamper-tests.log`

## Job C — Inspect

### C1 Local viewer start

Command: `atb view <bundle>`

Result: **EXECUTED — server start recorded**

Evidence: `.local/runs/track-b-stranger-atb-view.log`

Observed output: the installed CLI printed a local serving URL. Navigation and
human interpretation were not observed.

### C2 Trust-report output

Result: **NOT RE-RUN**

Evidence available: `.local/runs/track-c-outputs.log` contains captured output,
but no human comprehension result can be inferred from it.

## Job D — Bound the claim

### D1 Trust-model text

Result: **REPOSITORY-OBSERVED**

Evidence: `docs/concepts/trust-model.md` states that hash-chain verification
does not prove capture completeness. No comprehension test was run.

### D2 Integrity failure handling

Result: **EXECUTED**

Evidence: `.local/runs/track-b-tamper-tests.log`

Observed output: tampered inputs reported `Integrity: FAIL`, omitted meaningful
coverage, and returned exit 2.

## Job E — Investigate

### E1 Demo incident workflow

Result: **FAILED — exit 2**

Evidence: `.local/runs/track-b-demo-incident-1.log` and
`.local/runs/track-b-demo-incident-2.log`

The workflow reached incident list and report output, then failed its strict
finding assertion because Markdown escaped underscores.

### E2 Incident commands

Result: **PARTLY EXECUTED**

Evidence: the demo logs contain list and report output. A machine-readable
`--format json` mode exists in the command contract but was not used by these
two recorded demo runs.

## Job F — Detect tampering

Modification, reordering, and record-removal cases were **EXECUTED** and each
was rejected with exit 2. Evidence:
`.local/runs/track-b-tamper-tests.log`.

## Job G — Hand off

Copying the bundle and verifying it from a second local path were **EXECUTED —
exit 0**. Evidence: `.local/runs/track-b-handoff.log`. No separate unfamiliar
practitioner received the bundle, so instruction sufficiency was not observed.

## Job H — Act

Residual-risk and alternate-profile output was captured in
`.local/runs/track-b-act.log`. The log does not contain complete command timing
and exit metadata, so this job is **NOT RE-RUN** for success reporting. No human
selection of a next action was observed.

## Additional automated checks

- Accessibility Cypress spec: **EXECUTED — exit 0, 1/1 passing**
  (`.local/runs/track-b-a11y.log`).
- Investigation Cypress specs: **EXECUTED — exit 0, 3/3 passing**
  (`.local/runs/track-b-e2e.log`).
- Five invalid-input cases: **EXECUTED — each returned exit 1**
  (`.local/runs/track-b-errors.log`).
- JSON pipe check: **EXECUTED — pipeline exit 0 and stderr empty**
  (`.local/runs/track-b-pipeability.log`).
- Documentation command scan: recorded command invocations were accepted by
  the scan (`.local/runs/track-b-command-drift.log`).
- Local Markdown link scan: recorded links resolved
  (`.local/runs/track-b-link-check.log`).

## Empirical debt

A real P01 participant study must still observe acquisition, verification,
claim-bounding, tamper recognition, handoff, and next-action selection by
someone unfamiliar with ATB. The study should record elapsed time, wrong turns,
questions, and the participant's own explanation without treating successful
command execution as evidence of comprehension.
