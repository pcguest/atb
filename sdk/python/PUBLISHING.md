# Publishing `atb-sdk`

These commands prepare and publish the Python SDK to PyPI from `sdk/python/`.

## Prerequisites

```bash
python3 -m pip install --upgrade build twine
```

## Build and verify

```bash
python3 -m build
python3 -m twine check dist/*
```

## Publish

```bash
python3 -m twine upload dist/*
```

The package metadata points back to the main repository at <https://github.com/pcguest/atb>.
