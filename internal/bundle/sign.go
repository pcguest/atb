package bundle

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
)

// Sign appends an Ed25519 signature record to the bundle in place.
func Sign(bundlePath string, privateKey ed25519.PrivateKey) error {
	_, err := SignTo(bundlePath, bundlePath, privateKey)
	return err
}

// SignTo appends an Ed25519 signature record and writes the result to outputPath.
func SignTo(bundlePath string, outputPath string, privateKey ed25519.PrivateKey) (string, error) {
	rawBundle, err := os.ReadFile(bundlePath) // #nosec G304 -- caller supplies the bundle path explicitly
	if err != nil {
		return "", fmt.Errorf("read bundle: %w", err)
	}

	b, err := Load(bundlePath)
	if err != nil {
		return "", err
	}
	if err := b.Verify(); err != nil {
		return "", err
	}

	digest := sha256.Sum256(rawBundle)

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return "", fmt.Errorf("derive public key: loaded key is not Ed25519")
	}

	signature := ed25519.Sign(privateKey, digest[:])
	if len(signature) == 0 {
		return "", fmt.Errorf("sign bundle digest: empty signature")
	}

	if err := b.AppendWithOptions(event.TypeBundleSignature, signpkg.NewBundleSignatureRecord(digest[:], publicKey, signature), &AppendOptions{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return "", err
	}

	signedRecord := b.Records[len(b.Records)-1]
	if err := appendSignedRecord(outputPath, bundlePath, rawBundle, signedRecord); err != nil {
		return "", err
	}

	return hex.EncodeToString(digest[:]), nil
}

func appendSignedRecord(outputPath string, sourcePath string, rawBundle []byte, record Record) error {
	encodedRecord, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode signature record: %w", err)
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source bundle: %w", err)
	}

	payload := make([]byte, 0, len(rawBundle)+len(encodedRecord)+2)
	payload = append(payload, rawBundle...)
	if len(payload) > 0 && payload[len(payload)-1] != '\n' {
		payload = append(payload, '\n')
	}
	payload = append(payload, encodedRecord...)
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil { // #nosec G301 -- tightened directory permissions for bundle writes
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, payload, info.Mode().Perm()); err != nil { // #nosec G304 G703 -- output path is provided explicitly by the caller
		return fmt.Errorf("write signed bundle: %w", err)
	}
	return nil
}
