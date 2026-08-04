#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <spec>" >&2
  exit 1
fi

SPEC="$1"
HOST="${STATIC_HOST:-127.0.0.1}"
PORT="${STATIC_PORT:-3000}"
BASE_URL="${CYPRESS_BASE_URL:-http://${HOST}:${PORT}}"
SERVER_LOG="${STATIC_SERVER_LOG:-/tmp/atb-static-cypress.log}"

python3 -m http.server "$PORT" --bind "$HOST" --directory out >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

cleanup() {
  kill "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
}

trap cleanup EXIT INT TERM

ATTEMPT=0
until curl -fsI "${BASE_URL%/}/view/" >/dev/null 2>&1; do
  ATTEMPT=$((ATTEMPT + 1))
  if [ "$ATTEMPT" -ge 30 ]; then
    echo "static test server did not become ready at ${BASE_URL%/}/view/" >&2
    exit 1
  fi
  sleep 1
done

# ELECTRON_RUN_AS_NODE (sometimes set by IDEs) makes Cypress treat its binary as
# Node and reject Electron smoke-test flags. Clear it for the browser launch.
env -u ELECTRON_RUN_AS_NODE cypress run --spec "$SPEC" --browser firefox --env MOCK_API=true
