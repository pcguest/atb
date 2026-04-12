# Key management

This document describes the lifecycle of cryptographic keys used by ATB
for bundle signing, policy-event signing, and optional encrypted bundle
handoff.

## Overview

In ATB, signing provides tamper evidence. It proves that a bundle, or a
specific policy event, was produced by a holder of a private key and was
not modified afterwards. It is not an access-control mechanism.

ATB v1.4.0 supports:

- Ed25519 bundle signing for whole-bundle state attestation.
- Policy event signing for `ai.policy.decision` events.
- AES-256-GCM encryption for passphrase-based bundle handoff or storage.

Out of scope for v1.4.0: server-side key custody, multi-tenant key
management, and hardware security module integration.

## Generating a keypair

Use `atb keygen` to generate an Ed25519 signing keypair.

```bash
atb keygen --out-dir ./keys
```

By default, this writes two PEM-encoded files to the specified directory
or the current directory if omitted:

- `atb-key.pem`: the private key. Keep this secret.
- `atb-key.pub.pem`: the public key.

The private key is written with restricted file permissions, `0600`.

## Signing a bundle

Bundle signing captures a SHA-256 digest of the current bundle state and
appends a signed record. `atb bundle new`, or `atb init`, does not
generate a keypair automatically.

```bash
atb sign --bundle run.atb/bundle.atb --key ./keys/atb-key.pem
```

The signature record, `atb.bundle.signature`, includes the bundle hash,
the signature, and the public key used for verification.

## Signing a policy event

Use `--sign-policy` on `ai.policy.decision` events when you need
cryptographic proof of policy-decision origin.

```bash
atb append ai.policy.decision \
  --data '{"policy_id":"p-1","decision":"allow","action_id":"act-123"}' \
  --sign-policy ./keys/atb-key.pem
```

This adds `policy_signature` and `policy_signer_pubkey` to the event
payload. ATB verifies those fields during profile evaluation.

## Verification behaviour

`atb verify` and `atb trust-report` surface signature status during
validation.

- Bundle signatures:
  `atb verify` reports whether a signature record is present and whether
  it verifies. In JSON output, `bundle_signature` is omitted when no
  signature record is found.
- Policy signatures:
  valid signatures are reported as informational notes; absent signatures
  are warnings; invalid signatures are critical failures.

## Key storage recommendations

ATB is local-first and does not provide a built-in secrets vault.

- Do not commit `.pem` files to Git.
- Do not embed private keys in environment variables that may be logged by CI systems.
- Use `chmod 600 atb-key.pem` if you create or move key files outside `atb keygen`.
- Use an external secret manager in production when keys must be injected at runtime.

## Key rotation

Rotate Ed25519 signing keys by generating a new keypair, switching new
signing operations to the new private key, and retaining the old public
key for historical verification.

1. Generate a new Ed25519 keypair.

   ```bash
   atb keygen --out-dir ./keys-2026-04
   ```

2. Keep the previous verification material during the transition.
- Bundles already signed with the old key remain valid.
- Historical `ai.policy.decision` records signed with the old key remain valid.
- Retain the old public key for as long as any historical bundle or policy record signed with it may need verification.

3. Update the active signing key in the surrounding operational workflow.
- ATB v1.4.0 does not store an active signing key in `./.atb/config.json`.
- Future bundle signing must use the new key path with `atb sign --bundle <path> --key <new-key.pem>`.
- Future policy signing must use the new key path with `atb append ai.policy.decision ... --sign-policy <new-key.pem>`.
- If wrapper scripts, CI jobs, or secret injection layers choose the key path, update that operational configuration as well.

4. Re-sign or re-anchor only when required.
- Historical bundles do not need re-signing solely because a key rotated.
- If a current bundle must carry the new signing key, run `atb sign` again with the new private key. This appends a new `atb.bundle.signature` record, and `atb verify` evaluates the latest bundle signature record in the chain.
- If external time-bounding evidence is required for the newly signed bundle state, run `atb anchor [bundle_path] [--tsa-url <url>]` again after re-signing.

ATB stores the signing public key inside the bundle evidence, so bundles
remain verifiable after rotation. Key custody, distribution, and the
mapping of keys to trusted organisational identities remain the
responsibility of the operating organisation.

## Threat model boundary

Ed25519 signing proves that a bundle, or policy event, was signed by a
holder of the private key. It does not prove that the key holder was
authorised to perform the action, and it does not prevent a key holder
from signing an incomplete or misleading bundle.

## Encryption

ATB provides opt-in, client-side encryption for bundle handoff with
`atb encrypt` and `atb decrypt`.

```bash
ATB_PASSWORD=secret123 atb encrypt run.atb/bundle.atb --output handoff/bundle.atb.enc
ATB_PASSWORD=secret123 atb decrypt handoff/bundle.atb.enc --output ./bundle.atb
```

Encryption uses AES-256-GCM with versioned PBKDF2-SHA256 key derivation:

- New encrypted bundles use wire version `0x02` with `600000` iterations.
- Legacy `0x01` encrypted bundles remain decryptable with `100000` iterations.

The password can be provided with `--password` or `ATB_PASSWORD`.
Encryption is separate from signing. For maximum assurance, sign a
bundle before encrypting it.
