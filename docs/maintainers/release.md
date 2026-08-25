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
- `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`: Docker publication credentials.

PyPI uses GitHub OIDC trusted publishing, not a repository token. Optional S3
WORM automation uses `ATB_WORM_S3_ACCESS_KEY_ID` and
`ATB_WORM_S3_SECRET_ACCESS_KEY`, scoped to the minimum required object-write
and retention operations.

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

```bash
go build ./...
go test -race ./...
cd sdk/typescript && npm run typecheck && npm test && cd ../..
cd sdk/python && python -m pytest && cd ../..
```

Run the release preflight before opening the release PR. Contributors without
a running Docker daemon may skip the local Docker smoke build; the
`docker-publish.yml` workflow builds and publishes the image on tag push.

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

Wait for the tag-triggered workflows to succeed. Do not create a GitHub Release
manually: the `Release` workflow owns the draft, registry publication, retained
evidence bundle, and final transition out of draft.

The `Release` workflow contains its own gold gate (`make gate-gold-release`,
including the repository's ≥80% aggregate Go coverage threshold). Tag pushes
also trigger `Docker Publish`. Both workflows must be green before announcing
the release. A manual dispatch of `release.yml` builds and verifies artefacts
but cannot publish without a tag.

Publication is retry-safe: publish already verified immutable artefacts, append
all capture evidence before signing the retained evidence bundle, and make the
GitHub Release public only after registry publication and final verification.

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
the incomplete steps. If the artefacts must change, cut a new version.

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
