# atb push Validation Checklist

Do not start implementation unless the validation gate below is met.

## Validation Gate

- [ ] The feature is justified by a concrete secure-handoff workflow after local bundles are already useful
- [ ] The target workflows are customer handoff, incident review, or privacy review rather than generic observability
- [ ] Project docs can describe `atb push` in one sentence as encrypted handoff, not hosted tracing
- [ ] The proposed flow preserves ATB's local-first default when unused

## If Validation Passes

- [ ] Finalise the narrowest command shape and expiry policy
- [ ] Finalise KDF choice and local performance budget
- [ ] Choose a minimal ciphertext-only handoff backend
- [ ] Add `atb push <bundle>` command
- [ ] Add retrieval/decrypt flow only if required for recipient success
- [ ] Generate time-bounded retrieval links

## Security

- [ ] Secret material never sent to the server in plaintext
- [ ] Stored artefacts remain ciphertext-only
- [ ] Integrity is verified before transfer and after retrieval
- [ ] Abuse guardrails exist on any retrieval endpoint

## Testing

- [ ] Unit tests for encrypt/decrypt and KDF
- [ ] Integration test: verify -> push -> retrieve -> decrypt -> verify
- [ ] Negative tests: wrong secret, expired link, tampered ciphertext
- [ ] Security review checklist completed

## Documentation

- [ ] Update README only after the command ships
- [ ] Document the feature as optional encrypted handoff, not cloud tracing
- [ ] Keep quickstart focused on the local workflow unless usage proves otherwise

## Explicit Non-Goals

- [ ] No hosted workspace or shared dashboard scope
- [ ] No admin console, billing, or seat-management work
- [ ] No prompt tooling, evals, or generic observability expansion
