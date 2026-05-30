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
)

// unauthorisedResponseBody is the exact JSON body returned for failed
// authentication. Kept as a package-level constant so callers and tests can
// pin the byte-for-byte response contract.
const unauthorisedResponseBody = `{"error":"unauthorised","code":401}`

// Middleware returns an http.Handler that enforces shared-secret bearer-token
// authentication around next.
//
// Behaviour:
//   - When token is empty, every request is forwarded to next unchanged.
//   - When token is non-empty, every request except GET /health must carry
//     Authorization: Bearer <token>. Mismatched or absent tokens receive a
//     401 application/json response with the body
//     {"error":"unauthorised","code":401}.
//   - Token comparison uses crypto/subtle.ConstantTimeCompare to avoid
//     timing-based discrimination between near-correct and wholly-wrong
//     tokens.
//
// The health bypass is intentionally limited to GET so non-idempotent methods
// against /health cannot be used to smuggle requests past the guard.
func Middleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !hasValidBearerToken(r, token) {
			writeUnauthorised(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

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
