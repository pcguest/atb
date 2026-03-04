# Week 4 Publish Runbook

This runbook unblocks package publishing for both SDKs.

## 1) Required GitHub Secrets

Set these repository secrets in `pcguest/atb`:

- `PYPI_API_TOKEN`
- `NPM_TOKEN`

Quick check:

```bash
gh api repos/pcguest/atb/actions/secrets --jq '.secrets[].name'
```

## 2) Version Alignment (Current Target: 0.1.1)

These must match:

- `sdk/python/pyproject.toml` -> `version = "0.1.1"`
- `sdk/typescript/package.json` -> `"version": "0.1.1"`
- git tag -> `v0.1.1`

The CI publish workflows fail fast if tag/version mismatch.

## 3) Local Preflight

```bash
# Python package
cd sdk/python
python3 -m build
twine check dist/*

# TypeScript package
cd ../typescript
npm ci
npm run typecheck
npm run build
npm pack
rm -f atb-dev-sdk-0.1.1.tgz
```

## 4) Trigger Publish

```bash
cd ~/atb
git checkout main
git pull origin main

git tag -a v0.1.1 -m "Release v0.1.1: publish Python and TypeScript SDKs"
git push origin v0.1.1
```

Workflows triggered by tag:

- `.github/workflows/pypi.yml`
- `.github/workflows/npm.yml`

## 5) Verify Installs

```bash
# Python
python3 -m venv /tmp/atb-py-test
source /tmp/atb-py-test/bin/activate
pip install atb-sdk
python -c "from atb import Bundle; print(Bundle)"
deactivate
rm -rf /tmp/atb-py-test

# TypeScript
mkdir -p /tmp/atb-ts-test && cd /tmp/atb-ts-test
npm init -y
npm install @pcguest/atb-sdk
node -e "const { Bundle } = require('@pcguest/atb-sdk'); console.log(typeof Bundle)"
rm -rf /tmp/atb-ts-test
```

## 6) Dogfood the Release

```bash
cd ~/atb
./atb append release --data '{
  "component": "sdk",
  "version": "0.1.1",
  "action": "published to pypi and npm",
  "channels": ["https://pypi.org/project/atb-sdk/", "https://www.npmjs.com/package/@pcguest/atb-sdk"]
}'
./atb snapshot build --gate pass
./atb verify
```
