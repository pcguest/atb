// SPDX-License-Identifier: MIT
package auth

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey int

const (
	// contextKeyRole stores the authenticated user's role in the request
	// context. Unexported so callers must go through WithRole/GetRoleFromContext.
	contextKeyRole contextKey = iota
)

// GetRoleFromContext retrieves the authenticated role from the request context.
func GetRoleFromContext(ctx context.Context) (Role, bool) {
	role, ok := ctx.Value(contextKeyRole).(Role)
	return role, ok
}

// WithRole stores the authenticated role on ctx using the package-private key.
func WithRole(ctx context.Context, role Role) context.Context {
	return context.WithValue(ctx, contextKeyRole, role)
}
