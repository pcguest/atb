# Contributing

Thanks for your interest in improving ATB.

## Local Setup

```bash
git clone https://github.com/pcguest/atb.git
cd atb
```

### Go CLI

```bash
go test ./...
go build -o atb ./cmd/atb
```

### Python SDK

```bash
cd sdk/python
python3 -m venv venv
source venv/bin/activate
pip install -e .[dev]
pytest -v
```

### TypeScript SDK

```bash
cd sdk/typescript
npm ci
npm run typecheck
npm run build
```

## Commit Style

Use conventional commits:

- `feat(scope): description`
- `fix(scope): description`
- `docs(scope): description`
- `chore(scope): description`
- `test(scope): description`

## Python SDK Release to PyPI

The `Publish Python SDK` workflow runs on git tags matching `v*`.

1. Configure one publishing path:

- Token-based: set repository secret `PYPI_API_TOKEN`.
- Trusted publishing: create a matching publisher in PyPI for
  `pcguest/atb/.github/workflows/pypi.yml` with tag refs.
2. Validate package locally:

```bash
cd sdk/python
python -m pip install --upgrade pip
python -m pip install build twine
python -m build
twine check dist/*
```

3. Create and push an annotated tag (match package version):

```bash
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
```

4. Confirm publish status in GitHub Actions and verify:

```bash
pip install atb-sdk
```

## TypeScript SDK Release to npm

The `Publish TypeScript SDK to npm` workflow also runs on git tags matching `v*`.

1. Set repository secret `NPM_TOKEN` (publish token).
2. Validate package locally:

```bash
cd sdk/typescript
npm ci
npm run typecheck
npm run build
npm pack
```

3. Push a tag that matches `sdk/typescript/package.json` version:

```bash
git tag -a v0.1.1 -m "Release v0.1.1"
git push origin v0.1.1
```

4. Confirm publish status and verify:

```bash
npm install @pcguest/atb-sdk
```

## Link Verification

To verify external links:

```bash
grep -roE 'https?://[^\s\)"`]+' README.md docs/ | grep -v "github.com/pcguest/atb" | sort -u | xargs -n1 curl -sI | grep -E "HTTP|404|500"
```

Portable fallback:

```bash
rg --no-filename -o 'https?://[^\s)"`]+' README.md docs/ | grep -v "github.com/pcguest/atb" | sort -u | while read -r url; do curl -sI "$url" | head -1; done
```

Note: `www.npmjs.com` may return `403` to automated `HEAD` requests; confirm package availability with:

```bash
curl -sI https://registry.npmjs.org/@pcguest/atb-sdk | head -1
```
