// SPDX-License-Identifier: MIT
package bundle

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/pcguest/atb/internal/event"
	signpkg "github.com/pcguest/atb/internal/sign"
	"github.com/pcguest/atb/internal/signer"
)

var atomicWrite = writeAtomic

// Sign appends an Ed25519 signature record to the bundle in place.
func Sign(args ...any) error {
	ctx, bundlePath, privateKey, err := signArgs(args)
	if err != nil {
		return err
	}
	_, err = SignTo(ctx, bundlePath, bundlePath, privateKey)
	return err
}

// SignTo appends an Ed25519 signature record using a local Ed25519 private
// key and writes the result to outputPath. Preserved for backward
// compatibility with existing callers; new code should prefer
// SignToWithSigner.
func SignTo(args ...any) (string, error) {
	ctx, bundlePath, outputPath, privateKey, err := signToArgs(args)
	if err != nil {
		return "", err
	}
	return SignToWithSigner(ctx, bundlePath, outputPath, signer.NewLocalSigner(privateKey))
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
	if err := ctx.Err(); err != nil {
		return "", err
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
	return signToWithSignerLimited(ctx, bundlePath, outputPath, s, MaxBundleSizeBytes)
}

// signToWithSignerLimited keeps the read bound explicit so boundary tests do not
// need large inputs or a mutable process-wide limit. The caller holds the lock.
func signToWithSignerLimited(
	ctx context.Context,
	bundlePath string,
	outputPath string,
	s signer.Signer,
	maxBytes int64,
) (string, error) {
	// Open the bundle once and derive everything from that single read so the
	// digest, the parsed bundle, and the file mode all describe the same on-
	// disk bytes. A previous implementation called os.ReadFile and then
	// Load(bundlePath), opening the file twice; concurrent appenders could
	// slip a write between the two opens (TOCTOU). Holding the advisory
	// bundle lock around this function additionally prevents concurrent
	// atb writers in cooperating processes.
	f, err := os.Open(filepath.Clean(bundlePath)) // #nosec G304 -- caller supplies the bundle path explicitly
	if err != nil {
		return "", fmt.Errorf("read bundle: %w", err)
	}
	rawBundle, err := readSigningInput(f, maxBytes)
	if err != nil {
		_ = f.Close()
		return "", fmt.Errorf("read bundle: %w", err)
	}
	if err := ctx.Err(); err != nil {
		_ = f.Close()
		return "", err
	}
	sourceMode := bundleFileMode
	if info, statErr := f.Stat(); statErr == nil {
		sourceMode = info.Mode().Perm()
	}
	if cerr := f.Close(); cerr != nil {
		return "", fmt.Errorf("read bundle: close: %w", cerr)
	}

	b, err := LoadReader(bytes.NewReader(rawBundle))
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
	outputMode := bundleFileMode
	if filepath.Clean(bundlePath) == filepath.Clean(outputPath) {
		outputMode = sourceMode
	}
	if err := appendSignedRecord(outputPath, rawBundle, signedRecord, outputMode, maxBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(digest[:]), nil
}

func readSigningInput(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 || maxBytes == math.MaxInt64 {
		return nil, fmt.Errorf("sign bundle: invalid read limit: %w", ErrResourceLimit)
	}
	// Probe one byte beyond the policy, without trusting a stat/check-then-read
	// size or draining an oversized source. Hash only the accepted original bytes.
	raw, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("sign bundle: input exceeds %d bytes: %w", maxBytes, ErrResourceLimit)
	}
	return raw, nil
}

func signArgs(args []any) (context.Context, string, ed25519.PrivateKey, error) {
	switch len(args) {
	case 2:
		bundlePath, ok := args[0].(string)
		if !ok {
			return nil, "", nil, fmt.Errorf("sign bundle: expected bundle path string")
		}
		privateKey, ok := args[1].(ed25519.PrivateKey)
		if !ok {
			return nil, "", nil, fmt.Errorf("sign bundle: expected Ed25519 private key")
		}
		return context.Background(), bundlePath, privateKey, nil
	case 3:
		ctx, ok := args[0].(context.Context)
		if !ok {
			return nil, "", nil, fmt.Errorf("sign bundle: expected context.Context")
		}
		bundlePath, ok := args[1].(string)
		if !ok {
			return nil, "", nil, fmt.Errorf("sign bundle: expected bundle path string")
		}
		privateKey, ok := args[2].(ed25519.PrivateKey)
		if !ok {
			return nil, "", nil, fmt.Errorf("sign bundle: expected Ed25519 private key")
		}
		if ctx == nil {
			ctx = context.Background()
		}
		return ctx, bundlePath, privateKey, nil
	default:
		return nil, "", nil, fmt.Errorf("sign bundle: expected path and key or context, path, and key")
	}
}

func signToArgs(args []any) (context.Context, string, string, ed25519.PrivateKey, error) {
	switch len(args) {
	case 3:
		bundlePath, ok := args[0].(string)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected bundle path string")
		}
		outputPath, ok := args[1].(string)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected output path string")
		}
		privateKey, ok := args[2].(ed25519.PrivateKey)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected Ed25519 private key")
		}
		return context.Background(), bundlePath, outputPath, privateKey, nil
	case 4:
		ctx, ok := args[0].(context.Context)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected context.Context")
		}
		bundlePath, ok := args[1].(string)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected bundle path string")
		}
		outputPath, ok := args[2].(string)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected output path string")
		}
		privateKey, ok := args[3].(ed25519.PrivateKey)
		if !ok {
			return nil, "", "", nil, fmt.Errorf("sign bundle: expected Ed25519 private key")
		}
		if ctx == nil {
			ctx = context.Background()
		}
		return ctx, bundlePath, outputPath, privateKey, nil
	default:
		return nil, "", "", nil, fmt.Errorf("sign bundle: expected paths and key or context, paths, and key")
	}
}

func appendSignedRecord(outputPath string, rawBundle []byte, record Record, outputMode os.FileMode, maxBytes int64) error {
	encodedRecord, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode signature record: %w", err)
	}

	needsSeparator := len(rawBundle) > 0 && rawBundle[len(rawBundle)-1] != '\n'
	outputSize, capacity, err := signedPayloadMetrics(len(rawBundle), len(encodedRecord), needsSeparator)
	if err != nil {
		return err
	}
	if maxBytes <= 0 {
		return fmt.Errorf("sign bundle: invalid output limit: %w", ErrResourceLimit)
	}
	if int64(outputSize) > maxBytes {
		return fmt.Errorf("sign bundle: output exceeds %d bytes: %w", maxBytes, ErrResourceLimit)
	}

	payload := make([]byte, 0, capacity)
	payload = append(payload, rawBundle...)
	if needsSeparator {
		payload = append(payload, '\n')
	}
	payload = append(payload, encodedRecord...)
	payload = append(payload, '\n')

	if err := os.MkdirAll(filepath.Dir(outputPath), 0750); err != nil { // #nosec G301 -- tightened directory permissions for bundle writes
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := atomicWrite(outputPath, payload, outputMode); err != nil {
		return fmt.Errorf("write signed bundle: %w", err)
	}
	return nil
}

func signedPayloadCapacity(rawSize, recordSize int) (int, error) {
	// Reserve both possible newlines before adding lengths. Subtractions are
	// checked in order, so neither the guard nor the final sum can overflow.
	if rawSize < 0 || recordSize < 0 || rawSize > math.MaxInt-2 {
		return 0, fmt.Errorf("sign bundle: output size exceeds capacity: %w", ErrResourceLimit)
	}
	if recordSize > math.MaxInt-2-rawSize {
		return 0, fmt.Errorf("sign bundle: output size exceeds capacity: %w", ErrResourceLimit)
	}
	return rawSize + recordSize + 2, nil
}

func signedPayloadMetrics(rawSize, recordSize int, needsSeparator bool) (int, int, error) {
	// capacity reserves the separator and final LF. The actual emitted size is
	// one byte smaller when no separator is needed; an empty input never needs
	// one. Both values are derived after the overflow guard above.
	capacity, err := signedPayloadCapacity(rawSize, recordSize)
	if err != nil {
		return 0, 0, err
	}
	outputSize := capacity - 1
	if needsSeparator && rawSize > 0 {
		outputSize = capacity
	}
	return outputSize, capacity, nil
}
