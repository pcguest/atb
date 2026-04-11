package verify

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pcguest/atb/internal/bundle"
	"github.com/pcguest/atb/internal/event"
)

// SnapshotBundleHash reproduces the hash written by atb snapshot for the
// current bundle prefix.
func SnapshotBundleHash(records []bundle.Record) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, record := range records {
		if err := enc.Encode(record); err != nil {
			return "", err
		}
	}
	sum := sha256.Sum256(buf.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

// VerifySnapshotHashes recomputes each atb.snapshot bundle_hash against the
// bundle state that existed immediately before that snapshot record.
func VerifySnapshotHashes(records []bundle.Record) []string {
	failures := []string{}

	for i, record := range records {
		if record.Event.Type != event.TypeSnapshot {
			continue
		}

		seq := record.Event.Sequence
		fields, ok := record.Event.Data.(map[string]any)
		if !ok {
			failures = append(failures, fmt.Sprintf("snapshot_hash_mismatch at seq %d: snapshot data is not an object", seq))
			continue
		}

		recordedHash, _ := fields["bundle_hash"].(string)
		recordedHash = strings.TrimSpace(recordedHash)
		if recordedHash == "" {
			failures = append(failures, fmt.Sprintf("snapshot_hash_mismatch at seq %d: bundle_hash missing", seq))
			continue
		}

		computedHash, err := SnapshotBundleHash(records[:i])
		if err != nil {
			failures = append(failures, fmt.Sprintf("snapshot_hash_mismatch at seq %d: recompute bundle hash: %v", seq, err))
			continue
		}
		if recordedHash != computedHash {
			failures = append(failures, fmt.Sprintf("snapshot_hash_mismatch at seq %d", seq))
		}
	}

	return failures
}
