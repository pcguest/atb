# Disaster Recovery (Quarterly Test)

Objective: prove ATB can be restored from source and secrets in a fresh environment.

## Recovery steps

1. Clone and bootstrap

```bash
git clone https://github.com/pcguest/atb.git
cd atb
go test ./...
```

2. Reconfigure repository secrets

- `NPM_TOKEN`
- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`
- `DISCORD_WEBHOOK_URL`

PyPI publishing uses GitHub OIDC trusted publishing. Verify the trusted publisher
configuration in PyPI instead of restoring a repository secret.

3. Validate CI/CD

- Trigger CI workflow on latest `main` commit.
- Run registry health workflow manually.
- Confirm release workflow can build artifacts on a dry-run tag.

4. Validate package pipelines

- Confirm Python SDK build/test in `sdk/python`.
- Confirm TypeScript SDK build/test in `sdk/typescript`.

5. Record outcome

- Save date, operator, pass/fail, and blockers in the release log.

## RTO/RPO Targets (Pragmatic)

- Target restore time (RTO): < 2 hours
- Recovery point objective (RPO): latest pushed commit/tag
