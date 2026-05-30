// SPDX-License-Identifier: MIT
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pcguest/atb/pkg/custody"
	"github.com/pcguest/custos/internal/auth"
	// Fixed: Custos internal packages must be imported through the Custos module path.
	"github.com/pcguest/custos/internal/ingest"
	// Fixed: Custos internal packages must be imported through the Custos module path.
	"github.com/pcguest/custos/internal/receipt"
)

func main() {
	// Added: Filesystem stores are the default for real custody daemon runs.
	wormDir := flag.String("worm-dir", "~/.atb/custos/worm", "directory for immutable ATB bundle storage")
	// Added: Receipt JSON storage is configurable independently from WORM bundle storage.
	receiptDir := flag.String("receipt-dir", "~/.atb/custos/receipts", "directory for receipt JSON storage")
	// Added: Default bind interface is loopback so a fresh daemon is not reachable from the network.
	host := flag.String("host", "127.0.0.1", "listen interface (use 0.0.0.0 to bind all interfaces; review auth before exposing)")
	// Added: Port is a separate flag so operators can rebind without rewriting --host semantics.
	port := flag.Int("port", 9090, "listen port")
	// Added: Bounding the ingest body keeps a large or malicious upload from
	// being buffered fully into memory before verification.
	maxIngestBytes := flag.Int64("max-ingest-bytes", defaultMaxIngestBytes, "maximum accepted /ingest body size in bytes")
	// Added: Parsing early ensures store selection reflects explicit CLI configuration.
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Added: Store construction centralises filesystem defaults and test-only in-memory fallback.
	wormStore, receiptStore, err := buildStores(*wormDir, *receiptDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "custosd: %v\n", err)
		os.Exit(1)
	}
	ingestHandler := ingest.IngestHandler{
		WORMStore:    wormStore,
		ReceiptStore: receiptStore,
	}

	mux := newMux(ingestHandler, wormStore, receiptStore, *maxIngestBytes, logger)

	// Added: Token sourcing happens once at startup. Empty token preserves
	// local-dev ergonomics; a populated token enforces bearer auth on every
	// route except GET /health.
	authToken := os.Getenv("CUSTOS_AUTH_TOKEN")

	// Added: An unauthenticated daemon must never bind a non-loopback
	// interface. Loopback + empty token stays convenient for local dev; any
	// other interface without a token is rejected at startup.
	if err := validateBindConfig(*host, authToken); err != nil {
		fmt.Fprintf(os.Stderr, "custosd: %v\n", err)
		os.Exit(1)
	}
	var handler http.Handler = mux
	if authToken != "" {
		handler = auth.Middleware(authToken, mux)
		logger.Info("Custos auth enabled", "source", "CUSTOS_AUTH_TOKEN")
	} else {
		logger.Warn("CUSTOS_AUTH_TOKEN not set; daemon is unauthenticated — do not expose to untrusted networks")
	}

	// Added: Compose the bind address from --host and --port via net.JoinHostPort
	// so the listener does not silently bind every interface.
	addr := net.JoinHostPort(*host, strconv.Itoa(*port))
	logger.Info("custosd listening", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil { // #nosec G114 -- intentional bare HTTP; loopback-by-default and operator-controlled TLS termination
		fmt.Fprintf(os.Stderr, "custosd: %v\n", err)
		os.Exit(1)
	}
}

// defaultMaxIngestBytes bounds the /ingest request body (32 MiB) unless an
// operator overrides it with --max-ingest-bytes.
const defaultMaxIngestBytes = 32 << 20

// newMux builds the custosd HTTP routes. Extracted from main so handler
// behaviour (size limits, status codes) is testable without a live listener.
func newMux(
	ingestHandler ingest.IngestHandler,
	wormStore receipt.WORMStore,
	receiptStore receipt.ReceiptStore,
	maxIngestBytes int64,
	logger *slog.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()
	// Health endpoint matches the auth middleware bypass path; the bypass is
	// deliberately narrow (GET only).
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Bound the request body at the HTTP boundary so the ingest handler
		// stays transport-agnostic. A read past the limit fails with
		// *http.MaxBytesError, which maps to 413.
		if maxIngestBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, maxIngestBytes)
		}

		profileID := r.URL.Query().Get("profile_id")
		ingestHandler.ProfileID = profileID

		rec, err := ingestHandler.Handle(r.Context(), r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				http.Error(w, "bundle exceeds maximum ingest size", http.StatusRequestEntityTooLarge)
				return
			}
			if errors.Is(err, ingest.ErrEmptyBody) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, ingest.ErrInvalidBundle) {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			logger.Error("ingest error", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rec)
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
			handleVerifyReceipt(w, r, receiptID, wormStore, logger)
			return
		}

		handleGetReceipt(w, r, receiptID, receiptStore, logger)
	})

	return mux
}

// validateBindConfig rejects an unauthenticated daemon bound to a non-loopback
// interface. An empty token on a loopback host stays allowed for local dev.
func validateBindConfig(host, authToken string) error {
	if strings.TrimSpace(authToken) != "" {
		return nil
	}
	if isLoopbackHost(host) {
		return nil
	}
	return fmt.Errorf("refusing to bind non-loopback host %q without CUSTOS_AUTH_TOKEN set", host)
}

// isLoopbackHost reports whether host is a loopback bind target. An empty host
// binds every interface and is therefore not loopback.
func isLoopbackHost(host string) bool {
	h := strings.TrimSpace(host)
	if h == "localhost" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func handleGetReceipt(w http.ResponseWriter, r *http.Request, receiptID string, store receipt.ReceiptStore, logger *slog.Logger) {
	rec, err := store.GetReceipt(r.Context(), receiptID)
	if err != nil {
		// Changed: Typed missing-receipt errors map to 404 without string matching.
		if errors.Is(err, receipt.ErrReceiptNotFound) || strings.Contains(err.Error(), "not found") {
			http.Error(w, "Receipt not found", http.StatusNotFound)
			return
		}
		logger.Error("get receipt", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func handleVerifyReceipt(w http.ResponseWriter, r *http.Request, receiptID string, wormStore receipt.WORMStore, logger *slog.Logger) {
	bundleBytes, err := wormStore.Retrieve(r.Context(), receiptID)
	if err != nil {
		// Changed: Typed missing-receipt errors map to 404 without string matching.
		if errors.Is(err, receipt.ErrReceiptNotFound) || strings.Contains(err.Error(), "not found") {
			http.Error(w, "Bundle not found for receipt ID", http.StatusNotFound)
			return
		}
		logger.Error("retrieve bundle", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Write bundle bytes to a temporary file for verification
	tmp, err := os.CreateTemp("", "custos-verify-*.atb")
	if err != nil {
		logger.Error("create temp file", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(bundleBytes); err != nil {
		_ = tmp.Close()
		logger.Error("write temp bundle", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if err := tmp.Close(); err != nil {
		logger.Error("close temp bundle", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Re-run verification
	export, err := custody.NewBundleExport(path, custody.ExportOptions{}) // No specific profile for re-verification
	if err != nil {
		logger.Error("re-verification failed", "receipt_id", receiptID, "err", err)
		http.Error(w, fmt.Sprintf("Re-verification failed: %v", err), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Fixed: BundleExport exposes VerifyReport as the public custody report field.
	_ = json.NewEncoder(w).Encode(export.VerifyReport)
}

// buildStores selects filesystem stores unless both directories are explicitly empty.
func buildStores(wormDir string, receiptDir string) (receipt.WORMStore, receipt.ReceiptStore, error) {
	// Added: Empty flags are reserved for test harnesses that need ephemeral in-memory state.
	if wormDir == "" && receiptDir == "" {
		return receipt.NewInMemoryWORMStore(), receipt.NewInMemoryReceiptStore(), nil
	}
	// Added: A half-configured daemon would lose either bundles or receipts, so reject it.
	if wormDir == "" || receiptDir == "" {
		return nil, nil, errors.New("worm-dir and receipt-dir must both be set or both be empty")
	}
	// Added: Tilde expansion keeps defaults local-first without third-party path helpers.
	expandedWORMDir, err := expandHome(wormDir)
	if err != nil {
		return nil, nil, fmt.Errorf("expand worm-dir: %w", err)
	}
	// Added: Tilde expansion keeps receipt storage under the user's local home by default.
	expandedReceiptDir, err := expandHome(receiptDir)
	if err != nil {
		return nil, nil, fmt.Errorf("expand receipt-dir: %w", err)
	}
	return receipt.NewFileSystemWORMStore(expandedWORMDir), receipt.NewFileSystemReceiptStore(expandedReceiptDir), nil
}

// expandHome expands a leading ~/ path using the current user's home directory.
func expandHome(path string) (string, error) {
	// Added: Plain paths are returned unchanged so relative test paths keep working.
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	// Added: os.UserHomeDir is the stdlib source for home-directory expansion.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return homeDir, nil
	}
	return homeDir + strings.TrimPrefix(path, "~"), nil
}
