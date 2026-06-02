# Research note — Custos transparency log (Merkle inclusion proofs + signed checkpoints)

Status: **research / design note. Not yet implemented.** This note exists to lock
the trust model *before* any code, because a transparency log built naively
overclaims. It is reference-infrastructure design for the in-repo Custos layer,
not a hosted-product spec.

Sources are listed at the end.

## 1. The gap this closes

Custos today issues an Ed25519 **attestation** over a receipt's custody facts.
The signed message is exactly (`custos/internal/receipt/attest.go:47`):

```
bundle_hash "\n" receipt_id "\n" submitted_at "\n" signed_at
```

That proves one thing: *"the holder of this key signed these four facts."* It is
genuinely useful — a holder can verify it offline against a published key
(`GET /custody/key`) without trusting the store. But it is **silent** on the
properties custody actually needs:

| Attack by a dishonest/compromised operator | Defended by today's attestation? |
|---|---|
| **Suppression** — drop a receipt that was issued, deny it ever existed | ❌ No |
| **Backdating / post-dating** — sign a `submitted_at` that never happened | ❌ No (operator controls the clock and the key) |
| **Reordering** — present receipts in a false sequence | ❌ No |
| **Equivocation / split view** — show history *A* to an auditor and a divergent history *A′* to the submitter | ❌ No |

This is the precise gap between the vision's two halves: ATB makes evidence *hard
to forge* (hash-chained bundles, L1), but Custos does not yet make it *hard to
suppress or backdate*. The north star — "Rekor for AI execution evidence" — is a
**transparency log**, and a transparency log is exactly the structure that closes
the first three rows above, and (with a witness) the fourth.

## 2. What a transparency log adds, precisely

An append-only Merkle tree over the sequence of receipts, plus periodically
**signed checkpoints** (signed tree heads), gives:

- **Inclusion proof** — O(log n) hashes proving a given receipt is a leaf under a
  given checkpoint root. A holder verifies their receipt is *in* the log.
- **Consistency proof** — O(log n) hashes proving checkpoint *B* (size n₂) is a
  strict append-only extension of checkpoint *A* (size n₁ ≤ n₂): nothing already
  logged was modified, deleted, or reordered. This is what makes "append-only"
  *verifiable* rather than asserted.
- **Signed checkpoint** — the operator's commitment `(origin, size, root_hash)`,
  signed. Two parties who hold checkpoints can compare: if the operator ever
  tried to fork history, the two checkpoints are provably inconsistent.

What it still does **not** give on its own: equivocation resistance. A single
operator holding the only signing key can sign *two* perfectly-internally-valid
checkpoints for two divergent trees and show one to each audience. Detecting that
requires someone to **compare checkpoints across audiences** — gossip — or a
**witness** that refuses to cosign a fork (§5). Saying otherwise would be the
overclaim this note exists to prevent.

## 3. Design

### 3.1 Leaf definition

The log commits to **receipts**, not raw bundle bytes (the WORM store already
content-addresses bytes; see below). A leaf is the RFC 6962 leaf hash of the
canonical receipt:

```
leaf_hash = SHA-256( 0x00 || RFC8785(receipt_json_without_proof_fields) )
```

- `0x00` is the RFC 6962 leaf-prefix; interior nodes use `0x01 || left || right`.
  The domain-separation prefixes are load-bearing — they prevent a second-preimage
  attack that conflates a leaf with an interior node. (RFC 6962 §2.1.)
- We canonicalise with RFC 8785 (JCS) — the **same canonicalisation ATB already
  uses** for its event chain — so the leaf is reproducible by any verifier.
- The receipt's own `attestation`/proof fields are excluded from the leaf preimage
  (a leaf cannot commit to a proof of its own inclusion).

### 3.2 Relationship to the existing WORM store and the two hashes

Custos already distinguishes two hashes; the log adds a third commitment without
disturbing them:

| Value | Meaning | Owner |
|---|---|---|
| **content hash** — `sha256(bundle bytes)` | addresses the WORM blob; `receipt_id = sha256-<content hash>` | `internal/ingest/handler.go` |
| **bundle_hash** — ATB chain-head hash | per-bundle integrity anchor (L1) | the bundle itself |
| **leaf_hash / root_hash** *(new)* | the receipt's position in an append-only, externally-verifiable sequence | the log |

The content-addressed WORM store gives **deduplication and byte-integrity**; it
does *not* give ordering or append-only commitment (re-ingesting is idempotent and
unordered). The transparency log is the missing **sequencing + tamper-evidence of
the sequence**. They compose: WORM answers "are these the exact bytes?", the log
answers "and was this receipt logged, in this position, and not suppressed?".

### 3.3 Checkpoint format (C2SP signed-note)

Adopt the C2SP `tlog-checkpoint` note format verbatim, so the checkpoint is
witness-cosignable from day one with **no breaking change** when witnesses are
added later:

```
custos.example/log
42
CsUYapGGPo4dkMgIAUqom/Xajj7h2fB2MPA3j2jxq2I=

— custos.example/log <base64 ed25519 signature>
```

Line 1 = origin (log identity, schema-less URL). Line 2 = tree size (decimal, no
leading zeros). Line 3 = base64 root hash. Blank line, then one or more signature
lines (`— <key-name> <base64 sig>`). Additional signature lines are how **witness
cosignatures** attach (§5) — same note, more signatures.

Signing key: reuse the existing custody `Signer` (Ed25519) for the log's own
signature, OR provision a **separate log key** so "Custos attested receipt X" and
"the log committed to size N" are independent authorities. Recommend a separate
log key — it keeps the receipt-attestation key and the log-checkpoint key
independently rotatable and lets `GET /custody/key` and a new `GET /log/key`
publish them separately.

### 3.4 HTTP surface (additive; mirrors the existing receipt endpoints)

- `GET /checkpoint` → latest signed checkpoint (C2SP note text).
- `GET /receipts/:id/proof` → `{ leaf_index, leaf_hash, inclusion_proof[], checkpoint }`.
- `GET /log/consistency?from=<size>&to=<size>` → consistency proof between two
  checkpoint sizes.
- `GET /log/key` → the log's checkpoint-signing public key (unauthenticated, like
  `/custody/key` — a verification key is not a secret).

Ingestion (`POST /ingest`) gains one step: after the receipt is signed and stored,
append its leaf to the log and **include the leaf index + the current checkpoint**
in the ingest response, so the submitter walks away with a self-contained,
offline-verifiable inclusion proof (this is Rekor's "synchronous" behaviour).

### 3.5 Storage

The Merkle log needs durable, ordered, append-only leaf storage. Two reference
implementations matching the existing store pattern:

- `InMemoryLog` — for tests/dev (mirrors `InMemoryReceiptStore`).
- `FileSystemLog` — append-only leaf journal + a tile/hash cache, written with the
  same temp-fsync-rename discipline as `store_fs.go`. (A full tiled-log à la
  Trillian/`sunlight` is out of scope for the reference layer; an append-only
  journal that recomputes/cache the tree is sufficient at reference scale and is
  honest about not being a planet-scale log.)

## 4. Offline verifier

Ship a verifier (library function + `atb`/`custosd` subcommand) that, given a
receipt + inclusion proof + checkpoint (+ optional witness cosignatures), checks:

1. the receipt's own attestation (existing `VerifyAttestation`);
2. `leaf_hash` recomputes from the canonical receipt;
3. the inclusion proof reconstructs the checkpoint `root_hash`;
4. the checkpoint signature verifies against the published log key;
5. (if present) each witness cosignature verifies against its published key.

Crucially the verifier requires **no trust in the store** and no network for steps
1–4 — the holder pins the log key once and verifies thereafter offline.

## 5. Trust model — what it proves, and what it does NOT (read before building)

This is the section that keeps the feature honest.

**A single-operator log + signed checkpoint proves, to anyone who sees a given
checkpoint:**

- their receipt is included at a fixed position (inclusion proof);
- the log has only ever grown (consistency proof between any two checkpoints they
  hold);
- therefore suppression and silent backdating/reordering are detectable **to a
  party who keeps comparing checkpoints over time**, because the operator cannot
  retroactively alter or drop an entry without breaking consistency against a
  checkpoint someone already holds.

**It does NOT, by itself, prevent equivocation (split-view):** one operator with
one key can present a consistent history *A* to Alice and a *different* consistent
history *A′* to Bob. Each is internally valid; the fraud is only visible if Alice
and Bob compare checkpoints. Defences, in increasing strength:

1. **Publish checkpoints to an append-only public location** (a gossip endpoint, a
   public mirror, a notary, even a Git repo). Makes split-view risky because there
   is one public checkpoint stream to contradict.
2. **Witness cosignatures (recommended target).** A witness fetches each new
   checkpoint, verifies a consistency proof against the last checkpoint it
   cosigned, and only then adds its signature. Because a witness *will not cosign a
   checkpoint inconsistent with one it already cosigned*, the operator cannot get a
   forked checkpoint witnessed. A receipt carrying ≥1 independent witness
   cosignature is equivocation-resistant up to the honesty of *that one* witness;
   N witnesses raise the bar to N colluding parties. This is exactly the model
   Sigstore's Rekor adopted (synchronous witnessing) and is specified vendor-neutrally
   as C2SP `tlog-witness`.

**Therefore the honest public claim, per phase:**

- *Phase 1 (log + signed checkpoint, single operator):* "Custos's custody log is
  append-only and externally verifiable; any party holding a checkpoint can detect
  later suppression, modification, or reordering. A single operator could still
  show divergent views to parties who never compare checkpoints — equivocation
  resistance requires the witnessing in Phase 2."
- *Phase 2 (+ witness cosignatures):* "Receipts carry independent witness
  cosignatures; a forked or backdated log cannot be witnessed, so split-view
  attacks are detectable without the auditor and submitter ever meeting."

This phrasing slots directly into the provability ladder (`docs/provability-ladder.md`)
as a strengthening of **L4 (external witness)**: today L4 is "WORM push + RFC 3161
anchor + corroboration"; the log upgrades "WORM push" from *asserted* custody to a
*verifiable append-only log*, and the witness layer closes the L4 residual
"operator bypasses export controls / shows divergent views."

## 6. Recommended phasing

- **P0 — this note.** Lock the trust model and the leaf/checkpoint formats.
- **P1 — single-operator log.** `InMemoryLog` + `FileSystemLog`, RFC 6962 leaf/node
  hashing, inclusion + consistency proofs, C2SP checkpoint signing with a dedicated
  log key, the four endpoints, synchronous inclusion proof in the ingest response,
  and the offline verifier. Docs ship the Phase-1 trust claim verbatim. Tests:
  inclusion/consistency proof round-trips, tamper-detection (mutated leaf fails),
  empty-tree checkpoint, golden RFC 6962 test vectors.
- **P2 — witnessing.** Implement the C2SP `tlog-witness` add-checkpoint protocol;
  ship a minimal reference witness (`custos witness`) that cosigns only consistent
  checkpoints; attach cosignatures to checkpoints and verify them in the verifier.
  Upgrade the trust claim to Phase 2.
- **P3 — public checkpoint publication / gossip** (optional, lightweight): a
  published checkpoint feed so third parties can monitor, à la `rekor-monitor`.

## 7. Explicit non-goals

- Not a blockchain, not consensus, not a coin. A Merkle log + witnesses is strictly
  simpler and is the established pattern (CT, Go checksum DB, Sigstore).
- Not multi-tenant hosted operation (witness *networks*, billing, SLAs) — that
  stays outside the runtime per `AGENTS.md`. The in-repo layer ships a single log +
  a single reference witness to demonstrate the shape.
- Does **not** change ATB bundle integrity (L1) or the receipt attestation — both
  remain exactly as they are; the log is additive.

## Sources

- [RFC 6962 — Certificate Transparency](https://www.rfc-editor.org/rfc/rfc6962.html) (Merkle tree, leaf/node hashing, inclusion & consistency proofs, STH)
- [RFC 9162 — Certificate Transparency v2.0](https://datatracker.ietf.org/doc/html/rfc9162)
- [C2SP `tlog-checkpoint` — signed checkpoint note format](https://github.com/C2SP/C2SP/blob/main/tlog-checkpoint.md)
- [C2SP `tlog-witness` — witness cosignature protocol](https://github.com/C2SP/C2SP/blob/main/tlog-witness.md)
- [Sigstore Rekor — software supply-chain transparency log](https://docs.sigstore.dev/logging/overview/) and [rekor-tiles clients](https://github.com/sigstore/rekor-tiles/blob/main/CLIENTS.md)
- [transparency.dev — "Can I get a witness (network)?"](https://blog.transparency.dev/can-i-get-a-witness-network) (equivocation / split-view, witness cosigning)
- [Google Trillian — Transparent Logging guide](https://google.github.io/trillian/docs/TransparentLogging.html)
