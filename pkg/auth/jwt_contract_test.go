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
	"sync/atomic"
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

func TestNewJWTValidatorBoundsStartupJWKSFetch(t *testing.T) {
	previous := jwksFetchTimeout
	jwksFetchTimeout = 200 * time.Millisecond
	t.Cleanup(func() { jwksFetchTimeout = previous })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		// Stall the JWKS endpoint well past the shortened fetch timeout, but
		// finish eventually so httptest's Close is not left waiting on us.
		time.Sleep(1500 * time.Millisecond)
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	start := time.Now()
	_, err := NewJWTValidator(context.Background(), server.URL, "test-audience")
	if err == nil {
		t.Fatal("NewJWTValidator succeeded against a hanging JWKS endpoint")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("NewJWTValidator hung for %s despite fetch timeout", elapsed)
	}
}

func TestNewJWTValidatorRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name     string
		issuer   string
		audience string
	}{
		{name: "relative issuer", issuer: "issuer.example", audience: "audience"},
		{name: "unsupported scheme", issuer: "file:///tmp/issuer", audience: "audience"},
		{name: "embedded credentials", issuer: "https://user:secret@example.com", audience: "audience"},
		{name: "fragment", issuer: "https://example.com/#fragment", audience: "audience"},
		{name: "query", issuer: "https://example.com/?tenant=one", audience: "audience"},
		{name: "empty audience", issuer: "https://example.com", audience: "  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewJWTValidator(context.Background(), tc.issuer, tc.audience); err == nil {
				t.Fatalf("NewJWTValidator(%q, %q) unexpectedly succeeded", tc.issuer, tc.audience)
			}
		})
	}
}

func TestJWTValidatorBoundsJWKSResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(strings.Repeat("x", maxOIDCResponseBytes+1)))
	}))
	t.Cleanup(server.Close)

	_, err := NewJWTValidator(context.Background(), server.URL, "test-audience")
	if err == nil || !strings.Contains(err.Error(), "exceeds 1 MiB") {
		t.Fatalf("NewJWTValidator error=%v, want bounded-response error", err)
	}
}

func TestOIDCFetchRefusesRedirects(t *testing.T) {
	var targetRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetRequests.Add(1)
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(target.Close)
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirect.Close)

	if _, err := fetchJWKS(context.Background(), redirect.URL); err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("fetchJWKS error=%v, want redirect refusal", err)
	}
	if got := targetRequests.Load(); got != 0 {
		t.Fatalf("redirect target received %d requests; want 0", got)
	}
}

func TestJWTValidatorRejectsInvalidTokens(t *testing.T) {
	fixture := newJWTFixture(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fixture.validator.Validate(canceled, "unused"); err == nil {
		t.Fatal("canceled validation context succeeded")
	}
	if _, err := fixture.validator.Validate(context.Background(), strings.Repeat("x", maxJWTBytes+1)); err == nil || !strings.Contains(err.Error(), "exceeds 64 KiB") {
		t.Fatalf("oversized JWT error=%v, want bounded-token error", err)
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

func TestJWTValidatorRateLimitsUnknownKeyRefetches(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	key, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("create JWK: %v", err)
	}
	if err := key.Set(jwk.KeyIDKey, "known-kid"); err != nil {
		t.Fatalf("set JWK key ID: %v", err)
	}
	set := jwk.NewSet()
	if err := set.AddKey(key); err != nil {
		t.Fatalf("add JWK: %v", err)
	}

	var jwksRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		jwksRequests.Add(1)
		_ = json.NewEncoder(w).Encode(set)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	validator, err := NewJWTValidator(ctx, server.URL, "test-audience")
	if err != nil {
		t.Fatalf("NewJWTValidator: %v", err)
	}
	for _, kid := range []string{"unknown-one", "unknown-two"} {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss": server.URL, "aud": "test-audience",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Add(-time.Minute).Unix(),
		})
		token.Header["kid"] = kid
		signed, signErr := token.SignedString(privateKey)
		if signErr != nil {
			t.Fatalf("sign JWT: %v", signErr)
		}
		if _, validateErr := validator.Validate(context.Background(), signed); validateErr == nil {
			t.Fatalf("Validate unexpectedly accepted %q", kid)
		}
	}
	// One startup fetch plus only one miss-triggered refetch: the second miss
	// falls inside the cooldown and must use the cached set.
	if got := jwksRequests.Load(); got != 2 {
		t.Fatalf("JWKS requests=%d, want 2", got)
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
