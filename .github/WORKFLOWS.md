# ATB GitHub Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| ci.yml | push/PR to main | Test matrix + golden parity + docs smoke |
| security.yml | push/PR to main | gosec, bandit, npm audit security gates |
| release.yml | tag push | Build CLI + publish to PyPI/npm + GitHub Release |
| ops.yml | schedule/push/PR/issues | Docs deploy, registry health, feedback digest, labeling (conditional) |

## Adding a New Job

1. Pick the workflow that matches the job's purpose.
2. Add conditional `if:` to avoid unnecessary runs.
3. Test locally with `act` or trigger via `gh workflow run`.
4. Update this file.
