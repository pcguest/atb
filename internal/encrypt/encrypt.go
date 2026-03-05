// Package encrypt provides authenticated encryption for ATB bundles.
//
// Wire format (v1):
// [4 bytes magic "ATBE"][1 byte version][16 bytes salt][12 bytes nonce][16 bytes auth tag][N bytes ciphertext]
package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

const (
	Magic                 = "ATBE"
	Version          byte = 0x01
	SaltSize              = 16
	NonceSize             = 12
	TagSize               = 16
	KeySize               = 32
	PBKDF2Iterations      = 100_000
	HeaderSize            = len(Magic) + 1 + SaltSize + NonceSize + TagSize
)

var (
	ErrInvalidFormat      = errors.New("encrypt: invalid format")
	ErrUnsupportedVersion = errors.New("encrypt: unsupported version")
	ErrDecryptFailed      = errors.New("encrypt: decrypt failed")
)

// Encrypt encrypts plaintext using AES-256-GCM with a random salt and nonce.
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("encrypt: random salt: %w", err)
	}
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("encrypt: random nonce: %w", err)
	}
	return EncryptWithSaltNonce(plaintext, password, salt, nonce)
}

// EncryptWithSaltNonce encrypts plaintext with caller-provided salt and nonce.
// This is intended for deterministic test vectors and golden fixtures.
func EncryptWithSaltNonce(plaintext []byte, password string, salt []byte, nonce []byte) ([]byte, error) {
	if len(salt) != SaltSize {
		return nil, fmt.Errorf("encrypt: salt must be %d bytes", SaltSize)
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("encrypt: nonce must be %d bytes", NonceSize)
	}
	if password == "" {
		return nil, fmt.Errorf("encrypt: password cannot be empty")
	}

	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encrypt: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encrypt: gcm: %w", err)
	}

	sealed := gcm.Seal(nil, nonce, plaintext, nil)
	if len(sealed) < TagSize {
		return nil, fmt.Errorf("encrypt: invalid gcm output")
	}
	tagOffset := len(sealed) - TagSize
	ciphertext := sealed[:tagOffset]
	tag := sealed[tagOffset:]

	out := make([]byte, HeaderSize+len(ciphertext))
	offset := 0
	copy(out[offset:offset+len(Magic)], []byte(Magic))
	offset += len(Magic)
	out[offset] = Version
	offset++
	copy(out[offset:offset+SaltSize], salt)
	offset += SaltSize
	copy(out[offset:offset+NonceSize], nonce)
	offset += NonceSize
	copy(out[offset:offset+TagSize], tag)
	offset += TagSize
	copy(out[offset:], ciphertext)
	return out, nil
}

// Decrypt decrypts an encrypted ATB bundle payload.
func Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < HeaderSize {
		return nil, ErrInvalidFormat
	}
	if password == "" {
		return nil, fmt.Errorf("encrypt: password cannot be empty")
	}

	offset := 0
	if string(data[offset:offset+len(Magic)]) != Magic {
		return nil, ErrInvalidFormat
	}
	offset += len(Magic)

	version := data[offset]
	offset++
	if version != Version {
		return nil, fmt.Errorf("%w: got 0x%02x", ErrUnsupportedVersion, version)
	}

	salt := data[offset : offset+SaltSize]
	offset += SaltSize
	nonce := data[offset : offset+NonceSize]
	offset += NonceSize
	tag := data[offset : offset+TagSize]
	offset += TagSize
	ciphertext := data[offset:]

	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encrypt: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encrypt: gcm: %w", err)
	}

	sealed := make([]byte, len(ciphertext)+TagSize)
	copy(sealed[:len(ciphertext)], ciphertext)
	copy(sealed[len(ciphertext):], tag)

	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptFailed, err)
	}
	return plaintext, nil
}
