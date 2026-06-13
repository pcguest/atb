# ATB Configuration Reference

ATB core CLI is local-first and does not require environment variables for day-to-day use.

## Runtime defaults

- Default bundle directory: `./run.atb/`
- Default bundle file: `./run.atb/bundle.atb`

## CLI flags

| Command | Flags |
| --- | --- |
| `atb init` | `--dry-run`, `--format text or json` |
| `atb append` | `<type> <json>` or `--data <json>`, `--actor-id`, `--org-id`, `--workspace-id`, `--sign-policy`, `--dry-run`, `--format text or json` |
| `atb snapshot` | `<name>`, `--bundle <path>`, `--quiet`, `--dry-run`, `--format text or json` |
| `atb verify` | optional `bundle_path`, `--bundle <path>`, `--remote s3://bucket/key`, `--profile <id or path>`, `--json`, `--format text or json`, `--dry-run`, `--quiet`, `--trace`, `--with-anchor`, `--with-snapshot-check`, `--roots <pem-file>` |
| `atb inspect` | optional `bundle_path`, `--bundle <path>`, `--json`, `--seq <n>` |
| `atb events` | `--json`, `--profile <id>` |
| `atb trust-report` | optional `bundle_path`, `--format markdown, json, or text`, `--profile <id>` |
| `atb view` | optional `bundle_path`, `--bundle <path>`, `--host <host>`, `--port <port>`, `--no-open`, `--log-reveals` |
| `atb capture run` | `--bundle <path>`, `--snapshot <name>`, `--env-prefix <NAME>`, `--profile <id>`, `--lock-wait <duration>`, `-- <command> [args...]` |
| `atb import chatlog` | `--from <provider-type>`, `--input <path or ->`, `--bundle <path>`, `--snapshot <name>`, `--format text or json`, `--max-input-size <bytes>` |

## Capture environment variables

`atb capture run` injects environment variables into the child process so the
child can locate the bundle and tag events with the active capture run. The
canonical `ATB_`-prefixed variables are always set:

| Variable | Value |
| --- | --- |
| `ATB_BUNDLE_PATH` | Absolute path to the bundle the wrapper prepared. |
| `ATB_CAPTURE_RUN_ID` | Run identifier of the form `cap-<16-hex>`, generated per invocation. |
| `ATB_CAPTURE_MODE` | Always `run` for `atb capture run`. |

When `--env-prefix <NAME>` is supplied, the child additionally sees the same
three variables under the requested prefix. The prefix is upper-cased and
must match `^[A-Za-z][A-Za-z0-9_]*$`; an invalid prefix is rejected at flag
parse. `--env-prefix=ATB` duplicates the canonical prefix and emits a warning
on stderr.

| Variable (when `--env-prefix MYAPP` is set) | Value |
| --- | --- |
| `MYAPP_BUNDLE_PATH` | Same value as `ATB_BUNDLE_PATH`. |
| `MYAPP_CAPTURE_RUN_ID` | Same value as `ATB_CAPTURE_RUN_ID`. |
| `MYAPP_CAPTURE_MODE` | Always `run`. |

The `ATB_LOCK_WAIT` environment variable supplies a default for `--lock-wait`
when the flag is not passed; CLI flags always take precedence.

## Push defaults (`.atb/config.json`)

`atb push` reads per-project defaults from `.atb/config.json` under a `push` key. CLI flags always override config file values.

```json
{
  "version": 1,
  "push": {
    "target": "s3://my-bucket/prefix",
    "endpoint_url": "https://s3.eu-west-1.amazonaws.com",
    "region": "eu-west-1",
    "lock_mode": "COMPLIANCE",
    "lock_until": "2028-01-01",
    "credentials_source": "env"
  }
}
```

| Key | Description |
| --- | --- |
| `target` | Default S3 URI (e.g. `s3://bucket/prefix`). Overridden by the positional CLI argument. |
| `endpoint_url` | S3-compatible endpoint URL. Omit for standard AWS S3. |
| `region` | AWS region for SigV4 signing. |
| `lock_mode` | Object-lock mode: `COMPLIANCE` or `GOVERNANCE`. Required when `lock_until` is set. |
| `lock_until` | WORM retain-until date in `YYYY-MM-DD` format. |
| `credentials_source` | Credential resolver: `env` or `file` (maps to the standard AWS chain). |

## CI/CD secrets

These are configured in GitHub repository secrets for publish/notification workflows.

| Secret | Used by | Purpose |
| --- | --- | --- |
| `NPM_TOKEN` | `.github/workflows/release.yml` | Publish TypeScript SDK |
| `ATB_SIGNING_KEY_PEM` | `.github/workflows/release.yml` | Sign the release evidence bundle before publication |
| `DOCKERHUB_USERNAME` | `.github/workflows/docker-publish.yml` | Push Docker images |
| `DOCKERHUB_TOKEN` | `.github/workflows/docker-publish.yml` | Push Docker images |
| `DISCORD_WEBHOOK_URL` | `.github/workflows/ci.yml`, `.github/workflows/ops.yml` | Failure and ops notifications |
| `GITHUB_TOKEN` | GitHub-provided | Workflow auth for release publication |
| `ATB_WORM_S3_ACCESS_KEY_ID` | CI workflows using `atb push` | AWS access key ID for WORM S3 uploads |
| `ATB_WORM_S3_SECRET_ACCESS_KEY` | CI workflows using `atb push` | AWS secret access key paired with above |

PyPI publishing uses GitHub OIDC trusted publishing through `.github/workflows/release.yml`, so no repository secret is required for PyPI.

See `docs/release/secrets.md` for the canonical inventory, required permissions, and process for adding new secrets.

## Security notes

- Do not commit `.env` files, tokens, or private keys.
- Keep runtime payload secrets out of ATB event data unless explicitly required.
- Rotate publish and webhook tokens after any suspected exposure.
