# Lint suppressions

This file lists every active `//lint:ignore` (staticcheck) and `//nolint`
(go vet / golangci-lint) directive in the repository, with the reason each
suppression is in place. Add a row here whenever a new suppression is
introduced; remove the row when the underlying directive is removed.

The general policy: prefer fixing findings over suppressing them. Suppressions
are reserved for findings whose only "fix" would be a behaviour-changing
migration that this codebase is not yet ready to absorb. There are currently
no active lint suppressions.
