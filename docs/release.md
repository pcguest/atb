# Release runbook

Steps to cut a new ATB release. Replace `<NEW>` with the target version
(e.g. `1.10.0`) and `<OLD>` with the version being superseded.

## 1. Consolidate CHANGELOG

Move the `[Unreleased]` content into a new `## [v<NEW>] - <date>` block.
Leave `[Unreleased]` with only `<!-- No unreleased changes. -->`.

```bash
$EDITOR CHANGELOG.md
```

## 2. Bump version strings

Each location must be updated manually. `scripts/check-versions.sh` treats
`cmd/atb/main.go` as the source of truth and checks every other location
against it.

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
sed -i '' "s/Current release: \[\`v<OLD>\`\]/Current release: [\`v<NEW>\`]/" README.md
```

## 4. Run all test suites

```bash
go build ./...
go test -race ./...
cd sdk/typescript && npm run typecheck && npm test && cd ../..
cd sdk/python && python -m pytest && cd ../..
```

## 5. Commit and push

```bash
git add CHANGELOG.md
git commit -m "chore(changelog): write v<NEW> block"

git add cmd/atb/main.go sdk/python/pyproject.toml sdk/python/atb/__init__.py \
        sdk/typescript/package.json sdk/typescript/package-lock.json \
        web/package.json web/package-lock.json
git commit -m "chore(release): bump version strings to <NEW>"

git add README.md
git commit -m "docs(readme): bump badge and current release to v<NEW>"

git push origin main
```

## 6. Tag and wait for CI

```bash
git tag v<NEW>
git push origin v<NEW>
```

Wait for the `version-gate` workflow to succeed before creating the release.

```bash
gh run list --repo pcguest/atb --workflow version-gate.yml --limit 3
```

## 7. Create GitHub release

```bash
gh release create v<NEW> \
  --title "v<NEW>" \
  --notes "$(awk '/## \[v<NEW>\]/{found=1; next} found && /^## \[/{exit} found{print}' CHANGELOG.md)" \
  --latest
```

Release notes are pulled verbatim from the CHANGELOG block.

## 8. Close tracking issues

Comment on any issues resolved by the release, then close them.

```bash
gh issue comment <N> --repo pcguest/atb --body "Closed by v<NEW>. <brief note>."
gh issue close <N> --repo pcguest/atb
```
