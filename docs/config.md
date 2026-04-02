<!-- Archive: historical release doc for v1.1.0. Not maintained. -->
# ATB Configuration Reference

ATB core CLI is local-first and does not require environment variables for day-to-day use.

## Runtime Defaults

- Default bundle directory: `./run.atb/`
- Default bundle file: `./run.atb/bundle.atb`

## CLI Flags (Current)

| Command | Flags |
| --- | --- |
| `atb append` | `<json>` or `--data <json>` |
| `atb snapshot` | `--gate <pass|fail>` |
| `atb verify` | optional `bundle_path` |
| `atb view` | optional `bundle_path`, `--port <port>`, `--no-open`, `--ui-experimental` |

## CI/CD Secrets

These are configured in GitHub repository secrets for publish/notification workflows.

| Secret | Used by | Purpose |
| --- | --- | --- |
| `NPM_TOKEN` | `.github/workflows/release.yml` | Publish TypeScript SDK |
| `DOCKERHUB_USERNAME` | `.github/workflows/docker-publish.yml` | Push Docker images |
| `DOCKERHUB_TOKEN` | `.github/workflows/docker-publish.yml` | Push Docker images |
| `GITHUB_TOKEN` | GitHub-provided | Workflow auth for release publication |

PyPI publishing in v1.1.0 uses GitHub OIDC trusted publishing through `.github/workflows/release.yml`, so no repository secret is required for PyPI.

## Security Notes

- Do not commit `.env` files, tokens, or private keys.
- Keep runtime payload secrets out of ATB event data unless explicitly required.
- Rotate publish and webhook tokens after any suspected exposure.
