package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pcguest/atb/pkg/jcs"
	"github.com/pcguest/custos/internal/auth"
	"github.com/pcguest/custos/internal/receipt"
)

func TestCustosIntegrationFilesystem(t *testing.T) {
	// 1. Setup custosd in a temporary directory
	tempDir := t.TempDir()
	wormDir := filepath.Join(tempDir, "worm")
	receiptDir := filepath.Join(tempDir, "receipts")
	custosPort := getFreePort(t)
	custosEndpoint := fmt.Sprintf("http://127.0.0.1:%d", custosPort)
	custosAuthToken := "test-token"
	t.Setenv("CUSTOS_AUTH_TOKEN", custosAuthToken)

	// Start custosd in a goroutine
	custosStdout := new(bytes.Buffer)
	custosStderr := new(bytes.Buffer)

	go func() {
		custosArgs := []string{
			"--worm-dir", wormDir,
			"--receipt-dir", receiptDir,
			"--host", "127.0.0.1",
			"--port", fmt.Sprintf("%d", custosPort),
			"--receipt-max-age-days", "1", // Enable retention
			"--receipt-max-count", "10",
			"--cleanup-interval", "1s", // Frequent cleanup for testing
		}
		exitCode := run(custosArgs, custosStdout, custosStderr)
		if exitCode != 0 {
			t.Errorf("custosd exited with non-zero code: %d\nStdout: %s\nStderr: %s", exitCode, custosStdout.String(), custosStderr.String())
		}
	}()

	// Wait for custosd to start
	waitForCustos(t, custosEndpoint)

	// 2. Ingest a valid ATB bundle through the daemon.
	firstBundle := createDummyBundle(t, "first")
	if err := ingestBundle(t, custosEndpoint, custosAuthToken, firstBundle); err != nil {
		t.Fatalf("ingest first bundle: %v", err)
	}

	// 3. Verify event was ingested by Custos
	receipts, err := listCustosReceipts(t, custosEndpoint, custosAuthToken)
	if err != nil {
		t.Fatalf("Failed to list receipts from Custos: %v", err)
	}
	if len(receipts) == 0 {
		t.Fatal("Expected at least one receipt in Custos, got none")
	}
	for _, r := range receipts {
		if r.Attestation == nil {
			t.Fatalf("receipt %s missing attestation", r.ReceiptID)
		}
	}

	// 4. Ingest a second bundle and verify the custody log grows.
	secondBundle := createDummyBundle(t, "second")
	if err := ingestBundle(t, custosEndpoint, custosAuthToken, secondBundle); err != nil {
		t.Fatalf("ingest second bundle: %v", err)
	}

	// 5. Verify bundle was ingested by Custos (check for a new receipt)
	newReceipts, err := listCustosReceipts(t, custosEndpoint, custosAuthToken)
	if err != nil {
		t.Fatalf("Failed to list receipts from Custos after bundle push: %v", err)
	}
	if len(newReceipts) <= len(receipts) {
		t.Fatal("Expected more receipts after bundle push, but count did not increase")
	}

	// 6. Test retention policy (wait for cleanup to run)
	time.Sleep(2 * time.Second) // Wait for cleanup interval (1s) to pass at least once

	// Send another bundle to trigger cleanup and ensure it's still working.
	thirdBundle := createDummyBundle(t, "third")
	if err := ingestBundle(t, custosEndpoint, custosAuthToken, thirdBundle); err != nil {
		t.Fatalf("ingest third bundle: %v", err)
	}

	finalReceipts, err := listCustosReceipts(t, custosEndpoint, custosAuthToken)
	if err != nil {
		t.Fatalf("Failed to list receipts from Custos after retention cleanup: %v", err)
	}
	// With MaxCount=10, we should have at most 10 receipts.
	if len(finalReceipts) > 10 {
		t.Fatalf("Retention policy failed, expected at most 10 receipts, got %d", len(finalReceipts))
	}
}

func TestCustosIntegrationOIDCJWTAndRBAC(t *testing.T) {
	tempDir := t.TempDir()
	wormDir := filepath.Join(tempDir, "worm")
	receiptDir := filepath.Join(tempDir, "receipts")
	custosPort := getFreePort(t)
	custosEndpoint := fmt.Sprintf("http://127.0.0.1:%d", custosPort)
	t.Setenv("CUSTOS_AUTH_TOKEN", "")

	issuer, signJWT, cleanup := startOIDCTestIssuer(t)
	defer cleanup()
	audience := "custos-integration"

	custosStdout := new(bytes.Buffer)
	custosStderr := new(bytes.Buffer)
	go func() {
		custosArgs := []string{
			"--worm-dir", wormDir,
			"--receipt-dir", receiptDir,
			"--host", "127.0.0.1",
			"--port", fmt.Sprintf("%d", custosPort),
			"--cleanup-interval", "0",
			"--oidc-issuer", issuer,
			"--oidc-audience", audience,
			"--default-role", string(auth.RoleViewer),
		}
		exitCode := run(custosArgs, custosStdout, custosStderr)
		if exitCode != 0 {
			t.Errorf("custosd exited with non-zero code: %d\nStdout: %s\nStderr: %s", exitCode, custosStdout.String(), custosStderr.String())
		}
	}()
	waitForCustos(t, custosEndpoint)

	viewerToken := signJWT(auth.RoleViewer, issuer, audience)
	operatorToken := signJWT(auth.RoleOperator, issuer, audience)

	if status, body := postIngestStatus(t, custosEndpoint, viewerToken, createDummyBundle(t, "viewer-denied")); status != http.StatusForbidden {
		t.Fatalf("viewer ingest status = %d, want %d; body=%s", status, http.StatusForbidden, body)
	}
	if err := ingestBundle(t, custosEndpoint, operatorToken, createDummyBundle(t, "operator-accepted")); err != nil {
		t.Fatalf("operator JWT ingest: %v", err)
	}
	receipts, err := listCustosReceipts(t, custosEndpoint, viewerToken)
	if err != nil {
		t.Fatalf("viewer JWT list receipts: %v", err)
	}
	if len(receipts) != 1 {
		t.Fatalf("receipt count = %d, want 1", len(receipts))
	}
}

func getFreePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not resolve free port: %v", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("could not listen on free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForCustos(t *testing.T, endpoint string) {
	client := &http.Client{Timeout: 1 * time.Second}
	for i := 0; i < 10; i++ {
		resp, err := client.Get(endpoint + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("Custos daemon did not start within timeout at %s", endpoint)
}

func ingestBundle(t *testing.T, endpoint, token string, bundleBytes []byte) error {
	t.Helper()
	status, body := postIngestStatus(t, endpoint, token, bundleBytes)
	if status != http.StatusCreated {
		return fmt.Errorf("ingest returned status %d: %s", status, body)
	}
	return nil
}

func postIngestStatus(t *testing.T, endpoint, token string, bundleBytes []byte) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint+"/ingest", bytes.NewReader(bundleBytes))
	if err != nil {
		t.Fatalf("create ingest request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send ingest request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read ingest response: %v", err)
	}
	return resp.StatusCode, string(body)
}

func listCustosReceipts(t *testing.T, endpoint, token string) ([]receipt.Receipt, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), "GET", endpoint+"/receipts", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request to Custos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Custos returned non-200 status: %d %s, body: %s", resp.StatusCode, resp.Status, body)
	}

	var result struct {
		Count    int               `json:"count"`
		Receipts []receipt.Receipt `json:"receipts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result.Receipts, nil
}

func createDummyBundle(t *testing.T, label string) []byte {
	t.Helper()
	events := []testEvent{
		{
			Sequence: 0,
			PrevHash: genesisHash,
			Type:     "atb.bundle.manifest",
			HashAlgo: "sha256",
			Data: map[string]any{
				"version":    2,
				"created_at": "2026-06-16T00:00:00Z",
				"bundle_id":  "00000000000000000000000000000001",
			},
		},
		{
			Sequence: 1,
			PrevHash: genesisHash,
			Type:     "dev.test.event",
			HashAlgo: "sha256",
			Data: map[string]string{
				"message": label,
			},
		},
	}

	var out bytes.Buffer
	prev := genesisHash
	for i := range events {
		events[i].PrevHash = prev
		sum, err := computeTestEventHash(events[i])
		if err != nil {
			t.Fatalf("compute test bundle hash: %v", err)
		}
		record := testRecord{Event: events[i], Hash: sum}
		line, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal test bundle record: %v", err)
		}
		out.Write(line)
		out.WriteByte('\n')
		prev = sum
	}
	return out.Bytes()
}

const genesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

type testRecord struct {
	Event testEvent `json:"event"`
	Hash  string    `json:"hash"`
}

type testEvent struct {
	Sequence int    `json:"seq"`
	PrevHash string `json:"prev_hash"`
	Type     string `json:"type"`
	HashAlgo string `json:"hash_algo"`
	Data     any    `json:"data,omitempty"`
}

func computeTestEventHash(event testEvent) (string, error) {
	canonical, err := jcs.Marshal(event)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write([]byte(event.PrevHash))
	h.Write(canonical)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func startOIDCTestIssuer(t *testing.T) (issuer string, signJWT func(role auth.Role, issuer, audience string) string, cleanup func()) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	publicKey := privateKey.PublicKey
	jwkKey, err := jwk.FromRaw(&publicKey)
	if err != nil {
		t.Fatalf("create JWK: %v", err)
	}
	const kid = "custos-integration-kid"
	_ = jwkKey.Set("kid", kid)
	jwks := jwk.NewSet()
	jwks.AddKey(jwkKey)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	}))

	return server.URL, func(role auth.Role, issuer, audience string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss":  issuer,
			"aud":  audience,
			"exp":  time.Now().Add(time.Hour).Unix(),
			"iat":  time.Now().Unix(),
			"role": string(role),
		})
		token.Header["kid"] = kid
		signed, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("sign JWT: %v", err)
		}
		return signed
	}, server.Close
}
