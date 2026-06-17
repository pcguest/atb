// SPDX-License-Identifier: MIT
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// JWTValidator validates JWTs issued by an OIDC provider.
type JWTValidator struct {
	issuer   string
	audience string
	mu       sync.RWMutex
	jwks     jwk.Set
	jwksURL  string
	logger   *slog.Logger
}

// NewJWTValidator creates a new JWTValidator.
func NewJWTValidator(ctx context.Context, issuer, audience string) (*JWTValidator, error) {
	jwksURL := strings.TrimSuffix(issuer, "/") + "/.well-known/jwks.json"
	jwks, err := jwk.Fetch(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", jwksURL, err)
	}

	validator := &JWTValidator{
		issuer:   issuer,
		audience: audience,
		jwks:     jwks,
		jwksURL:  jwksURL,
		logger:   slog.Default(),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			newJwks, err := jwk.Fetch(context.Background(), jwksURL)
			if err != nil {
				validator.logger.Error("failed to refresh JWKS", "error", err)
				continue
			}
			validator.mu.Lock()
			validator.jwks = newJwks
			validator.mu.Unlock()
			validator.logger.Debug("JWKS refreshed successfully")
		}
	}()

	return validator, nil
}

// Validate validates a JWT token string and returns the claims.
func (v *JWTValidator) Validate(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found in token")
		}
		v.mu.RLock()
		defer v.mu.RUnlock()
		key, found := v.jwks.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("JWK with kid %s not found", kid)
		}
		var pubkey interface{}
		if err := key.Raw(&pubkey); err != nil {
			return nil, fmt.Errorf("failed to get public key from JWK: %w", err)
		}
		return pubkey, nil
	}, jwt.WithAudience(v.audience), jwt.WithIssuer(v.issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil {
		return nil, fmt.Errorf("JWT validation failed: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("JWT validation failed: token is invalid")
	}
	return claims, nil
}

// ExtractRole extracts the role from the JWT claims.
func (v *JWTValidator) ExtractRole(claims jwt.MapClaims) (Role, error) {
	if roleClaim, ok := claims["role"]; ok {
		if roleStr, ok := roleClaim.(string); ok {
			role := Role(roleStr)
			if role.IsValid() {
				return role, nil
			}
		}
	}
	if rolesClaim, ok := claims["roles"]; ok {
		if roles, ok := rolesClaim.([]string); ok {
			for _, roleStr := range roles {
				role := Role(roleStr)
				if role.IsValid() {
					return role, nil
				}
			}
		}
		if rolesArr, ok := rolesClaim.([]interface{}); ok {
			for _, r := range rolesArr {
				if roleStr, ok := r.(string); ok {
					role := Role(roleStr)
					if role.IsValid() {
						return role, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("no valid role claim found in JWT")
}
