#!/usr/bin/env bash
# Exercise the actual embedded binary. Run make build and make demo-incident first.
set -euo pipefail
cd "$(dirname "$0")/.."
fixture_bin="${ATB_BIN:-$PWD/atb}"
# Do not accidentally test against an already-running View process.  A fixed
# default port made a stale server look like the just-started fixture and could
# yield false results for the remaining fixtures.  An explicit port remains
# supported for callers that need one.
if [ -n "${ATB_VIEW_TEST_PORT:-}" ]; then
  fixture_port="$ATB_VIEW_TEST_PORT"
  if lsof -nP -iTCP:"$fixture_port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "Requested View test port $fixture_port is already in use." >&2
    exit 1
  fi
else
  fixture_port=""
  for candidate_port in $(seq 18889 18989); do
    if ! lsof -nP -iTCP:"$candidate_port" -sTCP:LISTEN >/dev/null 2>&1; then
      fixture_port="$candidate_port"
      break
    fi
  done
  if [ -z "$fixture_port" ]; then
    echo "No free View test port found in 18889-18989." >&2
    exit 1
  fi
fi
fixture_token=0000000000000000000000000000000000000000000000000000000000000001
fixture_pid=""
fixture_artifacts="${ATB_VIEW_TEST_ARTIFACTS:-$PWD/.tmp/view-fixtures}"
mkdir -p "$fixture_artifacts"
cleanup() {
  if [ -n "$fixture_pid" ]; then
    kill "$fixture_pid" 2>/dev/null || true
    wait "$fixture_pid" 2>/dev/null || true
    fixture_pid=""
  fi
}
trap cleanup EXIT
for fixture_path in \
  examples/bundles/profiles/privileged_tool_action-pass.atb \
  examples/bundles/profiles/privileged_tool_action-fail.atb \
  examples/bundles/profiles/rag_answer-pass.atb \
  examples/bundles/profiles/rag_answer-fail.atb \
  run.atb/incident-demo/incident.atb \
  run.atb/incident-demo/incident-content-tampered.atb; do
  test -f "$fixture_path" || { echo "Missing fixture: $fixture_path. Run make profile-fixtures and make demo-incident." >&2; exit 1; }
  fixture_name="$(basename "$fixture_path" .atb)"
  "$fixture_bin" view --bundle "$fixture_path" --session-token "$fixture_token" --no-open --port "$fixture_port" > "$fixture_artifacts/$fixture_name-server.log" 2>&1 &
  fixture_pid=$!
  fixture_ready=false
  for fixture_attempt in {1..30}; do
    if curl --silent --fail "http://127.0.0.1:$fixture_port/view/" > /dev/null; then
      fixture_ready=true
      break
    fi
    kill -0 "$fixture_pid" 2>/dev/null || { echo "View exited; inspect $fixture_artifacts/$fixture_name-server.log" >&2; exit 1; }
    sleep 1
  done
  "$fixture_ready" || { echo "View did not become ready" >&2; exit 1; }
  (
    cd web
    env -u ELECTRON_RUN_AS_NODE CYPRESS_BASE_URL="http://127.0.0.1:$fixture_port" \
      npx cypress run --browser firefox --spec cypress/e2e/live-investigation.cy.ts \
      --config "screenshotsFolder=$fixture_artifacts/$fixture_name" \
      --env "MOCK_API=false,SESSION_TOKEN=$fixture_token"
  )
  cleanup
done
