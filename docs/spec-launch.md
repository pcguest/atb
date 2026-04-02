# ATB Launch Specification

## Goal

Ship ATB v1.0.0 with deterministic, secure, automated release pipelines for:

- PyPI
- npm
- Docker Hub
- GitHub Releases

Target install experience:

- `go install github.com/pcguest/atb/cmd/atb@latest`
- `docker pull <org>/atb:<version>`

## Versioning Strategy

ATB follows Semantic Versioning (`MAJOR.MINOR.PATCH`) across CLI, SDKs, and Docker tags.

### Major

Bump MAJOR for any compatibility break:

- Breaking CLI command/flag behavior
- Breaking event schema structure or semantics
- Breaking SOC2/GDPR export structures
- Breaking public SDK API contracts

### Minor

Bump MINOR for backward-compatible feature additions:

- New commands, integrations, formats, middleware, or dashboard capabilities

### Patch

Bump PATCH for backward-compatible fixes:

- Bugfixes, security patches, and docs/tooling updates

### Deprecation Policy

- Mark deprecations in one MINOR release
- Keep deprecated behavior through at least one additional MINOR
- Remove deprecated behavior only in next MAJOR
- Current v1.x stance: Python and TypeScript ship SDKs only. Their `atb` entrypoints are compatibility stubs pending removal in the next MAJOR release.

## Release Checklist (v1.0.0)

1. Freeze release branch; merge only release-critical fixes.
2. Confirm versions and changelog are updated.
3. Run local preflight (`scripts/release-check.sh`).
4. Create annotated tag `v1.0.0`.
5. Push tag and monitor release and Docker publish workflows.
6. Verify published artifacts:
   - GitHub release binaries + checksums
   - PyPI package
   - npm package
   - Docker image
7. Run post-release smoke checks in clean environments.
8. Publish release announcement.

## Packaging Details

### Python

- Build backend: `setuptools.build_meta`
- Console script stub: `atb = atb_cli_stub:main`
- Metadata requirements:
  - License: MIT
  - Production classifier
  - Stable project URLs (homepage, repository, docs)
- Build commands:
  - `python -m build`
  - `twine check dist/*`

### npm / TypeScript

- Package must define:
  - compatibility stub mapping: `"atb": "./bin/atb"`
  - `types`, `exports`, `repository.directory`
  - `files` whitelist for deterministic package contents
- Publish command:
  - `npm publish --access public`

### Docker

- Multi-stage build:
  1. Go binary stage (`CGO_ENABLED=0`, `-trimpath`)
  2. Web build stage (`npm ci && npm run build`)
  3. Distroless/alpine runtime stage
- Runtime defaults:
  - `ENTRYPOINT ["/app/atb"]`
  - `EXPOSE 8080`
  - `VOLUME ["/data"]`

## Reproducibility Requirements

- Pin toolchain versions in CI
- Use lockfiles and `npm ci`
- Use deterministic build flags for Go binaries
- Produce release checksums for all binaries
- Pin Docker base images to versioned tags or digests

## Security & Secrets

- Do not hardcode credentials in repo/workflows
- Use GitHub Secrets/OIDC:
  - PyPI via trusted publishing (OIDC)
  - npm auth via `NPM_TOKEN`
  - Docker Hub auth via `DOCKERHUB_USERNAME` + `DOCKERHUB_TOKEN` in `docker-publish.yml`
- Optionally sign artifacts/images in future phase (cosign)

## CI/CD Flow

Trigger on tags matching `v*.*.*`.

`release.yml` handles GitHub Releases, PyPI, and npm:

1. **Validate**
   - Lockfile consistency checks
   - Full Go/Python/TypeScript test matrix
   - Version/tag consistency checks
2. **Build**
   - Multi-OS CLI binaries
   - Python wheel/sdist
   - npm package build
3. **Verify**
   - Smoke test built artifacts
4. **Publish**
   - GitHub Release upload
   - PyPI publish
   - npm publish

`docker-publish.yml` handles Docker image build and push for the same tag.

## Rollback Policy

- If release is partially published, cut a new PATCH tag for corrections
- Yank bad PyPI versions, deprecate bad npm versions, and retag Docker `latest`
- Record incident in changelog/release notes
