// SPDX-License-Identifier: MIT
package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultCADirName  = ".atb"
	defaultCACertName = "ca.crt"
	defaultCAKeyName  = "ca.key"
	caOrganisation    = "ATB Local Capture CA"
)

// LocalCA holds a locally generated MITM certificate authority.
type LocalCA struct {
	CertPath string
	KeyPath  string

	mu      sync.Mutex
	caCert  *x509.Certificate
	caKey   *ecdsa.PrivateKey
	created bool
}

// DefaultCAPaths returns the default CA certificate and key paths under the user home directory.
func DefaultCAPaths() (certPath, keyPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("proxy: resolve home directory: %w", err)
	}
	dir := filepath.Join(home, defaultCADirName)
	return filepath.Join(dir, defaultCACertName), filepath.Join(dir, defaultCAKeyName), nil
}

// LoadOrCreateLocalCA loads an existing CA or generates one on first run.
func LoadOrCreateLocalCA() (*LocalCA, error) {
	certPath, keyPath, err := DefaultCAPaths()
	if err != nil {
		return nil, err
	}
	ca := &LocalCA{CertPath: certPath, KeyPath: keyPath}
	if _, err := os.Stat(certPath); err == nil {
		if err := ca.load(); err != nil {
			return nil, err
		}
		return ca, nil
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o700); err != nil {
		return nil, fmt.Errorf("proxy: create ca directory: %w", err)
	}
	if err := ca.generate(); err != nil {
		return nil, err
	}
	ca.created = true
	return ca, nil
}

func (ca *LocalCA) load() error {
	certPEM, err := os.ReadFile(ca.CertPath)
	if err != nil {
		return fmt.Errorf("proxy: read ca cert: %w", err)
	}
	keyPEM, err := os.ReadFile(ca.KeyPath)
	if err != nil {
		return fmt.Errorf("proxy: read ca key: %w", err)
	}
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return errors.New("proxy: decode ca cert pem")
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return errors.New("proxy: decode ca key pem")
	}
	parsedCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("proxy: parse ca cert: %w", err)
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("proxy: parse ca key: %w", err)
	}
	ca.caCert = parsedCert
	ca.caKey = key
	return nil
}

func (ca *LocalCA) generate() error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("proxy: generate ca key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("proxy: generate ca serial: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{caOrganisation},
			CommonName:   "ATB Local Capture Root CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("proxy: create ca cert: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("proxy: marshal ca key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(ca.CertPath, certPEM, 0o644); err != nil {
		return fmt.Errorf("proxy: write ca cert: %w", err)
	}
	if err := os.WriteFile(ca.KeyPath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("proxy: write ca key: %w", err)
	}
	ca.caCert = template
	ca.caKey = key
	return nil
}

// Created reports whether the CA was generated during LoadOrCreateLocalCA.
func (ca *LocalCA) Created() bool {
	if ca == nil {
		return false
	}
	return ca.created
}

// LeafCertificate returns a TLS leaf certificate for the given host name.
func (ca *LocalCA) LeafCertificate(host string) (certPEM, keyPEM []byte, err error) {
	if ca == nil {
		return nil, nil, errors.New("proxy: nil ca")
	}
	ca.mu.Lock()
	defer ca.mu.Unlock()
	if ca.caCert == nil || ca.caKey == nil {
		return nil, nil, errors.New("proxy: ca not loaded")
	}

	host = stripPort(host)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("proxy: generate leaf key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("proxy: generate leaf serial: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{caOrganisation},
			CommonName:   host,
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{host},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.caCert, &key.PublicKey, ca.caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("proxy: create leaf cert: %w", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("proxy: marshal leaf key: %w", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

// PrintInstallInstructions writes first-run CA trust instructions for common platforms.
func PrintInstallInstructions(w io.Writer, certPath string) {
	fmt.Fprintf(w, "ATB generated a local capture CA. Install %s as a trusted root once:\n", certPath)
	fmt.Fprintf(w, "  macOS:   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", certPath)
	fmt.Fprintf(w, "  Linux:   sudo cp %s /usr/local/share/ca-certificates/atb-local-ca.crt && sudo update-ca-certificates\n", certPath)
	fmt.Fprintf(w, "  Windows: certutil -addstore Root %s\n", certPath)
}

func stripPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
