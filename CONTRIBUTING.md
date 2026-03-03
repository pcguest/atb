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

1. Ensure repository secret `PYPI_API_TOKEN` is configured.
2. Validate package locally:

```bash
cd sdk/python
python -m pip install --upgrade pip
python -m pip install build twine
python -m build
twine check dist/*
```

3. Create and push a tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

4. Confirm publish status in GitHub Actions and verify:

```bash
pip install atb-sdk
```
