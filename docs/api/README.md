# API Documentation (v1.1.0)

## How to Update API Docs
1. Change handlers in `pkg/api/v1/`
2. Run: `go run ./cmd/atb doc gen-openapi` to regenerate `openapi.yaml`
3. Update schema docs manually if needed
4. Run `make hygiene-full` to validate
