package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func TestNewJWTValidator(t *testing.T) {
	t.Parallel()

	// Create a test JWKS server
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey

	jwkKey, err := jwk.FromRaw(&publicKey)
	if err != nil {
		t.Fatalf("failed to create JWK from public key: %v", err)
	}
	_ = jwkKey.Set("kid", "test-kid")

	jwks := jwk.NewSet()
	jwks.AddKey(jwkKey)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(jwks)
			return
		}
		http.NotFound(w, r)
	}))
	defer jwksServer.Close()

	issuer := jwksServer.URL
	audience := "test-audience"

	validator, err := NewJWTValidator(context.Background(), issuer, audience)
	if err != nil {
		t.Fatalf("NewJWTValidator failed: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":  issuer,
		"aud":  audience,
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Unix(),
		"role": string(RoleViewer),
	})
	token.Header["kid"] = "test-kid"
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	if _, err := validator.Validate(context.Background(), signedToken); err != nil {
		t.Fatalf("expected validator to load JWKS and validate token: %v", err)
	}
}

func TestJWTValidatorValidateAndExtractRole(t *testing.T) {
	t.Parallel()

	// Create a test JWKS server
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey

	jwkKey, err := jwk.FromRaw(&publicKey)
	if err != nil {
		t.Fatalf("failed to create JWK from public key: %v", err)
	}
	_ = jwkKey.Set("kid", "test-kid")

	jwks := jwk.NewSet()
	jwks.AddKey(jwkKey)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(jwks)
			return
		}
		http.NotFound(w, r)
	}))
	defer jwksServer.Close()

	issuer := jwksServer.URL
	audience := "test-audience"

	validator, err := NewJWTValidator(context.Background(), issuer, audience)
	if err != nil {
		t.Fatalf("NewJWTValidator failed: %v", err)
	}

	// Test cases for JWT validation and role extraction
	tests := []struct {
		name           string
		claims         jwt.MapClaims
		wantRole       Role
		wantErr        bool
		wantExtractErr bool
	}{
		{
			name: "valid JWT with role claim",
			claims: jwt.MapClaims{
				"iss":  issuer,
				"aud":  audience,
				"exp":  time.Now().Add(time.Hour).Unix(),
				"iat":  time.Now().Unix(),
				"role": string(RoleOperator),
			},
			wantRole:       RoleOperator,
			wantErr:        false,
			wantExtractErr: false,
		},
		{
			name: "valid JWT with roles claim (array)",
			claims: jwt.MapClaims{
				"iss":   issuer,
				"aud":   audience,
				"exp":   time.Now().Add(time.Hour).Unix(),
				"iat":   time.Now().Unix(),
				"roles": []string{string(RoleViewer), string(RoleAuditor)},
			},
			wantRole:       RoleViewer, // Should pick the first valid role
			wantErr:        false,
			wantExtractErr: false,
		},
		{
			name: "valid JWT with no role claim",
			claims: jwt.MapClaims{
				"iss": issuer,
				"aud": audience,
				"exp": time.Now().Add(time.Hour).Unix(),
				"iat": time.Now().Unix(),
			},
			wantRole:       "",
			wantErr:        false,
			wantExtractErr: true,
		},
		{
			name: "expired JWT",
			claims: jwt.MapClaims{
				"iss":  issuer,
				"aud":  audience,
				"exp":  time.Now().Add(-time.Hour).Unix(),
				"iat":  time.Now().Unix(),
				"role": string(RoleViewer),
			},
			wantRole:       "",
			wantErr:        true,
			wantExtractErr: false, // Extraction might succeed even if validation fails
		},
		{
			name: "invalid audience",
			claims: jwt.MapClaims{
				"iss":  issuer,
				"aud":  "wrong-audience",
				"exp":  time.Now().Add(time.Hour).Unix(),
				"iat":  time.Now().Unix(),
				"role": string(RoleViewer),
			},
			wantRole:       "",
			wantErr:        true,
			wantExtractErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, tt.claims)
			token.Header["kid"] = "test-kid"
			signedToken, err := token.SignedString(privateKey)
			if err != nil {
				t.Fatalf("failed to sign token: %v", err)
			}

			claims, err := validator.Validate(context.Background(), signedToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			role, err := validator.ExtractRole(claims)
			if (err != nil) != tt.wantExtractErr {
				t.Errorf("ExtractRole() error = %v, wantExtractErr %v", err, tt.wantExtractErr)
				return
			}
			if !tt.wantExtractErr && role != tt.wantRole {
				t.Errorf("ExtractRole() got role = %q, want %q", role, tt.wantRole)
			}
		})
	}
}
