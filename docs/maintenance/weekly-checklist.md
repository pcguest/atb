# ATB Weekly Maintenance Checklist

## Monday (Automated)

- [x] Registry health check runs via GitHub Actions
  - Script: `./scripts/check-registries.sh`
  - Workflow: `.github/workflows/registry-health.yml`
  - Alert: Discord notification on failure

## Monday (Manual, 5 Minutes)

- [ ] Review registry health check results in GitHub Actions
- [ ] Check PyPI and npm download trends
  - PyPI: <https://pypistats.org/packages/atb-sdk>
  - npm: <https://npmtrends.com/@pcguest/atb-sdk>
- [ ] Scan GitHub Issues for install/compatibility reports

## After Any SDK Change

- [ ] Run golden parity test locally:

```bash
cd test/golden
GOCACHE=/tmp/atb-go-cache go test -v -run TestGoldenCanonicalization
python3 verify.py
node verify.js
```

- [ ] Confirm canonical JSON and hashes match across all three implementations
- [ ] If `test/golden/input.json` changed, update `expectedCanonical` and `expectedHash` in `test/golden/golden_test.go`

## Before Release

- [ ] Bump versions in sync
  - Python: `sdk/python/pyproject.toml` and `sdk/python/atb/__init__.py`
  - TypeScript: `sdk/typescript/package.json`
- [ ] Tag with matching semver (`vX.Y.Z`), allowing temporary patch drift between registries only during publish timing
- [ ] Ensure CI + `golden-test` pass before publishing
- [ ] Verify release notes/changelog section in GitHub release body

## Monthly

- [ ] Review cloud costs (R2, Vercel, Supabase)
- [ ] Review dependency/security advisories (Go, Python, TypeScript, GitHub Actions)
- [ ] Update `docs/quickstart.md` if CLI/SDK flows changed
- [ ] Publish a monthly progress update (blog + social)
