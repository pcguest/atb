CI Known Issues
Windows runner flake (tracking)

ATB CI runs on `windows-latest` in `.github/workflows/ci.yml`.
Intermittent Windows-only failures should be treated as infra flakes unless
reproducible locally or on rerun.

When a Windows-only failure occurs:

    link the failing run

    note whether rerun passed

    capture the failing step/test name for trend tracking

text
