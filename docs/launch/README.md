# docs/launch

Templates for release communications. Fill these in at release time; do not treat the
placeholder bullets as descriptions of the current release.

## Files

**[release-announcement-template.md](release-announcement-template.md)**
Post template for GitHub Releases and distribution channels. Contains a Highlights block
(replace with shipped items only), a stable Quick Start using the explicit `vX.Y.Z` tag, and
links to docs and the security policy. The use-case bullets are stable across releases.

**[release-notes-template.md](release-notes-template.md)**
Internal fill-in-the-blank template for drafting release notes before they are edited into
the Highlights block of the announcement. Structured around what shipped, who benefits, and
known limits. States the authoritative install path and distinguishes CLI from SDK packages.

## Assets

Canonical demo asset paths referenced from README.md and docs/quickstart.md:

| Path | Content |
|------|---------|
| `docs/launch/assets/atb-verify-demo.gif` | Terminal walkthrough: bundle init, events appended, `atb verify` success, manual tamper + failing verify, `atb view` profile/CAS summary panel |
| `docs/launch/assets/atb-verify-report.png` | Example `atb verify --profile` report summary: profile ID, pass/fail, CAS score/grade, chain/anchor status, critical obligation failures |

Before launch, regenerate both assets against the current release tag and check that README.md
and docs/quickstart.md references still point to these paths. If paths change, update both docs
at the same time.
