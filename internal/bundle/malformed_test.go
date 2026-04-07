package bundle_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func TestLoadMalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "malformed.atb")

	// 1. Invalid JSON line
	if err := os.WriteFile(path, []byte(`{"event":{"seq":0},"hash":"abc"}`+"\n"+`not json`+"\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := bundle.Load(path)
	if err == nil || !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("expected unmarshal error, got %v", err)
	}

	// 2. Over-large payload (exceeding MaxLineSizeBytes)
	largeData := strings.Repeat("x", bundle.MaxLineSizeBytes+1024)
	largeLine := `{"event":{"seq":1,"type":"test","data":"` + largeData + `"},"hash":"abc"}` + "\n"
	if err := os.WriteFile(path, []byte(largeLine), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err = bundle.Load(path)
	if err == nil || !strings.Contains(err.Error(), "token too long") {
		t.Fatalf("expected token too long error, got %v", err)
	}
}

func TestVerifyOutOrderSequence(t *testing.T) {
	b, err := bundle.New()
	if err != nil {
		t.Fatalf("new bundle: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"step": 1}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := b.Append("ai.tool.exec", map[string]any{"step": 2}); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Manually swap sequence numbers
	b.Records[1].Event.Sequence = 2
	b.Records[2].Event.Sequence = 1

	err = b.Verify()
	if err == nil || !strings.Contains(err.Error(), "sequence mismatch") {
		t.Fatalf("expected sequence mismatch error, got %v", err)
	}
}
