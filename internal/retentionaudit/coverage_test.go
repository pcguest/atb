// SPDX-License-Identifier: MIT
package retentionaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDigestContracts(t *testing.T) {
	first, err := Digest(map[string]any{"retention_days": 30})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Digest(map[string]any{"retention_days": 30})
	if err != nil || first == "" || first != second {
		t.Fatalf("first=%q second=%q err=%v", first, second, err)
	}
	if _, err := Digest(func() {}); err == nil || !strings.Contains(err.Error(), "marshal digest input") {
		t.Fatalf("unmarshalable digest error=%v", err)
	}
}

func TestAppendRejectsCorruptBundleAndAcceptsZeroTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "operations.atb")
	if err := Append(path, "atb.retention.test", map[string]any{"ok": true}, time.Time{}); err != nil {
		t.Fatalf("Append zero time: %v", err)
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := Append(path, "atb.retention.test", map[string]any{"ok": true}, time.Now())
	if err == nil || !strings.Contains(err.Error(), "retention audit: load") {
		t.Fatalf("corrupt append error=%v", err)
	}
}
