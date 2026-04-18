# API Documentation

## How to Update API Docs
1. Change handlers in `pkg/api/v1/`
2. Regenerate `openapi.yaml`:
   - Installed binary: `atb doc gen-openapi`
   - From source: `go run ./cmd/atb doc gen-openapi`
3. Update schema docs manually if needed
4. Run `make hygiene-full` to validate

`openapi.yaml` — OpenAPI 3.x specification for the `atb view` dashboard HTTP API (`GET /api/v1/bundle/profile`, `POST /api/v1/bundle/verify`, and related endpoints).
