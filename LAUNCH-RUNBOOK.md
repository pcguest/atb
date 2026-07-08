# Tenon and Mortise Launch Runbook

This is a review artefact. Do not run any step until Patrick gives the
step-specific word.

Hard stop: no public flip, repository rename, fresh repository create, push,
tag, release, SDK publish, Docker publish, or site deploy happens from this file
without explicit approval.

All shell commands use `/bin/bash` explicitly. Go validation uses `go1.26.4`.
All ATB commits must be signed by Patrick Guest `<patrickcguest@proton.me>`.

## Current State

- ATB local checkout: `/Users/paddyguest/atb`, branch `release/v1.15.0`.
- ATB local Tenon framing commit: `dafed1a`, committed locally, unpushed.
- Mortise local checkout: `/Users/paddyguest/mortise`, branch
  `site/marketing-front`.
- Mortise prose rename commit: `38c365e`, committed locally, unpushed.
- Current staging remote `pcguest/tenon` is private but has nine Dependabot PR
  refs. It is not the public module home and has no bearing on the launch push.
- Current legacy archive `pcguest/atb-legacy` is private. Read-only checks on
  2026-06-20 showed `https://github.com/pcguest/atb.git` redirects there.
- `pcguest/atb-legacy` stays private and untouched.
- Backups remain:
  `/Users/paddyguest/atb-backup-pre-rewrite-20260618.git` and
  `/Users/paddyguest/atb-clean-pushable-20260618.bundle`.

## Frozen Contract

These strings are load-bearing and frozen for v1:

- Go module path: `github.com/pcguest/atb`.
- CLI binary name: `atb`.
- Bundle extension: `.atb`.
- Schema identifiers at `atb.dev`, including `verify.report.v1`.
- SDK package names: PyPI `atb-sdk`, npm `@pcguest/atb-sdk`, Python import
  `atb`.
- Golden vectors.
- Canonical event-type set: 41 strings. The set is 16 `atb.*`, 18 `ai.*`,
  6 `data.*`, and 1 `dev.*`.
- Profile ids: `atb.profile.privileged_tool_action`,
  `atb.profile.rag_answer`, `atb.profile.data_export`,
  `atb.profile.policy_decision`, `atb.profile.human_override`,
  `atb.profile.background_automation`.

`schemas/event.v1.json` is byte-identical to clean remote `ec8d534`. The
corrected event count is not a contract change.

## Manual Prerequisites

These are Patrick-owned steps.

1. Re-add release secrets to the final public launch repo after it exists:
   `NPM_TOKEN`, `ATB_SIGNING_KEY_PEM`, `DOCKERHUB_USERNAME`,
   `DOCKERHUB_TOKEN`.
   Expected result: repository Actions secrets list contains those four names.
   Reversible: yes, by deleting or replacing secrets.
   Abort: do not run release workflows until all required names exist.

2. Do not add `GITHUB_TOKEN`. GitHub provides it per workflow run.
   Expected result: no repository secret named `GITHUB_TOKEN` is needed.
   Reversible: yes.
   Abort: if a workflow asks for a manual `GITHUB_TOKEN`, stop and inspect the
   workflow.

3. `DISCORD_WEBHOOK_URL` is optional. Add it only if notification jobs should
   send messages.
   Expected result: CI and ops notification steps can post when configured.
   Reversible: yes.
   Abort: leave absent if notifications are not wanted.

4. Do not add `PYPI_API_TOKEN`. PyPI publishing uses trusted publishing over
   OIDC.
   Expected result: no long-lived PyPI token is stored in GitHub.
   Reversible: yes.
   Abort: if PyPI publish fails as an untrusted publisher, fix PyPI trusted
   publisher configuration, not a token.

5. WORM secrets are not launch workflow requirements. Current workflows do not
   read `ATB_WORM_S3_ACCESS_KEY_ID` or `ATB_WORM_S3_SECRET_ACCESS_KEY`.
   Expected result: no WORM secret is needed for v1.15.0 launch workflows.
   Reversible: yes.
   Abort: if a new workflow introduces `atb push`, re-run workflow secret
   reconciliation first.

## PyPI Trusted Publisher Requirement

Source in workflow: `.github/workflows/release.yml`.

The release workflow publishes Python with
`pypa/gh-action-pypi-publish` in job `publish`, with `id-token: write`.
It uses no PyPI token.

PyPI trusted publishing is keyed to:

- PyPI project: `atb-sdk`.
- GitHub owner: `pcguest`.
- GitHub repository: `atb`.
- Workflow filename: `release.yml`.
- Workflow path in the repository: `.github/workflows/release.yml`.
- GitHub environment: any. The current workflow sets no environment.

The public PyPI project page shows past provenance, not the full authenticated
publisher-settings table. The current publisher settings must be checked by
Patrick in the PyPI UI.

Manual PyPI step before the release workflow runs:

1. Open PyPI project `atb-sdk`.
2. Go to `Publishing`.
3. Verify the existing GitHub Actions trusted publisher still lists:
   owner `pcguest`, repository `atb`, workflow `release.yml`, environment `Any`.
4. Leave it untouched if those fields match.
5. If the first release publish fails as an untrusted publisher, remove the stale
   entry and re-add: owner `pcguest`, repository `atb`, workflow `release.yml`,
   environment `Any`.

Expected result: the `release.yml` workflow in `pcguest/atb` can mint a
short-lived PyPI token for `atb-sdk`.

Reversible: yes. Remove or edit the trusted publisher in PyPI.

Abort: if the final repo name is not `pcguest/atb`, do not run the release
workflow. Update PyPI first.

Reference: PyPI docs say GitHub trusted publishers require the repository owner,
repository name, and workflow filename, with an optional environment. PyPI also
checks the GitHub repository owner's immutable id. The docs identify the repo
claim as the owner/name string and do not say PyPI checks GitHub's internal
repository id. A fresh `pcguest/atb` owned by the same GitHub account should
continue to match the existing publisher entry.

## Dependabot Inert Policy

Dependabot stays off through rename, fresh create, and public flip. Patrick may
turn it on after launch.

Recommended launch method: omit `.github/dependabot.yml` from the first pushed
tree. This avoids a missed `open-pull-requests-limit: 0` entry.

Initial launch diff:

```diff
diff --git a/.github/dependabot.yml b/.github/dependabot.yml
deleted file mode 100644
```

Settings before the first push to the fresh repo:

- Dependabot alerts: disabled.
- Dependabot security updates: disabled.
- Grouped security updates: disabled.

Expected result: no Dependabot branch, no Dependabot PR, and no `refs/pull/*`
can be created during launch.

Reversible: yes after launch. Re-add `.github/dependabot.yml` in a signed
post-launch commit and enable settings deliberately.

Abort: if any `refs/pull/*` appears in the fresh repo before public flip, stop.
Do not make that repo public.

## Naming Mechanic

The public repo name is `pcguest/atb`.

The public repo must be `pcguest/atb` because the frozen Go module path is
`github.com/pcguest/atb`. Tenon is the prose umbrella name only.

Current `pcguest/atb` resolves to the private archive repository now named
`pcguest/atb-legacy`. That archive must be renamed aside before a fresh
`pcguest/atb` can be created.

Suggested archive name: `pcguest/atb-legacy-archive`.

Renaming the archive aside frees the `atb` repository name for a fresh create.
It does not affect local commits, tags, bundles, or the clean local history. The
launch push source is the clean local tree or the verified clean bundle, never
the private `pcguest/tenon` staging remote and never the legacy archive.

## Module Path Gate

The frozen Go module path is `github.com/pcguest/atb`.

Read-only baseline on 2026-06-20:

- `gh repo view pcguest/atb` resolves to private `pcguest/atb-legacy`.
- `git ls-remote https://github.com/pcguest/atb.git HEAD` resolves, but to the
  private archive redirect, not to the fresh launch repo.

This baseline is expected before the launch create. It proves why the archive
must move aside.

This is a hard launch gate.

Expected result after the fresh public repo exists and `v1.15.0` is tagged:
an unauthenticated external
`GOTOOLCHAIN=go1.26.4 go install github.com/pcguest/atb/cmd/atb@v1.15.0`
resolves to the fresh public launch history, builds, and the binary reports
`1.15.0`.

Abort: if `github.com/pcguest/atb` resolves to private `pcguest/atb-legacy`, do
not tag, release, publish, or flip public.

## Launch Sequence

### 1. Final Local Verification

Owner: Codex, on Patrick's word.

Commands:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'git status --short --branch'
/bin/bash -lc 'go version'
/bin/bash -lc 'GOCACHE=/tmp/atb-gocache-launch GOTOOLCHAIN=go1.26.4 make test-golden'
/bin/bash -lc 'GOCACHE=/tmp/atb-gocache-launch GOTOOLCHAIN=go1.26.4 make hygiene-quick'
```

Expected result:

- Branch is `release/v1.15.0`.
- Go reports `go1.26.4`.
- `make test-golden` passes.
- `make hygiene-quick` passes.
- Working tree remains clean.

Reversible: yes. This is local validation only.

Abort: any dirty tree, failed gate, wrong Go version, or unexpected generated
diff stops launch.

### 2. Record Module-Path Baseline

Owner: Codex, on Patrick's word.

Commands:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'gh repo view pcguest/atb --json nameWithOwner,visibility,isPrivate,defaultBranchRef,pushedAt'
/bin/bash -lc 'git ls-remote https://github.com/pcguest/atb.git HEAD'
/bin/bash -lc 'tmp="$(mktemp -d)"; GOBIN="$tmp/bin" GOMODCACHE="$tmp/mod" GOCACHE="$tmp/cache" GOTOOLCHAIN=go1.26.4 GIT_TERMINAL_PROMPT=0 go install github.com/pcguest/atb/cmd/atb@latest'
```

Expected result:

- Metadata resolves to private `pcguest/atb-legacy`, or `go install` fails
  because the path is still private.
- This is the known pre-launch baseline.

Reversible: yes. This is read-only apart from temporary Go caches.

Abort: if `github.com/pcguest/atb` unexpectedly resolves to a public repository
not controlled by this launch, stop.

### 3. Rename Legacy Archive `pcguest/atb-legacy` Aside

Owner: Patrick presses the GitHub button, or Codex runs it only after explicit
approval.

Command option:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'gh repo rename atb-legacy-archive --repo pcguest/atb-legacy'
```

Expected result:

- Legacy archive becomes `pcguest/atb-legacy-archive`.
- The `pcguest/atb` name is free.
- Legacy archive remains private.
- Current staging `pcguest/tenon` remains private and irrelevant to the launch
  push.

Reversible: yes until a new repo takes the `atb` name. Rename it back if no
fresh repo has been created.

Abort: if GitHub reports a name conflict or visibility change, stop.

### 4. Create Fresh Private `pcguest/atb`

Owner: Patrick presses the GitHub button, or Codex runs it only after explicit
approval.

Command option:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'gh repo create pcguest/atb --private --disable-wiki'
```

Expected result:

- New private repo exists at `pcguest/atb`.
- It is not a fork.
- It is not created from a template.
- It has no commits and no PRs.

Reversible: yes before any public flip, by deleting the empty fresh repo after
Patrick approval.

Abort: if GitHub creates it from a template, as a fork, public, or with default
content, delete it before any push and recreate clean.

### 5. Disable Dependabot Settings Before Any Push

Owner: Patrick, because these are GitHub settings.

Manual path:

- Open `pcguest/atb`.
- Settings.
- Code security and analysis.
- Disable Dependabot alerts.
- Disable Dependabot security updates.
- Disable grouped security updates.

Expected result: Dependabot settings cannot open PRs before launch.

Reversible: yes after launch.

Abort: if settings cannot be disabled, do not push.

### 6. Prepare Dependabot-Omitted Launch Source

Owner: Codex, on Patrick's word.

Use a private temporary launch clone. Do not mutate the staging remote.

Command sketch:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'git status --short --branch'
/bin/bash -lc 'rm -rf /tmp/atb-launch-work'
/bin/bash -lc 'git clone --no-local /Users/paddyguest/atb /tmp/atb-launch-work'
cd /tmp/atb-launch-work
/bin/bash -lc 'git checkout main'
/bin/bash -lc 'git rm .github/dependabot.yml'
/bin/bash -lc 'git commit -S -m "chore: keep Dependabot inert for launch"'
/bin/bash -lc 'git checkout release/v1.15.0'
/bin/bash -lc 'git rm .github/dependabot.yml'
/bin/bash -lc 'git commit -S -m "chore: keep Dependabot inert for launch"'
```

Expected result:

- Temporary launch clone has signed branch-tip commits on `main` and
  `release/v1.15.0` that delete `.github/dependabot.yml`.
- `/Users/paddyguest/atb` remains unchanged.
- The first pushed default branch tip has no active Dependabot config.

Reversible: yes. Delete `/tmp/atb-launch-work`.

Abort: if any frozen identifier changes, any commit is unsigned, or the signing
identity is not Patrick Guest `<patrickcguest@proton.me>`, stop.

Note: this is the only planned launch-time source edit before the first push. It
touches only `.github/dependabot.yml`, which is not a frozen identifier.

### 7. Push Clean History to Fresh Private `pcguest/atb`

Owner: Codex, on Patrick's word.

Command shape:

```bash
cd /tmp/atb-launch-work
/bin/bash -lc 'git remote add launch https://github.com/pcguest/atb.git'
/bin/bash -lc 'git push launch main:main'
/bin/bash -lc 'git push launch release/v1.15.0:release/v1.15.0'
/bin/bash -lc 'git push launch --tags'
```

Expected result:

- Fresh repo has `main`, `release/v1.15.0`, and 22 signed tags.
- No Dependabot branches.
- No PRs.

Reversible: yes while private, by deleting and recreating the fresh repo after
Patrick approval. Do not force-push a public repo.

Abort: any push rejection, unexpected branch, or Dependabot branch stops launch.

Important: source is the clean local repo or verified clean bundle. Do not push
from `pcguest/tenon` or `pcguest/atb-legacy-archive`. The temporary launch clone
is derived from the clean local repo, not from any remote.

### 8. Mirror-Verify Fresh Repo

Owner: Codex, on Patrick's word.

Commands:

```bash
cd /tmp
/bin/bash -lc 'git clone --mirror https://github.com/pcguest/atb.git atb-launch-verify.git'
cd /tmp/atb-launch-verify.git
/bin/bash -lc 'git show-ref refs/pull || true'
/bin/bash -lc 'du -sh objects/pack'
/bin/bash -lc 'git rev-list --objects --all | rg -i "gosec|securego" || true'
/bin/bash -lc 'git log main release/v1.15.0 --tags --format="%an <%ae>%n%cn <%ce>" | sort -u'
/bin/bash -lc 'git for-each-ref --format="%(taggername) <%(taggeremail)>" refs/tags | sort -u'
/bin/bash -lc 'for t in $(git tag --list); do git tag -v "$t" >/dev/null || exit 1; done; echo all-tags-good'
```

Expected result:

- No `refs/pull/*`.
- Pack is around 4 MiB.
- No `gosec` or `securego` object path.
- Author, committer, and tagger identity is Patrick Guest
  `<patrickcguest@proton.me>`.
- All 22 tags verify.

Reversible: yes while private, by deleting and recreating the fresh repo after
Patrick approval.

Abort: any PR ref, bot identity, sign-off trailer, missing tag, bad tag
signature, large pack, or gosec residue stops launch.

### 9. Re-Add Required Secrets

Owner: Patrick.

Manual step:

- Add `NPM_TOKEN`.
- Add `ATB_SIGNING_KEY_PEM`.
- Add `DOCKERHUB_USERNAME`.
- Add `DOCKERHUB_TOKEN`.
- Optionally add `DISCORD_WEBHOOK_URL`.

Expected result: release and Docker workflows have the secrets they actually
read.

Reversible: yes. Delete or replace secrets.

Abort: do not tag or run release workflows until `NPM_TOKEN` and
`ATB_SIGNING_KEY_PEM` exist. Do not run Docker publish without Docker Hub
secrets.

### 10. Verify PyPI Trusted Publisher

Owner: Patrick.

Manual step:

- PyPI project `atb-sdk`.
- Publishing.
- Confirm the existing GitHub Actions trusted publisher lists:
  owner `pcguest`, repository `atb`, workflow `release.yml`, environment `Any`.
- Do not remove and re-add when those fields match.

Expected result: PyPI trusts `.github/workflows/release.yml` in `pcguest/atb`.

Reversible: yes. Edit or remove trusted publisher.

Abort: if PyPI points at any other owner, repository, workflow, or environment,
do not run the release workflow.
If the PyPI UI cannot show the final publisher entry clearly, stop.
Fallback: if the first release publish fails as an untrusted publisher, remove
the publisher entry and re-add: owner `pcguest`, repository `atb`, workflow
`release.yml`, environment `Any`.

### 11. Enable Private Vulnerability Reporting

Owner: Patrick.

Manual step:

- In `pcguest/atb`, enable private vulnerability reporting.
- Later, in `pcguest/mortise`, enable private vulnerability reporting.

Expected result:

- `https://github.com/pcguest/atb/security/advisories/new` is the ATB private
  report path.
- `https://github.com/pcguest/mortise/security/advisories/new` is the Mortise
  private report path.

Reversible: yes while private. After public launch, disabling it would break the
published security intake path.

Abort: do not flip public until PVR is enabled.

### 12. Confirm Mortise Repository Name

Owner: Patrick, or Codex on explicit approval.

The rename to `pcguest/mortise` has already happened; this step is now a
verification, not a mutation.

Command option:

```bash
gh repo view pcguest/mortise --json name,visibility
```

Expected result:

- Name is `mortise`; repo remains private.

Abort: if the repo is missing, renamed elsewhere, or visibility changed, stop.

### 13. Push Mortise Prose Rename

Owner: Codex, on Patrick's word.

Commands:

```bash
cd /Users/paddyguest/mortise
/bin/bash -lc 'git status --short --branch'
/bin/bash -lc 'git remote set-url origin https://github.com/pcguest/mortise.git'
/bin/bash -lc 'git push -u origin site/marketing-front'
```

Expected result:

- `38c365e` is on private `pcguest/mortise`.
- Working tree remains clean.

Reversible: yes while private, by deleting the pushed branch.

Abort: if local branch is dirty or remote is not private `pcguest/mortise`, stop.

### 14. Update ATB README Links to Mortise URL

Owner: Codex, on Patrick's word.

Command shape:

```bash
cd /tmp/atb-launch-work
/bin/bash -lc 'rg -n "mortise|Mortise|Mortise" README.md docs'
```

Then edit prose links only in `/tmp/atb-launch-work`, run the ATB hygiene
gate, and create one signed commit.

Expected result:

- Public prose points to `https://github.com/pcguest/mortise`.
- No ATB doc link, module reference, or URL depends on a repository named
  `tenon`.
- No frozen identifier changes.
- Signed commit by Patrick only.

Reversible: yes before push, by amending or replacing the local commit. After
push, use a follow-up signed commit.

Abort: if any module path, package name, event type, schema id, CLI name, or
bundle extension changes, stop.

### 15. Push Final ATB Launch Commit

Owner: Codex, on Patrick's word.

Commands:

```bash
cd /tmp/atb-launch-work
/bin/bash -lc 'git status --short --branch'
/bin/bash -lc 'git push launch release/v1.15.0:release/v1.15.0'
```

Expected result: final signed launch prose commit is in private fresh
`pcguest/atb`.

Reversible: yes while private if the repo is deleted and recreated. Prefer a
follow-up signed commit over rewriting after this point.

Abort: if `launch` does not point at the fresh PR-ref-free repo, or if
`/tmp/atb-launch-work` is not descended from the clean local repo, stop.

### 16. Flip Visibility

Recommended order: `pcguest/atb` first, then `pcguest/mortise`.

Owner: Patrick.

Manual step:

- Flip `pcguest/atb` public.
- Verify clone and install visibility.
- Flip `pcguest/mortise` public.

Expected result:

- Public ATB core is available first.
- Mortise prose links have a public target after its flip.

Reversible: limited. A public repository can be made private again, but the
history may already have been fetched.

Abort: if either repo has unexpected refs, wrong visibility, missing PVR, or
wrong trusted publisher state, do not flip.

### 17. Disable Tag-Triggered Publish Workflows

Owner: Patrick presses the GitHub button, or Codex runs it only after explicit
approval.

Reason: `.github/workflows/release.yml` and
`.github/workflows/docker-publish.yml` both run on `v*.*.*` tag pushes. The tag
must exist publicly before the `go install ...@v1.15.0` resolution proof can run,
but publication must not happen before that proof passes.

Commands:

```bash
/bin/bash -lc 'gh workflow disable release.yml --repo pcguest/atb'
/bin/bash -lc 'gh workflow disable docker-publish.yml --repo pcguest/atb'
```

Expected result:

- `release.yml` will not publish PyPI or npm on tag push.
- `docker-publish.yml` will not publish Docker images on tag push.
- Non-publishing validation workflows may still run.

Reversible: yes. Re-enable workflows after the module-resolution proof passes.

Abort: if either publish workflow cannot be disabled, do not tag.

### 18. Tag `v1.15.0` Once on Final Public History

Owner: Codex creates the signed tag only on Patrick's word. Patrick approves the
push.

Commands:

```bash
cd /tmp/atb-launch-work
/bin/bash -lc 'git fetch launch --tags'
/bin/bash -lc 'git checkout release/v1.15.0'
/bin/bash -lc 'git tag -s v1.15.0 -m "ATB v1.15.0"'
/bin/bash -lc 'git push launch v1.15.0'
```

Expected result:

- `v1.15.0` exists once on the final public repo.
- It points at the final launch commit.
- It is signed.
- Publish workflows remain disabled during this tag push.

Reversible: no in practice. If wrong, ship `v1.15.1`. Never force-move
`v1.15.0`.

Abort: if `v1.15.0` already exists anywhere on the final repo, stop and inspect.

### 19. Resolve Go Module Path at `v1.15.0`

Owner: Codex, on Patrick's word.

This is the first public module-resolution check after the tag exists.

Commands:

```bash
/bin/bash -lc 'tmp="$(mktemp -d)"; GOBIN="$tmp/bin" GOMODCACHE="$tmp/mod" GOCACHE="$tmp/cache" GOTOOLCHAIN=go1.26.4 GIT_TERMINAL_PROMPT=0 go install github.com/pcguest/atb/cmd/atb@v1.15.0; "$tmp/bin/atb" version'
```

Expected result:

- `go install` resolves from public `pcguest/atb`.
- The binary builds from tag `v1.15.0`.
- The binary reports `1.15.0`.

Reversible: validation only.

Abort: if `go install` cannot resolve `github.com/pcguest/atb`, or if the binary
does not report `1.15.0`, stop. Do not release, publish, or deploy.

### 20. Cut GitHub Release and Run Release Workflow

Owner: Patrick presses release buttons, or Codex uses GitHub CLI only after
explicit approval.

Command option:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'gh workflow enable release.yml --repo pcguest/atb'
/bin/bash -lc 'gh release create v1.15.0 --repo pcguest/atb --verify-tag --title "ATB v1.15.0" --notes-file RELEASE-REVIEW-v1.15.0.md'
/bin/bash -lc 'gh workflow run release.yml --repo pcguest/atb --ref v1.15.0'
```

Expected result:

- GitHub Release exists for `v1.15.0`.
- `.github/workflows/release.yml` runs deliberately against the tag.
- GitHub Release assets, PyPI `atb-sdk`, and npm `@pcguest/atb-sdk` publish
  through the workflow.

Reversible: partly. Release notes can be edited. Published PyPI and npm versions
are immutable in practice.

Abort: if PyPI trusted publisher or secrets are missing, stop before creating
the release or triggering publish.

### 21. Docker Publish

Owner: Patrick decides whether Docker is in the v1.15.0 launch round.

Command option:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'gh workflow enable docker-publish.yml --repo pcguest/atb'
/bin/bash -lc 'gh workflow run docker-publish.yml --repo pcguest/atb --ref v1.15.0 -f tag=v1.15.0'
```

Expected result: Docker images publish to `docker.io/$DOCKERHUB_USERNAME/atb`.

Reversible: limited. Docker tags can be moved, but consumers may have pulled
them.

Abort: if Docker Hub secrets are absent or Docker is out of launch scope, skip.

### 22. Deploy Mortise Site

Owner: Patrick presses the deploy button, or Codex runs it only after explicit
approval.

Local preview command:

```bash
cd /Users/paddyguest/mortise
/bin/bash -lc 'python3 -m http.server --directory site 8000'
```

Expected result: Mortise site renders locally before deploy.

Deploy command: depends on chosen host. Record the exact host command before
running it.

Reversible: yes if the host keeps prior artefacts or supports rollback.

Abort: if local preview shows old Mortise branding, broken links, or unsupported
claims, stop.

### 23. Post-Launch Verification

Owner: Codex, on Patrick's word.

Commands:

```bash
/bin/bash -lc 'rm -rf /tmp/atb-post-launch-verify'
/bin/bash -lc 'mkdir -p /tmp/atb-post-launch-verify'
/bin/bash -lc 'cd /tmp/atb-post-launch-verify; GOBIN=/tmp/atb-post-launch-verify/bin GOTOOLCHAIN=go1.26.4 go install github.com/pcguest/atb/cmd/atb@v1.15.0'
/bin/bash -lc 'npm view @pcguest/atb-sdk@1.15.0 version'
/bin/bash -lc 'python3 -m pip index versions atb-sdk'
/bin/bash -lc 'cd /tmp/atb-post-launch-verify; python3 -m venv venv'
/bin/bash -lc 'cd /tmp/atb-post-launch-verify; . venv/bin/activate; python -m pip install atb-sdk==1.15.0'
```

Quickstart integrity check:

```bash
cd /tmp
/bin/bash -lc '/tmp/atb-post-launch-verify/bin/atb bundle new --out intact.atb'
/bin/bash -lc '/tmp/atb-post-launch-verify/bin/atb verify --bundle intact.atb'
/bin/bash -lc 'cp intact.atb tampered.atb'
/bin/bash -lc 'printf "\n" >> tampered.atb'
/bin/bash -lc '/tmp/atb-post-launch-verify/bin/atb verify --bundle tampered.atb; test "$?" -eq 2'
```

Expected result:

- `go install` succeeds from public repo.
- npm reports `1.15.0`.
- pip installs `atb-sdk==1.15.0`.
- Intact verify exits `0`.
- Tamper verify exits `2`.

Reversible: validation only.

Abort: any failed install or wrong exit code stops post-launch sign-off.

### 24. Re-Enable Dependabot Deliberately

Owner: Codex prepares the signed post-launch commit only on Patrick's word.
Patrick decides whether to enable settings.

Command shape:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'git checkout main'
/bin/bash -lc 'git pull --ff-only launch main'
```

Then restore `.github/dependabot.yml`, commit signed, push, and enable the
chosen Dependabot settings.

Expected result:

- Dependabot PRs after this point are normal public maintenance PRs.
- The launch namespace remains PR-ref-free up to the public flip.

Reversible: yes by a signed follow-up commit and settings change.

Abort: if Dependabot opens PRs before this signed post-launch commit, stop and
inspect settings.

## Final Gate Before Any Irreversible Step

Before each irreversible step, repeat:

```bash
cd /Users/paddyguest/atb
/bin/bash -lc 'git status --short --branch'
cd /Users/paddyguest/mortise
/bin/bash -lc 'git status --short --branch'
```

Expected result:

- ATB state matches the current launch step.
- Mortise state matches the current launch step.
- No unrelated dirty files.

Abort: blank output where a value is expected, dirty files, wrong branch, wrong
remote, or ambiguity stops the run.
