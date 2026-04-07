# Key Management

This document provides guidance for security engineers and auditors on the lifecycle of cryptographic keys used in ATB for tamper evidence and bundle protection.

## Overview

In ATB, cryptographic signing is used for tamper evidence, ensuring that a bundle or specific policy event was produced by a known key holder and has not been modified since. It is not an access control mechanism.

The v0.9.x release supports:
- **Ed25519 Bundle Signing**: Signing the entire state of a hash chain to provide non-repudiation of the audit trail.
- **Policy Event Signing**: Signing individual `ai.policy.decision` events to prove the origin of a policy evaluation.
- **AES-256-GCM Encryption**: Passphrase-based encryption for protecting bundles during handoff or storage.

Out of scope for v0.9.x: Server-side key custody, multi-tenant key management, and hardware security module (HSM) integration.

## Generating a keypair

Use the `atb keygen` command to generate an Ed25519 signing keypair.

```bash
atb keygen --out-dir ./keys
```

By default, this command writes two PEM-encoded files to the specified directory (or the current directory if omitted):
- `atb-key.pem`: The private key (keep this secret).
- `atb-key.pub.pem`: The public key.

Keys are stored in raw Ed25519 PEM format. The private key is written with restricted file permissions (`0600`).

## Signing a bundle

Bundle signing captures a SHA-256 digest of the current bundle state and appends a signed record. Note that `atb bundle new` (or `atb init`) does not auto-generate a keypair; you must generate one before signing.

```bash
atb sign --bundle run.atb/bundle.atb --key ./keys/atb-key.pem
```

The signature record (`atb.bundle.signature`) includes the bundle hash, the signature, and the public key used for verification.

## Policy event signing

To ensure the integrity and origin of policy guardrails, you can sign `ai.policy.decision` events using the `--sign-policy` flag. This requires an Ed25519 private key.

```bash
atb append ai.policy.decision \
  --data '{"policy_id":"p-1","decision":"allow","action_id":"act-123"}' \
  --sign-policy ./keys/atb-key.pem
```

This operation adds a `policy_signature` and `policy_signer_pubkey` to the event payload. ATB verifies these fields during profile evaluation to ensure the decision has not been tampered with.

## Verification

`atb verify` and `atb trust-report` surface signature status during validation.

- **Bundle Signatures**: The terminal output reports:
  - `Signature present: yes/no`
  - `Signature verified: yes/no`
  If a signature is absent from the bundle, the `Bundle Signature` block is still shown in the terminal report with `Signature present: no`. In JSON output, the `bundle_signature` field is omitted if no signature record is found.
- **Policy Signatures**:
  - `ai.policy.decision: signature verified`: Displayed as an informational note when a valid signature is found.
  - `ai.policy.decision: policy_signature absent`: Displayed as a warning if a policy event is unsigned.
  - `ai.policy.decision: signature verification failed`: Displayed as a critical failure if the signature is invalid or the data has been mutated.

## Key storage recommendations

ATB is a local-first tool and does not provide a built-in secrets vault. Follow these practices:

- **Do NOT commit** `.pem` files to git. Add them to your `.gitignore`.
- **Do NOT embed** private keys in environment variables where they may be logged by CI/CD systems.
- **File Permissions**: Use `chmod 600 atb-key.pem` to ensure only the owner can read the private key. `atb keygen` sets these permissions automatically.
- **Secret Management**: In production environments, use a dedicated manager (e.g., AWS Secrets Manager, HashiCorp Vault) to inject key files at runtime.

## Key rotation

ATB supports signature verification using the public key embedded within the signature record itself.

- **Backward Compatibility**: Bundles signed with a rotated (old) key remain verifiable because the old public key is stored alongside the signature in the `atb.bundle.signature` or `ai.policy.decision` record.
- **Supercession**: When `atb sign` is run again with a new key, a new signature record is appended to the chain. `atb verify` evaluates the **latest** bundle signature record in the chain.
- **Multi-key Support**: A bundle can contain policy decisions signed by different keys over time; `atb verify` validates each decision record individually.

## Threat model boundary

Ed25519 signing proves that the bundle (or policy event) was signed by a holder of the private key. It does not prove the key holder is authorized to perform the action, nor does it prevent a key holder from signing a modified or incomplete bundle. ATB does not manage key authorization; the mapping of public keys to trusted identities is the responsibility of the auditor. See `docs/security.md` for the full security model.

## Encryption (AES-256-GCM)

ATB provides opt-in, client-side encryption for secure bundle handoff using the `atb encrypt` and `atb decrypt` commands.

```bash
ATB_PASSWORD=secret123 atb encrypt run.atb/bundle.atb --output handoff/bundle.atb.enc
ATB_PASSWORD=secret123 atb decrypt handoff/bundle.atb.enc --output ./bundle.atb
```

Encryption uses AES-256-GCM with PBKDF2-SHA256 key derivation (100,000 iterations). You can provide the password via the `--password` flag or the `ATB_PASSWORD` environment variable. Encryption is separate from signing; for maximum assurance, sign a bundle before encrypting it.
