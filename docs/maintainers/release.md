# Release runbook

Steps to cut a new ATB release. Replace `<NEW>` with the target version
(e.g. `1.10.0`) and `<OLD>` with the version being superseded.

For the standing maintainer workflow and versioning rules, see
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) and
[`VERSIONING.md`](../../VERSIONING.md).

## Release provenance and credentials

Release-line commits and tags use the maintainer identity and signing-key
fingerprint recorded in the protected release configuration. A release must
not move an existing public tag or replace immutable registry bytes. Before
tagging, verify commit and tag signatures, search commit bodies for unintended
authorship trailers, run `make gate-gold-release`, and confirm version
agreement across the CLI, SDKs, and viewer.

The release workflow uses secret names only; values never belong in source or
logs:

- `GITHUB_TOKEN`: job-scoped GitHub release automation token.
- `NPM_TOKEN`: package-scoped `@pcguest/atb-sdk` publish token with the minimum
  required write access and non-interactive 2FA policy.
- `ATB_SIGNING_KEY_PEM`: protected PKCS#8 Ed25519 key for the final release
  evidence bundle. See [key management](../evidence/key-management.md).
- `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`: Docker publication credentials,
  used only by the protected `docker-release` publish job.

PyPI uses GitHub OIDC trusted publishing, not a repository token. Optional S3
WORM automation uses `ATB_WORM_S3_ACCESS_KEY_ID` and
`ATB_WORM_S3_SECRET_ACCESS_KEY`, scoped to the minimum required object-write
and retention operations.

## Release gate

Publication is founder-controlled and fails closed. A release cannot publish
until all of the following repository settings exist:

- Repository variable `RELEASE_GATE_ENABLED` set to `true`.
- A protected `release` environment, used by the `Release` workflow's publish
  job, with required reviewers.
- A protected `docker-release` environment, used by the `Docker Publish`
  workflow's publish job, with required reviewers.

Approving the environment run is the approval to publish. This document does
not assert that these settings are configured: they are repository
configuration, not tracked files. Until they are in place, tag pushes and
manual dispatches stop at the gate and publish nothing. The gate also rejects
tags that are not stable SemVer.

Environment protection is a deliberate-action control, not an authorization
boundary. A required reviewer ensures that a person approves that specific run;
it does not prevent a repository writer from editing these workflow files or
the environment settings. Treat the gate as evidence of intent to publish, not
as proof that only trusted code can reach the registry.

Recommended hardening, not yet verified: move `DOCKERHUB_USERNAME`,
`DOCKERHUB_TOKEN`, `NPM_TOKEN`, and `ATB_SIGNING_KEY_PEM` from repository
secrets to environment-scoped secrets on the `release` and `docker-release`
environments, so a workflow change cannot read them without an approved run,
and configure those environments to prevent self-review of an approval.
Neither the move nor the self-review setting is verified by this document or
by repository tests.

### Docker publication is manual

`docker-publish.yml` has no tag trigger. Publish an image by dispatching the
workflow with three required inputs:

- `confirm`: must be exactly `PUBLISH`.
- `tag`: a stable SemVer tag, `vMAJOR.MINOR.PATCH`.
- `expected_sha`: the full 40-character lowercase hex commit SHA that the tag
  must resolve to.

The gate rejects any repository other than `pcguest/atb`, a wrong confirmation
string, a malformed tag or SHA, a `RELEASE_GATE_ENABLED` value other than
`true`, and any tag whose commit does not match `expected_sha`. The published
manifest carries `vMAJOR.MINOR.PATCH` and `vMAJOR.MINOR` only. Promoting
`:latest` is a separate, explicitly approved operation and never happens
automatically.

The unprivileged `build` jobs have no registry credentials and never push;
they validate the tag locally. Publication rebuilds the multi-architecture
image and pushes it only inside the protected `docker-release` environment,
after revalidating both the checked-out `HEAD` and the tag against
`expected_sha` immediately before the registry login. After the push, the
workflow records each published manifest digest and re-pulls each platform by
that digest to re-check the OCI version label and the embedded ELF architecture.
It executes the `atb version` output only on the publish runner's native
platform (`linux/amd64`), because that job does not install QEMU/binfmt
emulation for `linux/arm64`.

Publication is serialized per release tag (`concurrency.group` is
`docker-publish-<tag>`) with `cancel-in-progress: false`, so a second dispatch
for the same tag waits instead of racing the first. Before the build and push,
the publish job inspects the immutable `vMAJOR.MINOR.PATCH` manifest with
`docker buildx imagetools inspect` and fails if it already exists: the
immutable version tag is never overwritten, and replacing it requires a
separate, explicitly approved operation. Only an absent manifest proceeds; any
other inspection failure fails closed. The documented mutable `vMAJOR.MINOR`
tag is not checked. The exact tag is logged before the push, and the manifest
digest is logged immediately after the push.

## 1. Consolidate CHANGELOG

Move the `[Unreleased]` content into a new `## [v<NEW>] - <date>` block.
Leave `[Unreleased]` with only `<!-- No unreleased changes. -->`.

```bash
$EDITOR CHANGELOG.md
```

## 2. Bump version strings

Each location must be updated manually. `scripts/check-versions.sh` uses
the latest git tag as the release source of truth and checks the tracked
version locations against it.

### Go CLI

```bash
sed -i '' 's/version\s*= "<OLD>"/version          = "<NEW>"/' cmd/atb/main.go
go build ./...   # confirm it compiles
```

### Python SDK

```bash
sed -i '' 's/^version = "<OLD>"/version = "<NEW>"/' sdk/python/pyproject.toml
sed -i '' 's/__version__ = "<OLD>"/__version__ = "<NEW>"/' sdk/python/atb/__init__.py
```

### TypeScript SDK

```bash
cd sdk/typescript
npm version <NEW> --no-git-tag-version
npm install --package-lock-only   # updates package-lock.json without touching node_modules
cd ../..
```

### Web

```bash
cd web
npm version <NEW> --no-git-tag-version
npm install --package-lock-only
cd ..
```

### Verify agreement

```bash
ATB_SKIP_TAG_CHECK=1 bash scripts/check-versions.sh   # must exit 0
```

## 3. README badge

```bash
sed -i '' 's/release-v<OLD>-blue/release-v<NEW>-blue/' README.md
sed -i '' "s/Source version: \[\`v<OLD>\`\]/Source version: [\`v<NEW>\`]/" README.md
```

## 4. Run all test suites

For the authoritative clean-clone founder and package-consumer sequence, use
[local release-candidate acceptance](./local-acceptance.md). It covers the
isolated Python and Node environments, flagship incident, tamper failures,
Firefox viewer, packed SDKs, and installed CLI path.

```bash
go build ./...
go test -race ./...
cd sdk/typescript && npm run typecheck && npm test && cd ../..
cd sdk/python && python -m pytest && cd ../..
```

Run the release preflight before opening the release PR. Contributors without
a running Docker daemon may skip the local Docker smoke build; the
`docker-publish.yml` workflow validates the image in unprivileged build jobs
and pushes only from the protected publish job after release-gate approval.
The local Docker smoke requires Docker Buildx and either native execution or
binfmt/emulation for both `linux/amd64` and `linux/arm64`. It independently
checks the embedded ELF architecture because an emulated runtime pass does not
prove that the executable matches the image platform. If the local Docker
smoke is skipped, both native hosted platform jobs remain mandatory.

```bash
SKIP_DOCKER=1 scripts/release-check.sh
```

## 5. Commit and open a release PR

```bash
git switch -c release/v<NEW>

git add CHANGELOG.md
git commit -m "chore(changelog): write v<NEW> block"

git add cmd/atb/main.go sdk/python/pyproject.toml sdk/python/atb/__init__.py \
        sdk/typescript/package.json sdk/typescript/package-lock.json \
        web/package.json web/package-lock.json
git commit -m "chore(release): bump version strings to <NEW>"

git add README.md
git commit -m "docs(readme): bump badge and current release to v<NEW>"

git push -u origin release/v<NEW>

gh pr create \
  --title "chore(release): prepare v<NEW>" \
  --body "Prepare v<NEW> for merge to main and tagging."
```

Merge the release PR to `main` after CI passes.

## 6. Tag from `main` and wait for CI

```bash
git switch main
git pull origin main
git tag -a v<NEW> -m "v<NEW>"
git push origin v<NEW>
```

Wait for the tag-triggered `Release` workflow to succeed. Do not create a
GitHub Release manually: the `Release` workflow owns the draft, registry
publication, retained evidence bundle, and final transition out of draft.

The `Release` workflow contains its own gold gate (`make gate-gold-release`,
including the repository's ≥80% aggregate Go coverage threshold). A tag push
triggers only `Release`; `Docker Publish` has no tag trigger and does not run
on a tag push. Dispatch it separately with `confirm=PUBLISH`, the tag, and its
commit SHA. Both must be green before announcing the release. A manual dispatch
of `release.yml` builds and verifies artefacts but cannot publish without a
tag.

Publication stages immutable artefacts in a fixed order: publish already
verified artefacts, append all capture evidence before signing the retained
evidence bundle, and make the GitHub Release public only after registry
publication and final verification. Publication is not retried automatically:
if it stops partway, resuming the remaining steps is a manual operator action
(see step 7).

## 7. Verify automated publication

```bash
gh run list --repo pcguest/atb --workflow release.yml --limit 3
gh run list --repo pcguest/atb --workflow docker-publish.yml --limit 3
gh release view v<NEW> --repo pcguest/atb
```

The `Release` workflow first creates a draft, publishes the already verified
Python and npm artifacts, signs and attaches the final ATB release-evidence
bundle, and only then makes the GitHub Release public. If any step fails, leave
the draft and registry state intact, record which immutable registries accepted
the version, fix the workflow without changing published bytes, and resume only
the incomplete steps. Resuming is manual: re-run the failed jobs, or dispatch
`docker-publish.yml` again for the image, after the workflow fix; GitHub does
not automatically retry a partial registry publication. If the artefacts must
change, cut a new version.

Registry publication is serialized per ref: the `Release` workflow keeps its
workflow-level `release-<ref>` lock for build and validation, and the `publish`
job adds a separate `atb-release-<ref>` lock with `cancel-in-progress: false`.
A second run for the same ref waits rather than cancelling an in-progress
publication.

`Release` also accepts two `workflow_dispatch` boolean inputs, `skip_pypi` and
`skip_npm`, both defaulting to `false`. Tag pushes cannot set them. Use a skip
only to resume a partial publication after manually verifying that the exact
version and the same artefact hashes already exist in that registry: the input
removes that registry's publish step and its pre-publication existence check
for that dispatch. With both `false` (the default, and always on a tag push),
both registries are checked and published as before. Do not set a skip to work
around a version collision.

## 8. Close tracking issues

Comment on any issues resolved by the release, then close them.

```bash
gh issue comment <N> --repo pcguest/atb --body "Closed by v<NEW>. <brief note>."
gh issue close <N> --repo pcguest/atb
```

## Recovery exercise

Quarterly, restore the project from a fresh clone, run the full gate, restore
only the named CI credentials from the secret manager, verify PyPI OIDC trust,
and dry-run the build portion of the release workflow. Record date, operator,
result, and blockers. The pragmatic targets are a restore time under two hours
and a recovery point at the latest pushed commit or tag.
