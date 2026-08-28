/**
 * ATB Bundle — the primary interface for creating and managing ATB trace bundles.
 */

import { readFileSync, writeFileSync, mkdirSync, existsSync } from "node:fs";
import {
  createHash,
  createPrivateKey,
  createPublicKey,
  randomBytes,
  sign as signBytes,
  verify as verifyBytes,
  type KeyObject,
} from "node:crypto";
import { dirname } from "node:path";
import { derToP1363 } from "./asn1.js";
import { decryptBundle, encryptBundle } from "./encrypt.js";
import {
  normalizeOptionalIdentity,
  parseEvent,
  type AppendIdentityOptions,
  type Event,
} from "./event.js";
import { GENESIS_HASH, computeHash } from "./hash.js";
import type { ATBRecord, BundleOptions } from "./types.js";

const DEFAULT_PATH = "run.atb/bundle.atb";
const MANIFEST_EVENT_TYPE = "atb.bundle.manifest";
const SIGNATURE_EVENT_TYPE = "atb.bundle.signature";
const MANIFEST_VERSION = "1";
export const MAX_LINE_SIZE_BYTES = 16 * 1024 * 1024;
export const MAX_BUNDLE_SIZE_BYTES = 512 * 1024 * 1024;
export const MAX_BUNDLE_RECORDS = 1_000_000;

/** Resource bounds applied while loading an untrusted bundle. */
export interface BundleLoadLimits {
  maxBytes?: number;
  maxRecords?: number;
}

/** Raised when an input exceeds a configured safe-loading bound. */
export class BundleResourceLimitError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "BundleResourceLimitError";
  }
}

/** Verification evidence for one `atb.bundle.signature` record. */
export interface SignatureEvidence {
  sequence: number;
  backend: string;
  key_id: string;
  signed_at: string;
  pubkey: string;
  bundle_hash: string;
  valid: boolean;
  /**
   * Signing algorithm declared by the signature record. Defaults to
   * `"ed25519"` when the record predates the algorithm field. KMS-signed
   * bundles report `"ecdsa-p256"`.
   */
  algorithm: "ed25519" | "ecdsa-p256" | string;
}

/** Result returned by {@link Bundle.verify}. */
export interface VerifyResult {
  signatures: SignatureEvidence[];
}

function normalizeOptionalField(
  value: string | undefined
): string | undefined {
  if (value === undefined) {
    return undefined;
  }
  const trimmed = value.trim();
  if (trimmed === "") {
    return undefined;
  }
  return trimmed;
}

function withDerivedChain(event: Event, seq: number, prevHash: string): Event {
  const out: Event = {
    seq,
    prev_hash: prevHash,
    type: event.type,
    data: event.data,
  };
  if (event.hash_algo !== undefined) {
    out.hash_algo = event.hash_algo;
  }
  if (event.actor_id !== undefined) {
    out.actor_id = event.actor_id;
  }
  if (event.org_id !== undefined) {
    out.org_id = event.org_id;
  }
  if (event.workspace_id !== undefined) {
    out.workspace_id = event.workspace_id;
  }
  if (event.timestamp !== undefined) {
    out.timestamp = event.timestamp;
  }
  if (event.trace_id !== undefined) {
    out.trace_id = event.trace_id;
  }
  if (event.span_id !== undefined) {
    out.span_id = event.span_id;
  }
  if (event.parent_span_id !== undefined) {
    out.parent_span_id = event.parent_span_id;
  }
  return out;
}

export class ATBVerificationError extends Error {
  /**
   * @param message Human-readable verification failure.
   * @param eventIndex Zero-based record index that failed verification.
   * @param expectedHash Hash stored in the bundle.
   * @param computedHash Hash recomputed by the verifier.
   * @returns A verification error.
   */
  constructor(
    message: string,
    public readonly eventIndex: number,
    public readonly expectedHash: string,
    public readonly computedHash: string
  ) {
    super(message);
    this.name = "ATBVerificationError";
  }
}

/**
 * An in-memory ATB bundle.
 *
 * @example
 * ```ts
 * import {
 *   AI_REQUEST_RECEIVED_EVENT_TYPE,
 *   Bundle,
 * } from "@pcguest/atb-sdk";
 *
 * const bundle = new Bundle();
 * bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, { request_id: "req-001" });
 * bundle.save();
 *
 * const loaded = Bundle.load();
 * loaded.verify();
 * ```
 */
export class Bundle {
  readonly records: ATBRecord[] = [];
  private readonly path: string;
  private rawBytes?: Buffer;

  /**
   * @param options Bundle construction options.
   * @returns A new in-memory bundle with a manifest record.
   */
  constructor(options: BundleOptions = {}) {
    this.path = options.path ?? DEFAULT_PATH;
    this.records.push(this.createManifestRecord());
  }

  /**
   * Append a new event to the bundle.
   *
   * @param type Dot-namespaced event type.
   * @param data JSON-like event payload.
   * @param options Optional identity, timestamp, and trace metadata.
   * @returns The appended bundle record.
   * @throws TypeError when the event cannot be canonicalised.
   */
  append(
    type: string,
    data: unknown,
    options: AppendIdentityOptions = {}
  ): ATBRecord {
    const prevHash =
      this.records.length > 0
        ? this.records[this.records.length - 1].hash
        : GENESIS_HASH;

    const event: Event = {
      seq: this.hasManifestRecord() ? this.records.length : this.records.length + 1,
      prev_hash: prevHash,
      type,
      data,
      hash_algo: "sha256",
    };
    const actorID = normalizeOptionalIdentity(options.actorId);
    const orgID = normalizeOptionalIdentity(options.orgId);
    const workspaceID = normalizeOptionalIdentity(options.workspaceId);
    if (actorID !== undefined) {
      event.actor_id = actorID;
    }
    if (orgID !== undefined) {
      event.org_id = orgID;
    }
    if (workspaceID !== undefined) {
      event.workspace_id = workspaceID;
    }
    const timestamp = normalizeOptionalField(options.timestamp);
    const traceID = normalizeOptionalField(options.traceId);
    const spanID = normalizeOptionalField(options.spanId);
    const parentSpanID = normalizeOptionalField(options.parentSpanId);
    if (timestamp !== undefined) {
      event.timestamp = timestamp;
    }
    if (traceID !== undefined) {
      event.trace_id = traceID;
    }
    if (spanID !== undefined) {
      event.span_id = spanID;
    }
    if (parentSpanID !== undefined) {
      event.parent_span_id = parentSpanID;
    }

    const hash = computeHash(event, prevHash);
    const record: ATBRecord = { event, hash };
    this.records.push(record);
    return record;
  }

  /**
   * Verify the integrity of the entire bundle.
   *
   * @returns Signature evidence gathered while verifying the chain.
   * @throws ATBVerificationError when a record hash does not match.
   */
  verify(): VerifyResult {
    let prev = GENESIS_HASH;
    const hasManifest = this.hasManifestRecord();
    for (let i = 0; i < this.records.length; i++) {
      const record = this.records[i];
      const seq = hasManifest ? (i === 0 ? 0 : i) : i + 1;
      const event = withDerivedChain(record.event, seq, prev);
      const computed = computeHash(event, prev);
      if (computed !== record.hash) {
        throw new ATBVerificationError(
          `Tamper detected at event ${i + 1}: expected ${record.hash}, computed ${computed}`,
          i,
          record.hash,
          computed
        );
      }
      prev = computed;
    }
    return { signatures: this.signatures() };
  }

  /**
   * Return per-signature provenance and cryptographic verification results.
   *
   * @param path Optional bundle path to use as the raw signature source.
   * @returns Signature evidence entries in bundle order.
   * @throws Error when a signature record declares an unsupported algorithm.
   */
  signatures(path?: string): SignatureEvidence[] {
    const raw = this.signatureSourceBytes(path);
    const out: SignatureEvidence[] = [];
    for (let i = 0; i < this.records.length; i++) {
      const record = this.records[i];
      if (record.event.type !== SIGNATURE_EVENT_TYPE) {
        continue;
      }
      const data = signatureData(record.event.data);
      if (data === undefined) {
        continue;
      }
      const pubkey = stringField(data, "pubkey") || stringField(data, "public_key");
      out.push({
        sequence: record.event.seq,
        backend: stringField(data, "backend") || "local",
        key_id: stringField(data, "key_id"),
        signed_at: stringField(data, "signed_at"),
        pubkey,
        bundle_hash: stringField(data, "bundle_hash"),
        algorithm: stringField(data, "algorithm") || "ed25519",
        valid: verifySignatureRecord(data, raw, i),
      });
    }
    return out;
  }

  /**
   * Async sibling of `signatures()` that verifies every signature record
   * — including ECDSA-P256 — using WebCrypto's SubtleCrypto. Use this when
   * working with KMS-signed bundles; the synchronous `signatures()` only
   * verifies Ed25519 records and reports `valid: false` for any other
   * algorithm.
   *
   * @param path Optional bundle path to use as the raw signature source.
   * @returns Promise resolving to signature evidence entries in bundle order.
   * @throws Error when signature parsing or verification setup fails.
   */
  async signaturesAsync(path?: string): Promise<SignatureEvidence[]> {
    const raw = this.signatureSourceBytes(path);
    const out: SignatureEvidence[] = [];
    for (let i = 0; i < this.records.length; i++) {
      const record = this.records[i];
      if (record.event.type !== SIGNATURE_EVENT_TYPE) {
        continue;
      }
      const data = signatureData(record.event.data);
      if (data === undefined) {
        continue;
      }
      const pubkey = stringField(data, "pubkey") || stringField(data, "public_key");
      const algorithm = stringField(data, "algorithm") || "ed25519";
      out.push({
        sequence: record.event.seq,
        backend: stringField(data, "backend") || "local",
        key_id: stringField(data, "key_id"),
        signed_at: stringField(data, "signed_at"),
        pubkey,
        bundle_hash: stringField(data, "bundle_hash"),
        algorithm,
        valid: await verifySignatureRecordAsync(data, raw, i),
      });
    }
    return out;
  }

  /**
   * Append a local Ed25519 bundle signature record to a bundle file.
   *
   * @param privateKeyPem PEM string or bytes for an Ed25519 private key.
   * @param path Optional bundle path. Defaults to this bundle's path.
   * @param options Optional signature metadata.
   * @returns The appended signature record.
   * @throws Error when the key or bundle cannot be read or signed.
   */
  sign_local(
    privateKeyPem: string | Uint8Array,
    path?: string,
    options: { key_id?: string } = {}
  ): ATBRecord {
    const target = path ?? this.path;
    if (!existsSync(target)) {
      this.save(target);
    }

    const raw = readFileSync(target);
    const digest = createHash("sha256").update(raw).digest();
    const privateKey = loadPrivateKey(privateKeyPem);
    const signature = signBytes(null, digest, privateKey);
    const pubkey = rawEd25519PublicKey(privateKey);
    const signedAt = new Date().toISOString();
    const record = this.append(
      SIGNATURE_EVENT_TYPE,
      {
        bundle_hash: digest.toString("hex"),
        signature: signature.toString("base64"),
        pubkey: pubkey.toString("base64"),
        key_id: options.key_id ?? "",
        backend: "local",
        signed_at: signedAt,
      },
      { timestamp: signedAt }
    );

    const encodedRecord = Buffer.from(JSON.stringify(record), "utf8");
    const newline = raw.length > 0 && raw[raw.length - 1] !== 0x0a ? "\n" : "";
    const payload = Buffer.concat([
      raw,
      Buffer.from(newline, "utf8"),
      encodedRecord,
      Buffer.from("\n", "utf8"),
    ]);
    mkdirSync(dirname(target), { recursive: true });
    writeFileSync(target, payload);
    this.rawBytes = payload;
    return record;
  }

  /**
   * Alias for callers that prefer TypeScript naming conventions.
   *
   * @param privateKeyPem PEM string or bytes for an Ed25519 private key.
   * @param path Optional bundle path. Defaults to this bundle's path.
   * @param options Optional signature metadata.
   * @returns The appended signature record.
   * @throws Error when the key or bundle cannot be read or signed.
   */
  signLocal(
    privateKeyPem: string | Uint8Array,
    path?: string,
    options: { keyId?: string } = {}
  ): ATBRecord {
    return this.sign_local(privateKeyPem, path, { key_id: options.keyId });
  }

  /**
   * Save the bundle to disk in NDJSON format.
   *
   * @param path Optional output path. Defaults to this bundle's path.
   * @returns Nothing.
   * @throws Error when the target cannot be written.
   */
  save(path?: string): void {
    const target = path ?? this.path;
    mkdirSync(dirname(target), { recursive: true });
    const lines = this.records.map((r) => JSON.stringify(r));
    const payload = lines.join("\n") + "\n";
    writeFileSync(target, payload, "utf8");
    this.rawBytes = Buffer.from(payload, "utf8");
  }

  /**
   * Load a bundle from disk.
   *
   * @param path Optional bundle path. Defaults to `run.atb/bundle.atb`.
   * @returns Loaded bundle instance.
   * @throws Error when the file cannot be read or JSON parsing fails.
   * @throws TypeError when a bundle record is malformed.
   *
   * @example
   * ```ts
   * import { Bundle } from "@pcguest/atb-sdk";
   *
   * const bundle = Bundle.load("run.atb/bundle.atb");
   * bundle.verify();
   * for (const record of bundle.records) console.log(record.event.type);
   * ```
   */
  static load(path?: string, limits: BundleLoadLimits = {}): Bundle {
    const target = path ?? DEFAULT_PATH;
    const maxBytes = limits.maxBytes ?? MAX_BUNDLE_SIZE_BYTES;
    const maxRecords = limits.maxRecords ?? MAX_BUNDLE_RECORDS;
    if (maxBytes <= 0 || maxRecords <= 0) {
      throw new RangeError("bundle load limits must be positive");
    }
    const bundle = new Bundle({ path: target });
    bundle.records.length = 0;
    const raw = readFileSync(target);
    if (raw.byteLength > maxBytes) {
      throw new BundleResourceLimitError(
        `bundle exceeds maximum size of ${maxBytes} bytes`
      );
    }
    const content = raw.toString("utf8");
    const lines = content.split("\n");
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (Buffer.byteLength(line, "utf8") > MAX_LINE_SIZE_BYTES) {
        throw new BundleResourceLimitError(
          `bundle record at line ${i + 1} exceeds ${MAX_LINE_SIZE_BYTES} bytes`
        );
      }
      const trimmed = line.trim();
      if (!trimmed) continue;
      if (bundle.records.length >= maxRecords) {
        throw new BundleResourceLimitError(
          `bundle exceeds maximum record count of ${maxRecords}`
        );
      }
      const raw = JSON.parse(trimmed) as Record<string, unknown>;
      if (
        !raw ||
        typeof raw !== "object" ||
        typeof raw.hash !== "string" ||
        raw.event === undefined
      ) {
        throw new TypeError(`invalid record at line ${i + 1}`);
      }
      bundle.records.push({
        event: parseEvent(raw.event),
        hash: raw.hash,
      });
    }
    bundle.rawBytes = raw;
    return bundle;
  }

  /**
   * Encrypt this bundle to ATBE bytes.
   *
   * @param password Password used for key derivation.
   * @returns Encrypted ATBE bytes.
   * @throws ATBVerificationError when the bundle fails verification first.
   * @throws EncryptError when encryption fails.
   */
  async encrypt(password: string): Promise<Uint8Array> {
    this.verify();
    return encryptBundle(this.records, password);
  }

  /**
   * Decrypt ATBE bytes into a verified bundle.
   *
   * @param password Password used for key derivation.
   * @param data ATBE wire bytes.
   * @returns Verified bundle instance.
   * @throws EncryptError when decryption or payload verification fails.
   * @throws ATBVerificationError when the resulting bundle chain fails.
   */
  static async decrypt(password: string, data: Uint8Array): Promise<Bundle> {
    const decoded = await decryptBundle(password, data);
    const bundle = new Bundle();
    bundle.records.length = 0;
    bundle.records.push(...decoded.records);
    bundle.verify();
    return bundle;
  }

  /** @returns Number of records currently held by the bundle. */
  get length(): number {
    return this.records.length;
  }

  private hasManifestRecord(): boolean {
    return (
      this.records.length > 0 &&
      this.records[0].event.type === MANIFEST_EVENT_TYPE
    );
  }

  private createManifestRecord(): ATBRecord {
    const createdAt = nowRFC3339();
    const event: Event = {
      seq: 0,
      prev_hash: GENESIS_HASH,
      type: MANIFEST_EVENT_TYPE,
      hash_algo: "sha256",
      // Manifest v1 stores structured fields as a JSON-encoded string. This
      // double-encoding is intentional for hash-chain compatibility.
      data: JSON.stringify({
        version: MANIFEST_VERSION,
        created_at: createdAt,
        bundle_id: randomBytes(16).toString("hex"),
      }),
      timestamp: createdAt,
    };
    return { event, hash: computeHash(event, GENESIS_HASH) };
  }

  private signatureSourceBytes(path?: string): Buffer {
    if (path !== undefined) {
      return readFileSync(path);
    }
    if (this.rawBytes !== undefined) {
      return this.rawBytes;
    }
    return Buffer.from(
      this.records.map((record) => JSON.stringify(record)).join("\n") + "\n",
      "utf8"
    );
  }
}

function nowRFC3339(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}

function loadPrivateKey(privateKeyPem: string | Buffer | Uint8Array): KeyObject {
  let key: string | Buffer;
  if (typeof privateKeyPem === "string" && !privateKeyPem.includes("-----BEGIN")) {
    key = readFileSync(privateKeyPem);
  } else {
    key = Buffer.isBuffer(privateKeyPem)
      ? privateKeyPem
      : Buffer.from(privateKeyPem);
  }
  return createPrivateKey(key);
}

function rawEd25519PublicKey(privateKey: KeyObject): Buffer {
  const publicKey = createPublicKey(privateKey);
  const jwk = publicKey.export({ format: "jwk" }) as { x?: unknown };
  if (typeof jwk.x !== "string") {
    throw new TypeError("Ed25519 public key export missing x coordinate");
  }
  return Buffer.from(jwk.x, "base64url");
}

function publicKeyFromRaw(raw: Buffer): KeyObject {
  return createPublicKey({
    key: {
      kty: "OKP",
      crv: "Ed25519",
      x: raw.toString("base64url"),
    },
    format: "jwk",
  });
}

function verifySignatureRecord(
  data: Record<string, unknown>,
  raw: Buffer,
  index: number
): boolean {
  try {
    const algorithm = stringField(data, "algorithm") || "ed25519";
    if (algorithm !== "ed25519") {
      // Non-Ed25519 algorithms require WebCrypto's async API. Sync callers
      // should fall through to signaturesAsync(); we mark these invalid
      // here rather than throwing to preserve the existing sync contract.
      return false;
    }
    const bundleHash = Buffer.from(stringField(data, "bundle_hash"), "hex");
    const signature = Buffer.from(stringField(data, "signature"), "base64");
    const pubkey = Buffer.from(
      stringField(data, "pubkey") || stringField(data, "public_key"),
      "base64"
    );
    const prefix = rawPrefixBeforeRecord(raw, index);
    const expected = createHash("sha256").update(prefix).digest();
    if (!expected.equals(bundleHash)) {
      return false;
    }
    return verifyBytes(null, bundleHash, publicKeyFromRaw(pubkey), signature);
  } catch {
    return false;
  }
}

async function verifySignatureRecordAsync(
  data: Record<string, unknown>,
  raw: Buffer,
  index: number
): Promise<boolean> {
  const algorithm = stringField(data, "algorithm") || "ed25519";
  const bundleHash = Buffer.from(stringField(data, "bundle_hash"), "hex");
  const signature = Buffer.from(stringField(data, "signature"), "base64");
  const pubkey = Buffer.from(
    stringField(data, "pubkey") || stringField(data, "public_key"),
    "base64"
  );
  const prefix = rawPrefixBeforeRecord(raw, index);
  const expected = createHash("sha256").update(prefix).digest();
  if (!expected.equals(bundleHash)) {
    return false;
  }

  switch (algorithm) {
    case "ed25519":
      try {
        return verifyBytes(
          null,
          bundleHash,
          publicKeyFromRaw(pubkey),
          signature
        );
      } catch {
        return false;
      }
    case "ecdsa-p256":
      return verifyEcdsaP256(pubkey, signature, prefix);
    default:
      throw new Error(
        `atb: unsupported signature algorithm "${algorithm}": upgrade @pcguest/atb-sdk to verify this bundle`
      );
  }
}

async function verifyEcdsaP256(
  pubkey: Uint8Array,
  derSignature: Uint8Array,
  message: Uint8Array
): Promise<boolean> {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) {
    throw new Error(
      "atb: WebCrypto SubtleCrypto is not available in this runtime; ECDSA-P256 verification requires Node.js >= 18 or a modern browser"
    );
  }
  const key = await importEcdsaP256PublicKey(subtle, pubkey);

  let p1363: Uint8Array;
  try {
    p1363 = derToP1363(derSignature);
  } catch {
    return false;
  }

  // SubtleCrypto.verify with {name:"ECDSA", hash:"SHA-256"} hashes the
  // `message` argument internally with SHA-256 and verifies the signature
  // against that digest. The KMS contract signs SHA-256(pre-signature
  // NDJSON bytes); passing those bytes as `message` makes SubtleCrypto
  // recompute the same digest, matching what the KMS signed.
  return subtle.verify(
    { name: "ECDSA", hash: "SHA-256" },
    key,
    toArrayBuffer(p1363),
    toArrayBuffer(message)
  );
}

async function importEcdsaP256PublicKey(
  subtle: SubtleCrypto,
  raw: Uint8Array
): Promise<CryptoKey> {
  const buf = toArrayBuffer(raw);
  // Try SPKI (PKIX DER) first — that is the form returned by AWS/GCP KMS
  // and Vault Transit. Fall back to raw uncompressed (0x04 || X || Y).
  try {
    return await subtle.importKey(
      "spki",
      buf,
      { name: "ECDSA", namedCurve: "P-256" },
      false,
      ["verify"]
    );
  } catch {
    // ignore and try raw form
  }
  if (raw.length === 65 && raw[0] === 0x04) {
    return subtle.importKey(
      "raw",
      buf,
      { name: "ECDSA", namedCurve: "P-256" },
      false,
      ["verify"]
    );
  }
  throw new Error(
    "atb: ECDSA-P256 public key is neither PKIX DER (SPKI) nor 65-byte uncompressed point"
  );
}

function toArrayBuffer(view: Uint8Array): ArrayBuffer {
  const out = new ArrayBuffer(view.byteLength);
  new Uint8Array(out).set(view);
  return out;
}

function rawPrefixBeforeRecord(raw: Buffer, targetIndex: number): Buffer {
  let recordIndex = 0;
  let lineStart = 0;
  while (lineStart < raw.length) {
    let lineEnd = raw.indexOf(0x0a, lineStart);
    if (lineEnd === -1) {
      lineEnd = raw.length;
    }
    if (lineEnd > lineStart) {
      if (recordIndex === targetIndex) {
        return raw.subarray(0, lineStart);
      }
      recordIndex++;
    }
    if (lineEnd === raw.length) {
      break;
    }
    lineStart = lineEnd + 1;
  }
  return Buffer.alloc(0);
}

function signatureData(value: unknown): Record<string, unknown> | undefined {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }
  return value as Record<string, unknown>;
}

function stringField(data: Record<string, unknown>, key: string): string {
  const value = data[key];
  return typeof value === "string" ? value : "";
}
