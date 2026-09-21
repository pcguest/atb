#!/bin/sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: $0 <spec> [--record] [--key <record-key>]" >&2
  exit 1
fi

SPEC="$1"
shift
case "$SPEC" in
  /*) ;;
  *) SPEC="$(pwd)/$SPEC" ;;
esac
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
OUT_DIR="${SCRIPT_DIR}/../out"
HOST="${STATIC_HOST:-127.0.0.1}"
PORT="${STATIC_PORT:-3000}"
BASE_URL="${CYPRESS_BASE_URL:-http://${HOST}:${PORT}}"
SERVER_LOG="${STATIC_SERVER_LOG:-/tmp/atb-static-cypress.log}"

cd "${SCRIPT_DIR}/.."

# Validate OUT_DIR exists before starting server
if [ ! -d "$OUT_DIR" ]; then
  echo "static output directory not found: $OUT_DIR" >&2
  echo "run 'npm run build' in web/ first" >&2
  exit 1
fi

python3 -m http.server "$PORT" --bind "$HOST" --directory "$OUT_DIR" >"$SERVER_LOG" 2>&1 &
SERVER_PID=$!

# Verify server process stays alive
sleep 0.5
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
  echo "static server failed to start (see $SERVER_LOG)" >&2
  exit 1
fi

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

# Parse passthrough args: allow ONLY --record and --key <value>
PASSTHROUGH=""

is_reserved_flag() {
  case "$1" in
    --browser|--spec|--env|--config|--config-file|--project|--reporter|--reporter-options)
      return 0
      ;;
    --browser=*|--spec=*|--env=*|--config=*|--config-file=*|--project=*|--reporter=*|--reporter-options=*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --record)
      PASSTHROUGH="$PASSTHROUGH --record"
      shift
      ;;
    --key)
      if [ "$#" -lt 2 ]; then
        echo "error: --key requires a value" >&2
        exit 1
      fi
      PASSTHROUGH="$PASSTHROUGH --key $2"
      shift 2
      ;;
    --key=*)
      # Cypress accepts --key=VALUE but we require --key VALUE for consistent parsing
      echo "error: --key=VALUE syntax not supported; use --key VALUE" >&2
      exit 1
      ;;
    *)
      if is_reserved_flag "$1"; then
        echo "error: reserved flag '$1' is controlled by the wrapper and cannot be overridden" >&2
        exit 1
      fi
      case "$1" in
        --*)
          echo "error: unknown passthrough flag '$1' (allowed: --record, --key <value>)" >&2
          exit 1
          ;;
        *)
          echo "error: unexpected argument '$1' (allowed: --record, --key <value>)" >&2
          exit 1
          ;;
      esac
      ;;
  esac
done

# ELECTRON_RUN_AS_NODE (sometimes set by IDEs) makes Cypress treat its binary as
# Node and reject Electron smoke-test flags. Clear it for the browser launch.
# $PASSTHROUGH is intentionally unquoted to allow word-splitting into separate args.
# It only contains controlled tokens (--record, --key, alphanumeric key values).
# shellcheck disable=SC2086
CYPRESS_BASE_URL="$BASE_URL" env -u ELECTRON_RUN_AS_NODE npx --no-install cypress run --spec "$SPEC" --browser firefox --env MOCK_API=true $PASSTHROUGH
