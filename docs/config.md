# ATB Configuration Reference

ATB core CLI is local-first and does not require environment variables for day-to-day use.

## Runtime Defaults

- Default bundle directory: `./run.atb/`
- Default bundle file: `./run.atb/bundle.atb`

## CLI Flags (Current)

| Command | Flags |
| --- | --- |
| `atb init` | `--dry-run`, `--format text or json` |
| `atb append` | `<type> <json>` or `--data <json>`, `--actor-id`, `--org-id`, `--workspace-id`, `--sign-policy`, `--dry-run`, `--format text or json` |
| `atb snapshot` | `<name>`, `--bundle <path>`, `--quiet`, `--dry-run`, `--format text or json` |
| `atb verify` | optional `bundle_path`, `--bundle <path>`, `--profile <id or path>`, `--json`, `--format text or json`, `--quiet`, `--trace`, `--with-anchor`, `--roots <pem-file>` |
| `atb events` | `--json`, `--profile <id>` |
| `atb trust-report` | optional `bundle_path`, `--format markdown, json, or text`, `--profile <id>` |
| `atb view` | optional `bundle_path`, `--bundle <path>`, `--port <port>`, `--no-open`, `--log-reveals`, `--ui-experimental` |

## CI/CD Secrets

These are configured in GitHub repository secrets for publish/notification workflows.

| Secret | Used by | Purpose |
| --- | --- | --- |
| `NPM_TOKEN` | `.github/workflows/release.yml` | Publish TypeScript SDK |
| `DOCKERHUB_USERNAME` | `.github/workflows/docker-publish.yml` | Push Docker images |
| `DOCKERHUB_TOKEN` | `.github/workflows/docker-publish.yml` | Push Docker images |
| `DISCORD_WEBHOOK_URL` | `.github/workflows/ci.yml`, `.github/workflows/ops.yml` | Failure and ops notifications |
| `GITHUB_TOKEN` | GitHub-provided | Workflow auth for release publication |

PyPI publishing uses GitHub OIDC trusted publishing through `.github/workflows/release.yml`, so no repository secret is required for PyPI.

## Security Notes

- Do not commit `.env` files, tokens, or private keys.
- Keep runtime payload secrets out of ATB event data unless explicitly required.
- Rotate publish and webhook tokens after any suspected exposure.
