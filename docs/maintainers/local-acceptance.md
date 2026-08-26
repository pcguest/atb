# Local release-candidate acceptance

Use this runbook to test an exact candidate as a new user would. Run it from a
fresh clone, not a development checkout. Replace `<SHA>` with the candidate
commit. Nothing in this procedure publishes ATB.

Minimum supported tools are Go 1.26.7, Python 3.9, Node.js 22, and npm with the
committed lockfiles. Release tooling uses Python 3.11. The live viewer
acceptance uses Firefox.

## 1. Clone and prepare isolated tools

```bash
git clone https://github.com/pcguest/atb.git atb-rc
cd atb-rc
git checkout --detach <SHA>

go version
python3.9 --version
python3.11 --version
node --version
npm --version

python3.9 -m venv .venv
if ! .venv/bin/python -m pip --version >/dev/null 2>&1; then
  .venv/bin/python -m ensurepip --upgrade
fi
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install -r sdk/python/requirements-dev.txt
python -m pip install -e './sdk/python[dev]' --no-deps

python3.11 -m venv .venv-release
.venv-release/bin/python -m pip install --upgrade pip
.venv-release/bin/python -m pip install -r sdk/python/requirements-release.txt

npm --prefix web ci
npm --prefix sdk/typescript ci
make bootstrap-scanners
```

This creates all Python and Node state inside the clone. Success means the
declared requirements and lockfiles are sufficient; an undeclared global
package must not be needed. The scanners are downloaded from their official
release assets into `.tmp/bin`, verified against repository-pinned SHA-256
checksums, and checked for the exact supported versions. The gold gate prefers
those verified repository-local binaries even when an older global scanner is
installed.

Some system Python builds omit `ensurepip`. On those platforms, replace the
Python 3.9 `venv` plus `ensurepip` lines with `uv venv --python 3.9 --seed
<venv-path>`; keep `UV_CACHE_DIR="$PWD/.tmp/uv-cache"` and
`UV_PYTHON_INSTALL_DIR="$PWD/.tmp/uv-python"` so the fallback remains isolated
inside the checkout.

## 2. Build and run the release gates

```bash
make build
make gate-gold-release
SKIP_DOCKER=1 EXPECT=1.15.2 bash scripts/release-check.sh
```

`make build` produces `./atb` with the full embedded local viewer. The gold gate
runs the production viewer build, Go and SDK tests, security scans, coverage,
Firefox E2E, and accessibility. `release-check.sh` rebuilds and validates the
candidate packages in a disposable Python environment. Use the candidate's
source version for `EXPECT`; change it to `1.15.3` on the release branch.

Docker being skipped locally is not a pass. Record it as blocked and require
the hosted Docker build and Trivy image checks.

## 3. Run and interpret the flagship incident

```bash
make demo-incident
cat examples/incident-demo/application.log
./atb incident list --bundle run.atb/incident-demo/incident.atb
./atb incident report \
  --bundle run.atb/incident-demo/incident.atb \
  --session sess-incident-demo
./atb verify --bundle run.atb/incident-demo/incident.atb
```

The demo creates the same bundle twice, verifies it, reconstructs the session,
and deliberately creates three damaged copies. The application log shows only
a generic failure; it does not establish the decisive tool, policy reason,
reviewer, evidence order, or finding. The ATB report shows the recorded facts
and their order. `tool_without_approval` means no matching earlier approval is
present in the captured evidence, not that no approval existed elsewhere.

Verification success means the content and order presented in the bundle match
its hash chain. It does not prove complete capture, pre-capture truthfulness,
real-world actor identity, model correctness, legal compliance, or external
custody.

## 4. Check tamper failures directly

```bash
set +e
./atb verify --bundle run.atb/incident-demo/incident-content-tampered.atb; echo "content exit: $?"
./atb verify --bundle run.atb/incident-demo/incident-order-tampered.atb; echo "order exit: $?"
./atb verify --bundle run.atb/incident-demo/incident-record-removed.atb; echo "removal exit: $?"
set -e
```

Each command must exit `2`. That result means the presented record chain no
longer matches. Exit `0` would be a release blocker.

## 5. Review the same evidence visually

```bash
./atb view --bundle run.atb/incident-demo/incident.atb
```

Open the generated loopback URL in Firefox and confirm the incident, ordered
events, record hashes, profile/CAS information, and verification result agree
with the CLI. A source build contains the full viewer. A `go install` build has
the CLI but intentionally serves only viewer-installation guidance.

## 6. Test the Python package as a consumer

```bash
(cd sdk/python && ../../.venv-release/bin/python -m build --no-isolation && ../../.venv-release/bin/python -m twine check dist/*)
PY_CONSUMER="$(mktemp -d)"
python3.9 -m venv "$PY_CONSUMER/venv"
if ! "$PY_CONSUMER/venv/bin/python" -m pip --version >/dev/null 2>&1; then
  "$PY_CONSUMER/venv/bin/python" -m ensurepip --upgrade
fi
"$PY_CONSUMER/venv/bin/python" -m pip install --upgrade pip
"$PY_CONSUMER/venv/bin/python" -m pip install --no-deps sdk/python/dist/*.whl
"$PY_CONSUMER/venv/bin/python" - "$PY_CONSUMER/consumer.atb" <<'PY'
import importlib.util
import sys
import atb
from atb import Bundle

path = sys.argv[1]
bundle = Bundle(goal="clean Python consumer")
bundle.append("ai.policy.decision", {"decision": "allow", "policy_id": "local.workflow"})
bundle.save(path)
loaded = Bundle.load(path)
loaded.verify()
assert importlib.util.find_spec("pageindex") is None
print(atb.__version__)
PY
./atb verify --bundle "$PY_CONSUMER/consumer.atb"
```

Success proves the built wheel imports, writes, reloads, and interoperates with
the Go CLI without repository `PYTHONPATH` or optional PageIndex packages.

## 7. Test the packed TypeScript package

```bash
(cd sdk/typescript && npm run typecheck && npm test && npm run build && npm pack)
TS_CONSUMER="$(mktemp -d)"
cd "$TS_CONSUMER"
npm init -y
npm install "$OLDPWD"/sdk/typescript/pcguest-atb-sdk-*.tgz
node --input-type=module <<'JS'
import { AI_REQUEST_RECEIVED_EVENT_TYPE, Bundle, SDK_VERSION } from "@pcguest/atb-sdk";
const bundle = new Bundle();
bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
  request_id: "req-clean-consumer",
  actor_id_hash: "sha256:clean-consumer",
  purpose_tag: "release_acceptance",
});
bundle.save("consumer.atb");
Bundle.load("consumer.atb").verify();
console.log(SDK_VERSION);
JS
cd "$OLDPWD"
./atb verify --bundle "$TS_CONSUMER/consumer.atb"
```

Inspect `npm pack --dry-run` as well: only the built package surface, README,
LICENSE, and THIRD_PARTY_NOTICES should ship. Run the repository's focused
agent-client real-process test before release; the transport remains a bounded,
loopback-only, low-volume control-plane API.

## 8. Test the installed CLI path

```bash
GOBIN="$(mktemp -d)"
export GOBIN
go install -tags noembed ./cmd/atb
"$GOBIN/atb" version
"$GOBIN/atb" verify --bundle run.atb/incident-demo/incident.atb
"$GOBIN/atb" view --bundle run.atb/incident-demo/incident.atb \
  --no-open --port 18889 >/tmp/atb-go-install-view.log 2>&1 &
VIEW_PID=$!
sleep 2
curl -fsS http://127.0.0.1:18889/view/ | grep -F 'make build'
kill "$VIEW_PID"
```

The installed CLI must create, inspect, and verify bundles. Its viewer command
must display the documented installation guidance rather than silently
pretending to contain the full embedded viewer.

## 9. Operational ATB workflow

- During development: build with `make build`; use `.venv` for Python 3.9 SDK
  compatibility, `.venv-release` for packaging, and the two npm lockfiles.
- During an agent run: use an SDK/interceptor/importer or `atb capture run` to
  write local evidence.
- After an incident: run `atb verify`, `atb incident list`, then
  `atb incident report`.
- To verify received evidence: run `atb verify --bundle <bundle.atb>` offline.
- To inspect visually: run `atb view --bundle <bundle.atb>` from a source or
  release build with the embedded viewer.
- To demonstrate tampering: modify a copy and verify it; the integrity failure
  must exit `2`.
- To package incident evidence: use `atb incident export`.
- To package mapped review evidence: use `atb compliance pack`; it is technical
  evidence, not certification.
- Optional Mortise hand-off begins only when durable organisational custody,
  retention, receipts, shared access, or governance is required.
