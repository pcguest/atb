package bundle_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pcguest/atb/internal/bundle"
)

func FuzzLoad(f *testing.F) {
	seedBundle, err := bundle.New()
	if err != nil {
		f.Fatalf("create seed bundle: %v", err)
	}
	if len(seedBundle.Records) != 1 {
		f.Fatalf("seed bundle record count = %d, want 1", len(seedBundle.Records))
	}

	record, err := json.Marshal(seedBundle.Records[0])
	if err != nil {
		f.Fatalf("marshal seed bundle record: %v", err)
	}

	seeds := []string{
		string(append(record, '\n')),
		"",
		"\n",
		`{"event":{"seq":1,"prev_hash":"0000000000000000000000000000000000000000000000000000000000000000","type":"dev.session","data":"x"},"hash":"invalid"}`,
		`not json at all`,
		`{}`,
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Load panicked: %v", r)
			}
		}()

		path := filepath.Join(t.TempDir(), "bundle.atb")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write fuzz input: %v", err)
		}

		_, _ = bundle.Load(path)
	})
}
