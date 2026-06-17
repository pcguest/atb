// SPDX-License-Identifier: MIT

// Package auth provides shared-secret bearer-token authentication middleware
// for the Custos daemon. ATB records and verifies evidence; it does not
// certify legal or regulatory compliance, and this middleware is a transport
// guard rather than a full identity system.
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"

	atbauth "github.com/pcguest/atb/pkg/auth"
)

const (
	// unauthorisedResponseBody is the exact JSON body returned for failed
	// authentication. Kept as a package-level constant so callers and tests can
	// pin the byte-for-byte response contract.
	unauthorisedResponseBody = `{"error":"unauthorised","code":401}`
)

// Middleware returns an http.Handler that enforces shared-secret bearer-token
// authentication and/or OIDC/JWT validation with RBAC around next.
//
// Behaviour:
//   - When token is empty and jwtValidator is nil, every request is forwarded to next with defaultRole.
//   - When token is non-empty, every request except GET /health and GET /custody/key must carry
//     Authorization: Bearer <token>. Mismatched or absent tokens receive a 401.
//   - When jwtValidator is non-nil, it attempts to validate a JWT from the Authorization header.
//     If valid, the role is extracted from claims or defaults to defaultRole.
//   - If both token and jwtValidator are provided, the shared secret takes precedence.
//   - Token comparison uses crypto/subtle.ConstantTimeCompare to avoid timing-based
//     discrimination between near-correct and wholly-wrong tokens.
//   - The authenticated user's role is stored in the request context.
func Middleware(sharedSecretToken string, jwtValidator *JWTValidator, defaultRole Role, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass for /health and /custody/key GET requests
		if r.Method == http.MethodGet && (r.URL.Path == "/health" || r.URL.Path == "/custody/key") {
			next.ServeHTTP(w, r)
			return
		}

		var authenticatedRole Role = ""
		var authenticated bool

		// 1. Try shared secret token authentication
		if sharedSecretToken != "" {
			if hasValidBearerToken(r, sharedSecretToken) {
				authenticated = true
				authenticatedRole = RoleAdmin // Shared secret token implies admin access
			} else {
				writeUnauthorised(w)
				return
			}
		}

		// 2. If not authenticated by shared secret, try JWT validation
		if !authenticated && jwtValidator != nil {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := jwtValidator.Validate(r.Context(), tokenString)
				if err != nil {
					writeUnauthorised(w)
					return
				}
				role, err := jwtValidator.ExtractRole(claims)
				if err != nil || !role.IsValid() {
					authenticatedRole = defaultRole // Use default role if no valid role claim
				} else {
					authenticatedRole = role
				}
				authenticated = true
			} else {
				writeUnauthorised(w)
				return
			}
		}

		// 3. If no authentication mechanism is configured, or if authentication failed
		if !authenticated && sharedSecretToken == "" && jwtValidator == nil {
			// No authentication configured, allow all access (dev mode)
			ctx := atbauth.WithRole(r.Context(), defaultRole)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		} else if !authenticated {
			// Authentication configured but failed
			writeUnauthorised(w)
			return
		}

		// Store the authenticated role in context
		ctx := atbauth.WithRole(r.Context(), authenticatedRole)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRoleFromContext retrieves the authenticated role from the request context.
var GetRoleFromContext = atbauth.GetRoleFromContext

// hasValidBearerToken returns true when the request carries an exact
// Authorization: Bearer <token> header.
func hasValidBearerToken(r *http.Request, token string) bool {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	supplied := header[len(prefix):]
	// Constant-time comparison prevents the response timing from depending on
	// how many leading bytes of the supplied token are correct.
	return subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) == 1
}

// writeUnauthorised emits the fixed 401 JSON body used by the middleware.
func writeUnauthorised(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(unauthorisedResponseBody))
}
