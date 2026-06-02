package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestCustosPusher_Push(t *testing.T) {
	// Create a temporary directory for test bundles
	tmpDir, err := os.MkdirTemp("", "proxy_push_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy bundle file
	dummyBundlePath := filepath.Join(tmpDir, "test.atb")
	// Fixed: Tests handle bundle.New errors after the constructor started returning an error.
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("bundle.New: %v", err)
	}
	b.AppendWithOptions("atb.bundle.manifest", map[string]interface{}{"version": "1"}, nil)
	b.Append("test.event", map[string]interface{}{"key": "value"})
	b.Save(dummyBundlePath)

	t.Run("no push when CustosEndpoint is empty", func(t *testing.T) {
		_, err := NewCustosPusher("")
		if err == nil || !strings.Contains(err.Error(), "custos endpoint is not configured") {
			t.Errorf("Expected 'custos endpoint is not configured' error, got: %v", err)
		}
		pusher := &CustosPusher{}
		err = pusher.Push(context.Background(), dummyBundlePath)
		if err == nil || !strings.Contains(err.Error(), "custos endpoint is not configured") {
			t.Errorf("Expected 'custos endpoint is not configured' error on Push, got: %v", err)
		}
	})

	t.Run("push executed on session close when endpoint is set", func(t *testing.T) {
		var pushCount atomic.Int32
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pushCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			if len(body) == 0 {
				t.Error("Expected non-empty request body")
			}
			w.Header().Set("Content-Type", "application/json")
			// Fixed: Tests use the proxy-local receipt response to avoid Custos internal imports.
			json.NewEncoder(w).Encode(pushReceipt{ReceiptID: "test-receipt-123"})
		}))
		defer mockServer.Close()

		pusher, err := NewCustosPusher(mockServer.URL)
		if err != nil {
			t.Fatalf("NewCustosPusher: %v", err)
		}
		err = pusher.Push(context.Background(), dummyBundlePath)
		if err != nil {
			t.Fatalf("Push failed: %v", err)
		}
		if pushCount.Load() != 1 {
			t.Errorf("Expected 1 push, got %d", pushCount.Load())
		}
	})

	t.Run("receipt id logging behaviour", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Fixed: Tests use the proxy-local receipt response to avoid Custos internal imports.
			json.NewEncoder(w).Encode(pushReceipt{ReceiptID: "logged-receipt-456"})
		}))
		defer mockServer.Close()

		pusher, err := NewCustosPusher(mockServer.URL)
		if err != nil {
			t.Fatalf("NewCustosPusher: %v", err)
		}
		// Capture stdout/stderr to check log output
		// Fixed: Custos success messages are written to stdout, so the test captures stdout.
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		// Fixed: Redirect stdout to the pipe that is inspected below.
		os.Stdout = w

		err = pusher.Push(context.Background(), dummyBundlePath)
		if err != nil {
			t.Fatalf("Push failed: %v", err)
		}

		w.Close()
		// Fixed: Restore stdout after the push so later tests receive normal output.
		os.Stdout = oldStdout
		out, _ := io.ReadAll(r)
		if !strings.Contains(string(out), "Custos push successful, receipt_id: logged-receipt-456") {
			t.Errorf("Expected log message with receipt ID, got: %s", string(out))
		}
	})

	t.Run("no retry on 4xx validation errors", func(t *testing.T) {
		var pushCount atomic.Int32
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pushCount.Add(1)
			http.Error(w, "Bad Request", http.StatusBadRequest)
		}))
		defer mockServer.Close()

		pusher, err := NewCustosPusher(mockServer.URL)
		if err != nil {
			t.Fatalf("NewCustosPusher: %v", err)
		}
		err = pusher.Push(context.Background(), dummyBundlePath)
		if err == nil || !strings.Contains(err.Error(), "custos push failed with client error 400") {
			t.Errorf("Expected 400 error, got: %v", err)
		}
		if pushCount.Load() != 1 {
			t.Errorf("Expected 1 push attempt for 4xx error, got %d", pushCount.Load())
		}
	})

	t.Run("one retry after a short delay for network faults (5xx)", func(t *testing.T) {
		var pushCount atomic.Int32
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := pushCount.Add(1)
			if count == 1 {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			// Fixed: Tests use the proxy-local receipt response to avoid Custos internal imports.
			json.NewEncoder(w).Encode(pushReceipt{ReceiptID: "test-receipt-retry"})
		}))
		defer mockServer.Close()

		pusher, err := NewCustosPusher(mockServer.URL)
		if err != nil {
			t.Fatalf("NewCustosPusher: %v", err)
		}
		err = pusher.Push(context.Background(), dummyBundlePath)
		if err != nil {
			t.Fatalf("Push failed: %v", err)
		}
		if pushCount.Load() != 2 {
			t.Errorf("Expected 2 push attempts for 5xx error, got %d", pushCount.Load())
		}
	})
}
