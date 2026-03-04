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
| `atb view` | optional `bundle_path`, `--port <port>` |

## CI/CD Secrets

These are configured in GitHub repository secrets for publish/notification workflows.

| Secret | Used by | Purpose |
| --- | --- | --- |
| `PYPI_API_TOKEN` | `.github/workflows/pypi.yml` | Publish Python SDK |
| `NPM_TOKEN` | `.github/workflows/npm.yml` | Publish TypeScript SDK |
| `DISCORD_WEBHOOK_URL` | CI/release/health workflows | Failure/release notifications |
| `GITHUB_TOKEN` | GitHub-provided | Workflow auth (labeling/docs/release) |

## Security Notes

- Do not commit `.env` files, tokens, or private keys.
- Keep runtime payload secrets out of ATB event data unless explicitly required.
- Rotate publish and webhook tokens after any suspected exposure.
