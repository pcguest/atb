// SPDX-License-Identifier: MIT
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

// jwksFetchTimeout bounds each JWKS fetch so validator startup and the
// background refresh loop cannot hang on an unresponsive issuer. Variable so
// tests can shorten it.
var jwksFetchTimeout = 10 * time.Second

// resolveJWKSURL resolves the issuer's JWKS endpoint from its OIDC discovery
// document (/.well-known/openid-configuration -> jwks_uri). Providers that do
// not serve discovery metadata fall back to the conventional
// /.well-known/jwks.json location.
func resolveJWKSURL(ctx context.Context, issuer string) string {
	base := strings.TrimSuffix(issuer, "/")
	fallback := base + "/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/.well-known/openid-configuration", nil)
	if err != nil {
		return fallback
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback
	}
	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil || doc.JWKSURI == "" {
		return fallback
	}
	if u, err := url.Parse(doc.JWKSURI); err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return fallback
	}
	return doc.JWKSURI
}

// NewJWTValidator creates a new JWTValidator.
func NewJWTValidator(ctx context.Context, issuer, audience string) (*JWTValidator, error) {
	// Bound the startup discovery and fetch; ctx itself stays alive so it can
	// scope the background refresh loop below.
	fetchCtx, cancel := context.WithTimeout(ctx, jwksFetchTimeout)
	defer cancel()
	jwksURL := resolveJWKSURL(fetchCtx, issuer)
	jwks, err := jwk.Fetch(fetchCtx, jwksURL)
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

	validator.startRefresh(ctx, 5*time.Minute, func(refreshCtx context.Context) (jwk.Set, error) {
		bounded, cancel := context.WithTimeout(refreshCtx, jwksFetchTimeout)
		defer cancel()
		return jwk.Fetch(bounded, jwksURL)
	})

	return validator, nil
}

func (v *JWTValidator) startRefresh(ctx context.Context, interval time.Duration, fetch func(context.Context) (jwk.Set, error)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				newJwks, err := fetch(ctx)
				if err != nil {
					v.logger.Error("failed to refresh JWKS", "error", err)
					continue
				}
				v.mu.Lock()
				v.jwks = newJwks
				v.mu.Unlock()
				v.logger.Debug("JWKS refreshed successfully")
			}
		}
	}()
}

// refetchKeyID fetches the JWKS once more and looks up kid, swapping in the
// fresh set on success. Used when a token presents a kid missing from the
// cached set (mid-rotation).
func (v *JWTValidator) refetchKeyID(ctx context.Context, kid string) (jwk.Key, bool) {
	fresh, err := jwk.Fetch(ctx, v.jwksURL)
	if err != nil {
		v.logger.Error("failed to refetch JWKS on kid miss", "error", err)
		return nil, false
	}
	v.mu.Lock()
	v.jwks = fresh
	v.mu.Unlock()
	return fresh.LookupKeyID(kid)
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
		key, found := v.jwks.LookupKeyID(kid)
		v.mu.RUnlock()
		if !found {
			// The signing key may have rotated since the last refresh; refetch
			// once before rejecting so rotation windows don't 401 valid tokens.
			key, found = v.refetchKeyID(ctx, kid)
		}
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
