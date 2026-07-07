// SPDX-License-Identifier: MIT
package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type jwtFixture struct {
	validator *JWTValidator
	private   *rsa.PrivateKey
	issuer    string
	audience  string
	cancel    context.CancelFunc
}

func newJWTFixture(t *testing.T) jwtFixture {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	key, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("create JWK: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, "test-kid"); err != nil {
		t.Fatalf("set JWK key ID: %v", err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(key); err != nil {
		t.Fatalf("add JWK: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(set); err != nil {
			t.Errorf("encode JWKS: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	validator, err := NewJWTValidator(ctx, server.URL+"/", "test-audience")
	if err != nil {
		cancel()
		t.Fatalf("NewJWTValidator: %v", err)
	}
	t.Cleanup(cancel)
	return jwtFixture{
		validator: validator,
		private:   privateKey,
		issuer:    server.URL + "/",
		audience:  "test-audience",
		cancel:    cancel,
	}
}

func (f jwtFixture) sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, kid any) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	if kid != nil {
		token.Header["kid"] = kid
	}
	signed, err := token.SignedString(f.private)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	return signed
}

func (f jwtFixture) validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":  f.issuer,
		"aud":  f.audience,
		"exp":  time.Now().Add(time.Hour).Unix(),
		"iat":  time.Now().Add(-time.Minute).Unix(),
		"role": string(RoleOperator),
	}
}

func TestJWTValidatorValidatesRS256AndExtractsRoles(t *testing.T) {
	fixture := newJWTFixture(t)
	claims, err := fixture.validator.Validate(
		context.Background(),
		fixture.sign(t, jwt.SigningMethodRS256, fixture.validClaims(), "test-kid"),
	)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	role, err := fixture.validator.ExtractRole(claims)
	if err != nil || role != RoleOperator {
		t.Fatalf("ExtractRole=%q, %v; want operator", role, err)
	}

	for _, tc := range []struct {
		name   string
		claims jwt.MapClaims
		want   Role
	}{
		{name: "typed roles", claims: jwt.MapClaims{"roles": []string{"unknown", string(RoleAuditor)}}, want: RoleAuditor},
		{name: "decoded roles", claims: jwt.MapClaims{"roles": []interface{}{42, "unknown", string(RoleAdmin)}}, want: RoleAdmin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fixture.validator.ExtractRole(tc.claims)
			if err != nil || got != tc.want {
				t.Fatalf("ExtractRole=%q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestJWTValidatorRejectsInvalidTokens(t *testing.T) {
	fixture := newJWTFixture(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fixture.validator.Validate(canceled, "unused"); err == nil {
		t.Fatal("canceled validation context succeeded")
	}

	tests := []struct {
		name   string
		method jwt.SigningMethod
		claims jwt.MapClaims
		kid    any
		want   string
	}{
		{name: "wrong algorithm", method: jwt.SigningMethodPS256, claims: fixture.validClaims(), kid: "test-kid", want: "unexpected signing method"},
		{name: "missing key id", method: jwt.SigningMethodRS256, claims: fixture.validClaims(), want: "kid header not found"},
		{name: "non-string key id", method: jwt.SigningMethodRS256, claims: fixture.validClaims(), kid: 42, want: "kid header not found"},
		{name: "unknown key id", method: jwt.SigningMethodRS256, claims: fixture.validClaims(), kid: "unknown", want: "not found"},
		{name: "expired", method: jwt.SigningMethodRS256, claims: jwt.MapClaims{
			"iss": fixture.issuer, "aud": fixture.audience,
			"exp": time.Now().Add(-time.Hour).Unix(), "iat": time.Now().Add(-2 * time.Hour).Unix(),
		}, kid: "test-kid", want: "token is expired"},
		{name: "wrong audience", method: jwt.SigningMethodRS256, claims: jwt.MapClaims{
			"iss": fixture.issuer, "aud": "wrong",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		}, kid: "test-kid", want: "invalid audience"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fixture.validator.Validate(context.Background(), fixture.sign(t, tc.method, tc.claims, tc.kid))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate error=%v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestJWTValidatorResolvesJWKSFromDiscovery(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	key, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("create JWK: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, "discovery-kid"); err != nil {
		t.Fatalf("set JWK key ID: %v", err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(key); err != nil {
		t.Fatalf("add JWK: %v", err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"jwks_uri": server.URL + "/custom/keys"})
		case "/custom/keys":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(set)
		default:
			// No /.well-known/jwks.json: resolution must come from discovery.
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator, err := NewJWTValidator(ctx, server.URL, "test-audience")
	if err != nil {
		t.Fatalf("NewJWTValidator with discovery metadata: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": server.URL, "aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
	})
	token.Header["kid"] = "discovery-kid"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	if _, err := validator.Validate(context.Background(), signed); err != nil {
		t.Fatalf("Validate with discovery-resolved JWKS: %v", err)
	}
}

func TestJWTValidatorRefetchesJWKSOnKidMiss(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	newKey := func(kid string) jwk.Key {
		k, err := jwk.FromRaw(&privateKey.PublicKey)
		if err != nil {
			t.Fatalf("create JWK: %v", err)
		}
		if err := k.Set(jwk.KeyIDKey, kid); err != nil {
			t.Fatalf("set JWK key ID: %v", err)
		}
		return k
	}

	var mu sync.Mutex
	served := jwk.NewSet()
	if err := served.AddKey(newKey("old-kid")); err != nil {
		t.Fatalf("add JWK: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(served)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator, err := NewJWTValidator(ctx, server.URL, "test-audience")
	if err != nil {
		t.Fatalf("NewJWTValidator: %v", err)
	}

	// Rotate: the provider now serves a key the validator has not cached.
	mu.Lock()
	rotated := jwk.NewSet()
	if err := rotated.AddKey(newKey("rotated-kid")); err != nil {
		mu.Unlock()
		t.Fatalf("add rotated JWK: %v", err)
	}
	served = rotated
	mu.Unlock()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": server.URL, "aud": "test-audience",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
	})
	token.Header["kid"] = "rotated-kid"
	signed, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign JWT: %v", err)
	}
	if _, err := validator.Validate(context.Background(), signed); err != nil {
		t.Fatalf("Validate after rotation should refetch JWKS: %v", err)
	}
}

func TestJWTValidatorConstructorAndRoleFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	if _, err := NewJWTValidator(context.Background(), server.URL, "audience"); err == nil {
		t.Fatal("NewJWTValidator succeeded against unavailable issuer")
	}

	validator := &JWTValidator{}
	for _, claims := range []jwt.MapClaims{
		{},
		{"role": "unknown"},
		{"role": 42},
		{"roles": []string{"unknown"}},
		{"roles": []interface{}{42, "unknown"}},
	} {
		if _, err := validator.ExtractRole(claims); err == nil {
			t.Fatalf("ExtractRole(%v) unexpectedly succeeded", claims)
		}
	}
}

func TestRolePermissionsAndContext(t *testing.T) {
	for _, role := range []Role{RoleViewer, RoleAuditor, RoleOperator, RoleAdmin} {
		if !role.IsValid() {
			t.Errorf("%q should be valid", role)
		}
	}
	if Role("unknown").IsValid() {
		t.Fatal("unknown role is valid")
	}
	if !RoleAdmin.HasPermission(RoleAdmin) ||
		!RoleOperator.HasPermission(RoleAuditor) ||
		!RoleAuditor.HasPermission(RoleViewer) ||
		RoleViewer.HasPermission(RoleAuditor) ||
		RoleAdmin.HasPermission(Role("unknown")) {
		t.Fatal("role permission ordering is incorrect")
	}

	ctx := WithRole(context.Background(), RoleAuditor)
	if got, ok := GetRoleFromContext(ctx); !ok || got != RoleAuditor {
		t.Fatalf("GetRoleFromContext=%q, %v", got, ok)
	}
	if _, ok := GetRoleFromContext(context.Background()); ok {
		t.Fatal("empty context returned a role")
	}
}
