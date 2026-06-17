// SPDX-License-Identifier: MIT
package auth

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey int

const (
	// ContextKeyRole is the key used to store the authenticated user's role in the request context.
	ContextKeyRole contextKey = iota
)

// GetRoleFromContext retrieves the authenticated role from the request context.
func GetRoleFromContext(ctx context.Context) (Role, bool) {
	role, ok := ctx.Value(ContextKeyRole).(Role)
	return role, ok
}

// WithRole stores the authenticated role on ctx using the package-private key.
func WithRole(ctx context.Context, role Role) context.Context {
	return context.WithValue(ctx, ContextKeyRole, role)
}
