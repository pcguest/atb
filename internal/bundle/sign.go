// SPDX-License-Identifier: MIT
package bundle

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
	"github.com/pcguest/atb/internal/signer"
)

var atomicWrite = writeAtomic

// Sign appends an Ed25519 signature record to the bundle in place.
func Sign(bundlePath string, privateKey ed25519.PrivateKey) error {
	_, err := SignTo(bundlePath, bundlePath, privateKey)
	return err
}

// SignTo appends an Ed25519 signature record using a local Ed25519 private
// key and writes the result to outputPath. Preserved for backward
// compatibility with existing callers; new code should prefer
// SignToWithSigner.
func SignTo(bundlePath string, outputPath string, privateKey ed25519.PrivateKey) (string, error) {
	return SignToWithSigner(context.Background(), bundlePath, outputPath, signer.NewLocalSigner(privateKey))
}

// SignToWithSigner appends an Ed25519 (or future-algorithm) signature
// record to the bundle at bundlePath and writes the result to outputPath.
// All key operations are delegated to s. The on-the-wire pre-image is the
// 32-byte SHA-256 digest of the pre-signature bundle NDJSON bytes; that
// digest is what is passed to s.Sign.
func SignToWithSigner(ctx context.Context, bundlePath, outputPath string, s signer.Signer) (string, error) {
	return SignToWithSignerRetry(ctx, bundlePath, outputPath, s, 0, 0)
}

// SignToWithSignerRetry is SignToWithSigner with opt-in bounded waiting for
// lock contention. A zero lockWait keeps the existing fail-fast behaviour.
func SignToWithSignerRetry(
	ctx context.Context,
	bundlePath string,
	outputPath string,
	s signer.Signer,
	lockWait time.Duration,
	lockInterval time.Duration,
) (string, error) {
	if s == nil {
		return "", fmt.Errorf("signer is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if lockWait <= 0 {
		var bundleHash string
		err := withBundleLock(bundlePath, func() error {
			var err error
			bundleHash, err = signToWithSignerUnlocked(ctx, bundlePath, outputPath, s)
			return err
		})
		if errors.Is(err, ErrBundleLocked) {
			return "", fmt.Errorf("sign bundle: %w", err)
		}
		return bundleHash, err
	}

	// Hold the lock across read + verify + sign + write so a concurrent
	// appender cannot mutate the bundle between digest computation and
	// the signature record being written. The output path may differ from
	// the bundle path; we lock the input only because that's the file we
	// just hashed. If outputPath != bundlePath, the writer of the output
	// is the only writer there in normal CLI usage.
	release, lockErr := AcquireWithRetry(ctx, bundlePath, lockWait, lockInterval)
	if lockErr != nil {
		return "", fmt.Errorf("sign bundle: %w", lockErr)
	}
	defer func() {
		_ = release()
	}()

	return signToWithSignerUnlocked(ctx, bundlePath, outputPath, s)
}

func signToWithSignerUnlocked(ctx context.Context, bundlePath string, outputPath string, s signer.Signer) (string, error) {
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

	signature, pubKey, keyID, backend, algorithm, err := s.Sign(ctx, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign bundle digest: %w", err)
	}
	if len(signature) == 0 {
		return "", fmt.Errorf("sign bundle digest: empty signature")
	}
	switch algorithm {
	case "", "ed25519":
		if len(pubKey) != ed25519.PublicKeySize {
			return "", fmt.Errorf("sign bundle digest: pubkey length %d, want %d", len(pubKey), ed25519.PublicKeySize)
		}
		algorithm = "ed25519"
	case "ecdsa-p256":
		if len(pubKey) != 65 {
			return "", fmt.Errorf("sign bundle digest: pubkey length %d, want 65 (ECDSA P-256 uncompressed)", len(pubKey))
		}
	default:
		return "", fmt.Errorf("sign bundle digest: unsupported signing algorithm %q", algorithm)
	}

	signedAt := time.Now().UTC().Format(time.RFC3339)
	record := signpkg.NewBundleSignatureRecord(digest[:], pubKey, signature, keyID, backend, algorithm, signedAt)
	if err := b.AppendWithOptions(event.TypeBundleSignature, record, &AppendOptions{
		Timestamp: signedAt,
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

	sourceMode := os.FileMode(0o600)
	if filepath.Clean(sourcePath) == filepath.Clean(outputPath) {
		if info, err := os.Stat(sourcePath); err == nil {
			sourceMode = info.Mode().Perm()
		}
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
	if err := atomicWrite(outputPath, payload, sourceMode); err != nil {
		return fmt.Errorf("write signed bundle: %w", err)
	}
	return nil
}
