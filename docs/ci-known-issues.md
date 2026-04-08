# CI known issues

This document records currently known CI issues so they can be tracked consistently without overstating their scope.

## Windows runner flake

**Description:** ATB CI runs on `windows-latest` in `.github/workflows/ci.yml`. Intermittent Windows-only failures should be treated as infrastructure flakes unless they are reproducible locally or on rerun.

**Impact:** A failing Windows job can block confidence in the pipeline until it is clear whether the failure is a real regression or a runner-specific flake.

**Current status:** Tracking only. No permanent fix is recorded in this document.

**Workaround:** When a Windows-only failure occurs, link the failing run, note whether the rerun passed, and capture the failing step or test name for trend tracking.
