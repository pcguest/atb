// SPDX-License-Identifier: MIT
package golden

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pcguest/atb/internal/canonicalize"
	"github.com/pcguest/atb/internal/encrypt"
	"github.com/pcguest/atb/internal/hash"
)

const (
	encryptParityPassword = "test-password-for-parity"
)

func TestEncryptParity_AllSDKs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cross-SDK parity build in short mode")
	}

	repoRoot := mustRepoRoot(t)

	plaintext := canonicalParityPayload(t)
	salt := bytes.Repeat([]byte{0x42}, encrypt.SaltSize)
	nonce := bytes.Repeat([]byte{0x7A}, encrypt.NonceSize)

	goCiphertext, err := encrypt.EncryptWithSaltNonce(
		plaintext,
		encryptParityPassword,
		salt,
		nonce,
	)
	if err != nil {
		t.Fatalf("go encrypt parity baseline: %v", err)
	}

	pyCiphertext := runPythonEncrypt(t, repoRoot, plaintext, encryptParityPassword, salt, nonce)
	if !bytes.Equal(goCiphertext, pyCiphertext) {
		t.Fatalf(
			"go/python parity mismatch\n go=%s\n py=%s",
			hex.EncodeToString(goCiphertext),
			hex.EncodeToString(pyCiphertext),
		)
	}

	tsCiphertext := runTypeScriptEncrypt(t, repoRoot, plaintext, encryptParityPassword, salt, nonce)
	if !bytes.Equal(goCiphertext, tsCiphertext) {
		t.Fatalf(
			"go/typescript parity mismatch\n go=%s\n ts=%s",
			hex.EncodeToString(goCiphertext),
			hex.EncodeToString(tsCiphertext),
		)
	}

	goldenPath := filepath.Join(repoRoot, "test", "golden", "encrypt-vector.hex")
	goldenHex := hex.EncodeToString(goCiphertext)
	if os.Getenv("ATB_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(goldenHex), 0o644); err != nil {
			t.Fatalf("write encryption golden fixture: %v", err)
		}
	}
	existing, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read encryption golden fixture: %v", err)
	}
	if strings.TrimSpace(string(existing)) != goldenHex {
		t.Fatalf(
			"encryption golden fixture mismatch\n got:  %s\n want: %s\n"+
				"set ATB_UPDATE_GOLDEN=1 to refresh fixture",
			strings.TrimSpace(string(existing)),
			goldenHex,
		)
	}
}

func canonicalParityPayload(t *testing.T) []byte {
	t.Helper()
	payload := map[string]any{
		"head_hash": hash.GenesisHash,
		"records":   []any{},
	}
	canonical, err := canonicalize.Marshal(payload)
	if err != nil {
		t.Fatalf("canonicalize parity payload: %v", err)
	}
	return canonical
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Package runs from test/golden.
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func runPythonEncrypt(
	t *testing.T,
	repoRoot string,
	plaintext []byte,
	password string,
	salt []byte,
	nonce []byte,
) []byte {
	t.Helper()

	pythonBin := os.Getenv("ATB_PYTHON_BIN")
	if pythonBin == "" {
		for _, candidate := range []string{
			filepath.Join(repoRoot, ".venv", "bin", "python"),
			filepath.Join(repoRoot, "sdk", "python", "venv", "bin", "python"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				pythonBin = candidate
				break
			}
		}
		if pythonBin == "" {
			pythonBin = "python3"
		}
	}
	scriptPath := filepath.Join(repoRoot, "test", "golden", "encrypt_parity.py")
	hexOut := runParityCommand(
		t,
		repoRoot,
		pythonBin,
		[]string{scriptPath},
		plaintext,
		password,
		salt,
		nonce,
	)
	out, err := hex.DecodeString(hexOut)
	if err != nil {
		t.Fatalf("decode python parity output: %v", err)
	}
	return out
}

func runTypeScriptEncrypt(
	t *testing.T,
	repoRoot string,
	plaintext []byte,
	password string,
	salt []byte,
	nonce []byte,
) []byte {
	t.Helper()

	tsDir := filepath.Join(repoRoot, "sdk", "typescript")
	build := exec.Command("npm", "run", "build")
	build.Dir = tsDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("typescript build for parity failed: %v\n%s", err, string(out))
	}

	scriptPath := filepath.Join(repoRoot, "test", "golden", "encrypt_parity.js")
	hexOut := runParityCommand(
		t,
		repoRoot,
		"node",
		[]string{scriptPath},
		plaintext,
		password,
		salt,
		nonce,
	)
	out, err := hex.DecodeString(hexOut)
	if err != nil {
		t.Fatalf("decode typescript parity output: %v", err)
	}
	return out
}

func runParityCommand(
	t *testing.T,
	repoRoot string,
	bin string,
	args []string,
	plaintext []byte,
	password string,
	salt []byte,
	nonce []byte,
) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("ATB_PARITY_PLAINTEXT_B64=%s", base64.StdEncoding.EncodeToString(plaintext)),
		fmt.Sprintf("ATB_PARITY_PASSWORD=%s", password),
		fmt.Sprintf("ATB_PARITY_SALT_HEX=%s", hex.EncodeToString(salt)),
		fmt.Sprintf("ATB_PARITY_NONCE_HEX=%s", hex.EncodeToString(nonce)),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("parity command %s failed: %v\n%s", bin, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
