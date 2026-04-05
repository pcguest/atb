# Integration Tests

These integration tests verify cross-SDK bundle compatibility against the Go verifier. They cover SDK-generated `.atb` bundles written by language runtimes outside Go and confirm that the Go loader can read them and that `bundle.Verify()` succeeds.

## Run

```bash
make test-integration
```

## Prerequisites

- Python 3.9 or newer
- Python SDK dependencies installed for `sdk/python`
