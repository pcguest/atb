package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pcguest/atb/pkg/custody"
	// Fixed: Custos tests import Custos internals through the Custos module path.
	"github.com/pcguest/custos/internal/ingest"
	// Fixed: Custos tests import Custos internals through the Custos module path.
	"github.com/pcguest/custos/internal/receipt"
)

// TestCustosIngestAndRetrieve tests the full ingest and retrieve flow.
func TestCustosIngestAndRetrieve(t *testing.T) {
	// Start a test HTTP server for custosd
	wormStore := receipt.NewInMemoryWORMStore()
	receiptStore := receipt.NewInMemoryReceiptStore()

	ingestHandler := ingest.IngestHandler{
		WORMStore:    wormStore,
		ReceiptStore: receiptStore,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		profileID := r.URL.Query().Get("profile_id")
		ingestHandler.ProfileID = profileID

		rec, err := ingestHandler.Handle(r.Context(), r.Body)
		if err != nil {
			if errors.Is(err, ingest.ErrEmptyBody) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, ingest.ErrInvalidBundle) {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rec)
	})

	mux.HandleFunc("/receipts/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/receipts/"), "/")
		if len(pathParts) < 1 || pathParts[0] == "" {
			http.Error(w, "Receipt ID required", http.StatusBadRequest)
			return
		}
		receiptID := pathParts[0]

		if len(pathParts) == 2 && pathParts[1] == "verify" {
			handleVerifyReceipt(w, r, receiptID, wormStore)
			return
		}
		handleGetReceipt(w, r, receiptID, receiptStore)
	})

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Load an example bundle
	exampleBundlePath := filepath.Join("..", "..", "examples", "bundles", "profiles", "privileged_tool_action-pass.atb")
	bundleBytes, err := ioutil.ReadFile(exampleBundlePath)
	if err != nil {
		t.Fatalf("Failed to read example bundle: %v", err)
	}
	// Fixed: The expected head hash is derived through the public custody API.
	expectedExport, err := custody.NewBundleExport(exampleBundlePath, custody.ExportOptions{ProfileID: "atb.profile.privileged_tool_action"})
	if err != nil {
		t.Fatalf("Failed to evaluate expected bundle export: %v", err)
	}

	// 1. Test POST /ingest
	ingestURL := fmt.Sprintf("%s/ingest?profile_id=atb.profile.privileged_tool_action", testServer.URL)
	req, err := http.NewRequest(http.MethodPost, ingestURL, bytes.NewReader(bundleBytes))
	if err != nil {
		t.Fatalf("Failed to create ingest request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send ingest request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusCreated, resp.StatusCode, string(bodyBytes))
	}

	var rec receipt.Receipt
	if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
		t.Fatalf("Failed to decode receipt: %v", err)
	}
	if rec.ReceiptID == "" {
		t.Fatal("Expected a receipt ID, got empty")
	}
	if rec.BundleHash == "" {
		t.Fatal("Expected a bundle hash, got empty")
	}
	// Fixed: The receipt bundle hash must match the public custody head hash.
	if rec.BundleHash != expectedExport.BundleHash {
		t.Fatalf("Expected bundle hash %q, got %q", expectedExport.BundleHash, rec.BundleHash)
	}
	if rec.ProfileID != "atb.profile.privileged_tool_action" {
		t.Errorf("Expected profile ID 'atb.profile.privileged_tool_action', got %s", rec.ProfileID)
	}
	if len(rec.VerifyReport) == 0 {
		t.Error("Expected verify report, got empty")
	}

	// 2. Test GET /receipts/:id
	getReceiptURL := fmt.Sprintf("%s/receipts/%s", testServer.URL, rec.ReceiptID)
	resp, err = http.Get(getReceiptURL)
	if err != nil {
		t.Fatalf("Failed to send get receipt request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.StatusCode, string(bodyBytes))
	}

	var fetchedRec receipt.Receipt
	if err := json.NewDecoder(resp.Body).Decode(&fetchedRec); err != nil {
		t.Fatalf("Failed to decode fetched receipt: %v", err)
	}
	if !reflect.DeepEqual(rec, fetchedRec) {
		t.Errorf("Fetched receipt does not match ingested receipt.\nGot: %+v\nWant: %+v", fetchedRec, rec)
	}

	// 3. Test GET /receipts/:id/verify
	verifyReceiptURL := fmt.Sprintf("%s/receipts/%s/verify", testServer.URL, rec.ReceiptID)
	resp, err = http.Get(verifyReceiptURL)
	if err != nil {
		t.Fatalf("Failed to send verify receipt request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		t.Fatalf("Expected status %d, got %d. Body: %s", http.StatusOK, resp.StatusCode, string(bodyBytes))
	}

	// Fixed: Decode the public custody verifier report instead of importing atb/internal/verify.
	var verifierReport custody.VerifierReport
	if err := json.NewDecoder(resp.Body).Decode(&verifierReport); err != nil {
		t.Fatalf("Failed to decode verifier report: %v", err)
	}
	if !verifierReport.GateResult.ChainValid {
		t.Error("Expected re-verified bundle to be chain valid")
	}
	// Fixed: The re-verification report must use the stable public report contract.
	if verifierReport.ReportVersion != custody.VerifyReportVersion {
		t.Errorf("Expected report version %q, got %s", custody.VerifyReportVersion, verifierReport.ReportVersion)
	}
}

// handleGetReceipt and handleVerifyReceipt are copied from custos/cmd/custosd/main.go for testing purposes.
func handleGetReceipt(w http.ResponseWriter, r *http.Request, receiptID string, store receipt.ReceiptStore) {
	rec, err := store.GetReceipt(r.Context(), receiptID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Receipt not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}

func handleVerifyReceipt(w http.ResponseWriter, r *http.Request, receiptID string, wormStore receipt.WORMStore) {
	bundleBytes, err := wormStore.Retrieve(r.Context(), receiptID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Bundle not found for receipt ID", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	tmp, err := os.CreateTemp("", "custos-verify-*.atb")
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(bundleBytes); err != nil {
		_ = tmp.Close()
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmp.Close(); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	export, err := custody.NewBundleExport(path, custody.ExportOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Re-verification failed: %v", err), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Fixed: BundleExport exposes VerifyReport as the public custody report field.
	json.NewEncoder(w).Encode(export.VerifyReport)
}
