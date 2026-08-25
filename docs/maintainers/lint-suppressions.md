# Lint suppressions

This file lists every active `//lint:ignore` (staticcheck) and `//nolint`
(go vet / golangci-lint) directive in the repository, with the reason each
suppression is in place. Add a row here whenever a new suppression is
introduced; remove the row when the underlying directive is removed.

The general policy: prefer fixing findings over suppressing them. Suppressions
are reserved for findings whose only "fix" would be a behaviour-changing
migration that this codebase is not yet ready to absorb.

## internal/sign/bundle.go:190

**Finding:** staticcheck SA1019: `elliptic.Unmarshal` has been deprecated since Go 1.21.
**Reason suppressed:** the verifier consumes the SEC1 uncompressed wire format that AWS KMS, GCP Cloud KMS, and Vault all emit for P-256 public keys. The replacement package (`crypto/ecdh`) is ECDH-only and cannot produce an `*ecdsa.PublicKey`, which is the type the verifier passes to `ecdsa.Verify`. A migration would require restructuring the entire ECDSA verification path; deferred.

## internal/sign/bundle_test.go:54

**Finding:** staticcheck SA1019: `elliptic.Marshal` has been deprecated since Go 1.21.
**Reason suppressed:** test fixture serialises an `ecdsa.PrivateKey`'s public point in the same SEC1 uncompressed wire format the verifier accepts; `crypto/ecdh` cannot reach this byte sequence from an ECDSA key. Suppressed alongside the production-side suppression in `internal/sign/bundle.go`.

## internal/sign/bundle_test.go:104

**Finding:** staticcheck SA1019: `elliptic.Marshal` has been deprecated since Go 1.21.
**Reason suppressed:** same as the previous entry — second test case using the same SEC1 uncompressed encoding.

## internal/signer/vault/vault.go:263

**Finding:** staticcheck SA1019: `elliptic.Marshal` has been deprecated since Go 1.21.
**Reason suppressed:** Vault Transit returns P-256 public keys in PEM/PKIX form; we re-encode to SEC1 uncompressed (65 bytes, leading `0x04`) for the on-disk signature record. `crypto/ecdh` is ECDH-only and cannot serialise an `ecdsa.PublicKey`.

## internal/signer/awskms/awskms.go:117

**Finding:** staticcheck SA1019: `elliptic.Marshal` has been deprecated since Go 1.21.
**Reason suppressed:** AWS KMS `GetPublicKey` returns DER/PKIX-encoded P-256 keys; we re-encode to SEC1 uncompressed for the on-disk signature record, matching the format the verifier expects. `crypto/ecdh` cannot bridge the two.

## internal/signer/gcpkms/gcpkms.go:121

**Finding:** staticcheck SA1019: `elliptic.Marshal` has been deprecated since Go 1.21.
**Reason suppressed:** GCP Cloud KMS `GetPublicKey` returns PEM-encoded P-256 keys; we re-encode to SEC1 uncompressed for the on-disk signature record. Same constraint as the AWS KMS and Vault entries above.
