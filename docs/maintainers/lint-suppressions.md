# Lint suppressions

This file lists every active `//lint:ignore` (staticcheck) and `//nolint`
(go vet / golangci-lint) directive in the repository, with the reason each
suppression is in place. Add a row here whenever a new suppression is
introduced; remove the row when the underlying directive is removed.

The general policy: prefer fixing findings over suppressing them. Suppressions
are reserved for findings whose only "fix" would be a behaviour-changing
migration that this codebase is not yet ready to absorb.

| Path | Rule | Reason | Scope and mitigation |
| --- | --- | --- | --- |
| `internal/compliancepack/coverage_test.go` | `SA1012` | `Build` deliberately accepts a nil context for backward-compatible callers. | One compatibility test; production callers use non-nil contexts and the test verifies fallback behaviour. |
| `internal/mcp/coverage_test.go` | `SA1012` | `Serve` deliberately accepts a nil context and substitutes a background context. | One compatibility test; production callers use non-nil contexts and the test verifies fallback behaviour. |
