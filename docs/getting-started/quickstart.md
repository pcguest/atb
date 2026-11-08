# ATB quickstart

This guide is for a developer who wants to record and verify an AI workflow decision for the
first time. After installation, the record-and-verify cycle runs locally without a
backend or network access. Installation and dependency downloads require a
network connection. At the end you will have a verified `.atb` bundle on disk
and a working understanding of the record-and-verify cycle.

## 1. Your first bundle in 5 minutes

```bash
atb_install_dir="$HOME/.local/bin"
GOBIN="$atb_install_dir" go install github.com/pcguest/atb/v2/cmd/atb@latest
export PATH="$atb_install_dir:$PATH"
atb bundle new
atb append ai.request.received --data='{"request_id":"req-1042","actor_id_hash":"sha256-user-1042","purpose_tag":"support-triage"}'
atb append ai.action.precommit --data='{"action_id":"act-1042","action_type":"route_case","action_parameters_digest":"sha256-route-tier-2","target_resource_id":"support-queue","intended_effect":"escalate_to_manual_review"}'
atb append ai.policy.decision --data='{"policy_id":"pol-severity-routing","policy_version":"2026-04","decision":"deny","decision_reason_codes":["sev2_requires_manual_review"],"subject_id_hash":"sha256-user-1042","action_id":"act-1042"}'
atb snapshot incident_review_failed
atb verify --profile atb.profile.policy_decision --format json
atb trust-report --profile atb.profile.policy_decision --format markdown
```

`@latest` selects a Go module version, not necessarily the newest main commit.
See [installation options](#2-installation-options) for published/source version
differences and the source build needed for visual investigation.

The Go CLI is the primary install path. The Python and TypeScript
packages are SDKs that write the same `.atb` format.

### Reproduce the incident-forensics wedge locally

From a source checkout with the Python SDK dependencies installed:

```bash
make demo-incident
```

The script creates an offline bundle containing a denied policy decision, an
unapproved tool call, a failed action, a human override with reviewer identity
evidence, and a closed session. It then runs `verify`, `trust-report`,
`incident list`, and `incident report`. Open the same bundle in the local
viewer:

```bash
./atb view \
  --bundle run.atb/incident-demo/incident.atb \
  --profile atb.profile.policy_decision
```

This is the canonical ATB workflow: the AI workflow itself can fail
while the audit evidence still verifies cleanly. Use the snapshot name
to label workflow state. `atb verify` checks the hash chain and any
recorded bundle signature or anchor evidence. `atb bundle new` is the
explicit alias for `atb init`; both are supported. The example uses
`atb.profile.policy_decision` because the recorded evidence shows a
request, a pending action, and the policy outcome that blocked it. Use an
explicit `--profile` for first-run verification so the result is stable
even when a bundle contains events that could match more than one built-in
profile. For
the full local review and export path, continue with [Run
forensics](../investigate/incidents.md).

### What the output means

Abbreviated `atb verify --profile atb.profile.policy_decision --format json` output:

```json
{
  "profile_id": "atb.profile.policy_decision",
  "pass": true,
  "cas_score": 0.70,
  "cas_grade": "Medium",
  "critical_failures": [],
  "residual_risk": {
    "level": "Medium",
    "drivers": [],
    "recommended_next_evidence": []
  }
}
```

- `pass: true` means the bundle chain is intact and the selected profile passed its required checks.
- `cas_score` and `cas_grade` describe how much of the expected evidence ATB can see for that workflow.
- `residual_risk` is an object. `residual_risk.level` summarises what is still weak or missing in the recorded evidence, not whether the underlying decision was correct.

If you want the full integrity fields, run `atb verify --profile atb.profile.policy_decision --json`
and check `integrity.chain_valid: true`.

### Common first-run outcomes

- `pass: true` with a populated `profile_id` means the selected profile passed and the bundle chain is intact.
- `profile_id: ""` means no workflow profile was selected or matched. This is common for manifest-only or zero-event bundles. In that case `pass` and `profile_pass` stay `false` even when the hash chain is intact. An empty `critical_failures` array does not mean the profile passed. Read `gate_result.chain_valid` or the text line `Integrity: PASS`, and expect exit 0. The full distinction is in [What verification means](../verification-meaning.md#no-profile-selected).
- `pass: false` with non-empty `critical_failures` means the selected profile is missing required evidence.
- `residual_risk.level: "Critical"` means do not treat the bundle as trustworthy evidence until you inspect the failure. `residual_risk.level: "High"` with a driver that no profile was evaluated is the no-profile case above, not a broken chain. Use `atb verify --json` when you need the diagnostic integrity report; `--format json` remains the stable automation contract. Diagnostic JSON can show `anchoring.status: "failed"` for an absent optional anchor. Read it with `anchor_present: false` and `anchor_required: false`: no timestamp check was attempted. See [Absent optional anchor](../verification-meaning.md#absent-optional-anchor).

The markdown trust report from the first command can include `Operational Safety (PASS)` and `Test Coverage (PASS)`, and its opening `Verdict` can be `PASS` for an intact bundle. Those two headings mean the CLI found named ATB repository files in the working directory or embedded in the binary. They do not assess the workflow you recorded, and they do not mean tests ran. The bundle chain in that report is `Cryptographic Integrity`. This verdict is not the stable verify `pass` field. Details are in [Trust-report file headings](../verification-meaning.md#trust-report-file-headings).

### Verify against a specific profile

Use `--profile <id>` to evaluate against a specific built-in profile.
See `atb events --profile <id>` for the required event types per
profile. Local YAML definitions are also supported via
`--profile ./path/to/profile.yaml`:

```bash
atb events --profile atb.profile.rag_answer
atb verify --bundle run.atb/bundle.atb --profile atb.profile.privileged_tool_action
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --format json
atb verify --bundle run.atb/bundle.atb --profile atb.profile.rag_answer --json
atb verify --bundle run.atb/bundle.atb --profile ./profiles/release-review.yaml
```

Use `--format json` for the compact verifier report (`pass`, `cas_score`, `cas_grade`,
`critical_failures`, `residual_risk`). Use `--json` for the full report, including
`integrity.chain_valid` and the full CAS breakdown.

### CI lock contention

Bundle writes fail fast when another process holds the advisory lock. In CI fan-out, opt into bounded retries with `ATB_LOCK_WAIT` or `--lock-wait`:

```bash
ATB_LOCK_WAIT=10s atb snapshot ci_checkpoint
```

## 2. Installation options

### Published packages and unreleased source

As of **2026-11-08**, the latest [GitHub release](https://github.com/pcguest/atb/releases/latest),
[PyPI SDK](https://pypi.org/project/atb-sdk/) and
[npm SDK](https://www.npmjs.com/package/@pcguest/atb-sdk) are **1.18.1**. The
repository's **2.0.0** version strings describe unreleased source; no 2.0.0 tag
or registry package exists at this checkpoint. An in-tree version or changelog
heading does not establish publication. Check those release/registry pages
again when choosing a version; use documentation matching the installed build.

| Route | Local verification | Investigation UI |
| --- | --- | --- |
| Published CLI release | Go CLI features for that release | Consult that release's assets/docs |
| `go install` | Go CLI features for the resolved module version | Minimal installation guidance; no bundle data |
| Source + `make build` | Current source CLI | Full embedded local UI |
| Python or npm SDK package | SDK verification capabilities; Go CLI for profiles and incident analysis | No embedded Go viewer |

### Go-installed CLI

```bash
go list -m -json github.com/pcguest/atb/v2@latest
atb_install_dir="$HOME/.local/bin"
GOBIN="$atb_install_dir" go install github.com/pcguest/atb/v2/cmd/atb@latest
export PATH="$atb_install_dir:$PATH"
"$atb_install_dir/atb" version
go version -m "$atb_install_dir/atb"
```

The explicit user-local install directory keeps generated binaries outside the
checkout. The commands inspect that exact binary and prepend its directory
for the remaining CLI examples.

[`@latest`](https://go.dev/ref/mod#version-queries) follows Go's module-version
selection rules: release versions take precedence; in the absence of an
applicable tagged version Go can resolve a pseudo-version. It does not always
mean main. At the checkpoint above, the v2 query resolved to
`v2.0.0-20261108115548-779af5a73d8a`; this is a source pseudo-version, not a
published v2.0.0 release. The `atb version` string alone does not identify the
source revision; retain the `go version -m` module/build information too.

### Source build with the investigation UI

Prerequisites: Git, Go 1.26.10, Node.js 22 and npm. From a checkout, run the
actual Makefile target, which installs/builds the web dependencies and then
embeds the generated assets in the Go binary:

```bash
git clone https://github.com/pcguest/atb.git
cd atb
# Reproduce the 2026-11-08 integrated source checkpoint:
git checkout 779af5a73d8a6fcb8536733798aab2238c235137
make build
./atb version
./atb bundle new
./atb append ai.request.received --data='{"request_id":"req-1042"}'
./atb verify run.atb/bundle.atb
./atb view run.atb/bundle.atb
```

Use `./atb` to select this build rather than another `atb` on PATH. To evaluate
newer source, choose and record that revision explicitly. The source version
string still does not imply a tagged or published release.

The current viewer's human keyboard, screen-reader and 200%/400% zoom
acceptance remains outstanding. Automated checks or historical owner reports
do not establish that current acceptance. The viewer is not
accessibility-certified; this developer preview prioritises the CLI.

### Python and TypeScript SDKs

```bash
python -m pip install atb-sdk
npm install @pcguest/atb-sdk
```

These install the registry versions, which may differ from this source guide.
They are SDKs only. Their installed `atb` command is a compatibility stub that
prints Go CLI install guidance and is scheduled for removal in a future major
release. Use the Go CLI for the full profile/incident workflow and a source
build for the embedded UI.

## 3. Record your first bundle (Python SDK)

```python
from atb import Bundle

bundle = Bundle()

bundle.append("ai.request.received", {
    "request_id": "req-001",
    "actor_id_hash": "sha256-user-001",
    "purpose_tag": "rag_answer",
}, timestamp="2026-03-03T10:00:00Z")

bundle.append("ai.model.invoked", {
    "model_provider": "openai",
    "model_id": "gpt-4",
    "model_parameters_digest": "sha256-params-001",
    "prompt_digest": "sha256-prompt-001",
}, timestamp="2026-03-03T10:00:02Z")

bundle.append("ai.model.output", {
    "output_digest": "sha256-output-001",
    "output_format": "text",
}, timestamp="2026-03-03T10:00:03Z")

bundle.save("my-trace.atb")
bundle.verify()
```

## 4. Record your first bundle (TypeScript SDK)

```typescript
import { Bundle } from "@pcguest/atb-sdk";

const bundle = new Bundle();

bundle.append("ai.request.received", {
  request_id: "req-001",
  actor_id_hash: "sha256-user-001",
  purpose_tag: "rag_answer",
});

bundle.append("ai.model.invoked", {
  model_provider: "openai",
  model_id: "gpt-4",
  model_parameters_digest: "sha256-params-001",
  prompt_digest: "sha256-prompt-001",
});

await bundle.save("my-trace.atb");
bundle.verify();
```

For a runnable TypeScript quickstart example, see [examples/typescript/](../../examples/typescript/).

## 5. Open the local review UI

```bash
# Default bundle path
atb view

# Custom bundle path
atb view my-trace.atb --port 8080

# Evaluate against a profile at startup (shows profile/CAS summary in the viewer)
atb view --profile atb.profile.rag_answer
atb view --bundle run.atb/bundle.atb --profile ./profiles/custom.yaml
```

`atb view` opens one local investigation surface for one bundle at a time.
`--profile` runs verification at startup and makes its separate integrity,
profile-coverage, and custody answers available immediately. Without
`--profile`, use "Verify bundle" in the command palette to trigger
`POST /api/v1/bundle/verify`. Investigation follows Run → Findings → Timeline
→ Context → Relationships → Evidence → Evidence status; findings link directly
to their supporting records.

`atb view` requires building from source to include the embedded review UI:
`make build` from the checkout root (see [installation options](#2-installation-options)).

If you install with `go install`, `atb view` serves a minimal install-guidance page with no
bundle data. Build from source when you need the embedded review UI.

Security note: `atb view` accepts loopback hosts only. All API endpoints require a
session token generated at startup and delivered in the browser URL fragment. It is not a
network-hosted viewer.

Viewer details:

- [Viewer specification](../specification/viewer.md)

## 6. Run evidence export

```bash
atb incident list --bundle run.atb/bundle.atb
atb incident report --bundle run.atb/bundle.atb --session <session-id> --format markdown
atb incident export --bundle run.atb/bundle.atb --session <session-id> --out incident-review-evidence.zip
```

For incident review, prefer `atb incident export`: it packages the verified
bundle, session-scoped reports, chain-of-custody manifest, checksums, and
hash-addressed event evidence. It stays local unless you explicitly pass a
Mortise endpoint.

For a profile-aware EU AI Act review pack:

```bash
atb compliance pack \
  --bundle run.atb/bundle.atb \
  --profile atb.profile.policy_decision \
  --regime eu-ai-act \
  --out eu-ai-act-pack.zip
```

Compliance packs are deterministic offline review artefacts. They include the
selected profile/CAS result, obligation mapping, relevant incident reports, and
retention evidence when `.atb/operations.atb` is present.

## 7. Compliance exports

```bash
atb config retention --days 183
atb archive
atb export --format soc2 --bundle run.atb/bundle.atb --output soc2-evidence.zip
atb export --format soc2 --output soc2-evidence.zip --with-verify
# Writes soc2-evidence.zip and soc2-evidence.zip.verify.json
atb export --format gdpr --type dsr --subject-id usr_123 --bundle run.atb/bundle.atb --output gdpr-dsr.zip
atb export --format gdpr --type ropa --bundle run.atb/bundle.atb --output gdpr-ropa.zip
```

Reference docs:

- [Compliance hub](../compliance/README.md)
- [Run forensics](../investigate/incidents.md)

## 8. AI integrations

```python
from atb.langchain_callback import ATBCallbackHandler

handler = ATBCallbackHandler(privacy_mode="hash")
```

Integration docs:

- [Capture guide](../capture/overview.md)
- [Chatlog import](../integrations/chatlog-import.md)
- [LangChain integration](../integrations/langchain.md)
- [Vercel AI SDK integration](../integrations/vercel-ai.md)

## 9. Next steps

- [Go example](../../examples/go/README.md)
- [Capture guide](../capture/overview.md)
- [Python SDK](../../sdk/python/README.md)
- [TypeScript SDK](../../sdk/typescript/README.md)
- [Documentation hub](../README.md)
- [ATB bundle specification](../specification/bundle-v1.md)
- [Security model](../concepts/trust-model.md)
- [Configuration reference](./configuration.md)
- [Contributing](../../CONTRIBUTING.md)
