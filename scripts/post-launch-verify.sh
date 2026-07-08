#!/usr/bin/env bash
#
# Post-launch external verification for ATB v1.15.1.
#
# Run this AFTER launch, from a machine with no special access, to prove the
# release works from the outside: the CLI installs from the public repo at
# 1.15.1, both SDKs publish at 1.15.1 (not the previous published 1.14.5),
# the quickstart runs, and verify exits 0 on an intact bundle and 2 on a
# tampered one.
#
# It fails loudly if any artefact is stale at 1.14.5, missing, or unreachable.
# It makes no changes to any repository, registry, or remote. It only reads
# public artefacts and works in a temporary directory.
#
# Requirements on the runner: go, python3 (with venv), npm. No credentials.
#
# Usage:
#   scripts/post-launch-verify.sh
#
# Override the expected version if you are verifying a later release:
#   EXPECT=1.15.2 scripts/post-launch-verify.sh

set -u

EXPECT="${EXPECT:-1.15.1}"
# 1.14.5 is the newest version ever published to the registries; the gated
# v1.15.0 baseline was never released, so a stale mirror serves 1.14.5.
STALE="1.14.5"
MODULE="github.com/pcguest/atb/cmd/atb"
PYPKG="atb-sdk"
NPMPKG="@pcguest/atb-sdk"

pass=0
fail=0
ok()   { printf "  PASS  %s\n" "$1"; pass=$((pass+1)); }
bad()  { printf "  FAIL  %s\n" "$1"; fail=$((fail+1)); }
head() { printf "\n== %s ==\n" "$1"; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
export GOBIN="$WORK/gobin"
export PATH="$GOBIN:$PATH"

# ---------------------------------------------------------------------------
head "1. CLI installs from the public repo at $EXPECT"
# ---------------------------------------------------------------------------
if go install "$MODULE@v$EXPECT" 2>"$WORK/goinstall.log"; then
  ver="$("$GOBIN/atb" version 2>/dev/null | awk '{print $NF}')"
  if [ "$ver" = "$EXPECT" ]; then
    ok "go install $MODULE@v$EXPECT -> atb $ver"
  elif [ "$ver" = "$STALE" ]; then
    bad "CLI reports the STALE version $STALE, expected $EXPECT"
  else
    bad "CLI reports '$ver', expected $EXPECT"
  fi
else
  bad "go install failed (repo private or tag missing?): $(tail -1 "$WORK/goinstall.log")"
fi

# ---------------------------------------------------------------------------
head "2. Quickstart runs on a fresh checkout"
# ---------------------------------------------------------------------------
if command -v atb >/dev/null 2>&1; then
  qs="$WORK/qs"; mkdir -p "$qs"
  if ( cd "$qs"
       printf '#!/bin/sh\necho hello world\n' > task.sh; chmod +x task.sh
       atb bundle new >/dev/null 2>&1
       atb capture run -- ./task.sh >/dev/null 2>&1
       atb verify run.atb/bundle.atb >/dev/null 2>&1 ); then
    ok "bundle new, capture run, verify (exit 0)"
  else
    bad "quickstart sequence did not complete cleanly"
  fi
else
  bad "atb not on PATH; skipping quickstart"
fi

# ---------------------------------------------------------------------------
head "3. Python SDK ($PYPKG) publishes at $EXPECT and interoperates with the CLI"
# ---------------------------------------------------------------------------
if ! python3 -m venv "$WORK/venv" 2>/dev/null || ! . "$WORK/venv/bin/activate" 2>/dev/null; then
  bad "python venv setup failed; skipping Python SDK checks (pip must not run against the host)"
elif pip install -q "$PYPKG==$EXPECT" 2>"$WORK/pip.log"; then
  pyver="$(python3 -c 'import importlib.metadata as m; print(m.version("atb-sdk"))' 2>/dev/null)"
  if [ "$pyver" = "$EXPECT" ]; then
    ok "pip install $PYPKG==$EXPECT -> $pyver"
  else
    bad "Python SDK reports '$pyver', expected $EXPECT"
  fi

  # Build a 3-record bundle with the SDK, verify it with the CLI: exit 0.
  python3 - "$WORK/check.atb" <<'PY'
import sys
from atb import Bundle
b = Bundle(goal="post-launch check")
b.append("ai.policy.decision", {"decision": "allow", "policy_id": "local.workflow"})
b.append("ai.action.error", {"message": "demo error"})
b.save(sys.argv[1])
PY
  if command -v atb >/dev/null 2>&1; then
    if atb verify "$WORK/check.atb" >/dev/null 2>&1; then
      ok "CLI verifies an SDK-written bundle (exit 0)"
    else
      bad "CLI failed to verify an SDK-written bundle"
    fi

    # Tamper a non-manifest record; verify must exit exactly 2.
    awk '/ai.action.error/{gsub(/demo error/,"TAMPERED")}1' \
      "$WORK/check.atb" > "$WORK/check.t" && mv "$WORK/check.t" "$WORK/check.atb"
    atb verify "$WORK/check.atb" >/dev/null 2>&1; code=$?
    if [ "$code" -eq 2 ]; then
      ok "verify exits 2 on a tampered bundle"
    else
      bad "verify exited $code on a tampered bundle, expected 2"
    fi
  else
    bad "atb not on PATH; cannot cross-check the SDK bundle"
  fi
else
  bad "pip install $PYPKG==$EXPECT failed: $(tail -1 "$WORK/pip.log")"
fi
deactivate 2>/dev/null || true

# ---------------------------------------------------------------------------
head "4. TypeScript SDK ($NPMPKG) publishes at $EXPECT"
# ---------------------------------------------------------------------------
npmver="$(npm view "$NPMPKG" version 2>/dev/null)"
if [ "$npmver" = "$EXPECT" ]; then
  ok "npm view $NPMPKG version -> $npmver"
elif [ "$npmver" = "$STALE" ]; then
  bad "npm reports the STALE version $STALE, expected $EXPECT"
elif [ -z "$npmver" ]; then
  bad "npm could not resolve $NPMPKG (unpublished or unreachable)"
else
  bad "npm reports '$npmver', expected $EXPECT"
fi

# ---------------------------------------------------------------------------
head "Result"
# ---------------------------------------------------------------------------
printf "  %d passed, %d failed\n" "$pass" "$fail"
if [ "$fail" -ne 0 ]; then
  printf "  LAUNCH NOT CONFIRMED. Investigate the failures above before announcing.\n"
  exit 1
fi
printf "  Launch confirmed from the outside at v%s.\n" "$EXPECT"
