import {
  createHash,
  createCipheriv,
  createDecipheriv,
  pbkdf2Sync,
  randomBytes,
} from "node:crypto";
import { canonicalize } from "./canonicalize.js";
import { parseEvent, prepareForCanonical, type Event } from "./event.js";
import { GENESIS_HASH } from "./hash.js";
import type { ATBRecord } from "./types.js";

export const MAGIC = "ATBE";
export const LEGACY_VERSION = 0x01;
export const VERSION = 0x02;
export const SALT_SIZE = 16;
export const NONCE_SIZE = 12;
export const TAG_SIZE = 16;
export const KEY_SIZE = 32;
export const LEGACY_PBKDF2_ITERATIONS = 100_000;
export const PBKDF2_ITERATIONS = 600_000;
export const HEADER_SIZE =
  MAGIC.length + 1 + SALT_SIZE + NONCE_SIZE + TAG_SIZE;

/** Error raised by encryption and decryption helpers. */
export class EncryptError extends Error {
  /**
   * @param message Human-readable encryption failure message.
   * @returns A new encryption error.
   */
  constructor(message: string) {
    super(message);
    this.name = "EncryptError";
  }
}

/** Optional deterministic encryption inputs for tests and golden vectors. */
export interface EncryptOptions {
  salt?: Uint8Array;
  nonce?: Uint8Array;
}

interface WirePayload {
  head_hash: string;
  records: ATBRecord[];
}

/** Decrypted and verified bundle payload. */
export interface DecryptedBundlePayload {
  headHash: string;
  records: ATBRecord[];
}

function deriveKey(
  password: string,
  salt: Uint8Array,
  iterations: number
): Buffer {
  if (!password) {
    throw new EncryptError("password cannot be empty");
  }
  if (salt.length !== SALT_SIZE) {
    throw new EncryptError(`salt must be ${SALT_SIZE} bytes`);
  }
  return pbkdf2Sync(password, salt, iterations, KEY_SIZE, "sha256");
}

function kdfIterationsForVersion(version: number): number {
  switch (version) {
    case LEGACY_VERSION:
      return LEGACY_PBKDF2_ITERATIONS;
    case VERSION:
      return PBKDF2_ITERATIONS;
    default:
      throw new EncryptError(
        `unsupported version: 0x${version.toString(16).padStart(2, "0")}`
      );
  }
}

function computeEventHash(event: Event, prevHash: string): string {
  return createHash("sha256")
    .update(prevHash, "utf8")
    .update(canonicalize(prepareForCanonical(event)), "utf8")
    .digest("hex");
}

/**
 * @param plaintext Plain bytes to encrypt.
 * @param password Password used for PBKDF2 key derivation.
 * @param options Optional deterministic salt and nonce.
 * @returns ATBE wire bytes.
 * @throws EncryptError when password, salt, nonce, or AES-GCM output is invalid.
 */
export function encryptRaw(
  plaintext: Uint8Array,
  password: string,
  options: EncryptOptions = {}
): Uint8Array {
  return encryptRawWithVersion(plaintext, password, VERSION, options);
}

function encryptRawWithVersion(
  plaintext: Uint8Array,
  password: string,
  version: number,
  options: EncryptOptions = {}
): Uint8Array {
  const salt = options.salt ? Buffer.from(options.salt) : randomBytes(SALT_SIZE);
  const nonce = options.nonce
    ? Buffer.from(options.nonce)
    : randomBytes(NONCE_SIZE);
  if (nonce.length !== NONCE_SIZE) {
    throw new EncryptError(`nonce must be ${NONCE_SIZE} bytes`);
  }

  const key = deriveKey(password, salt, kdfIterationsForVersion(version));
  const cipher = createCipheriv("aes-256-gcm", key, nonce);
  const ciphertext = Buffer.concat([
    cipher.update(Buffer.from(plaintext)),
    cipher.final(),
  ]);
  const tag = cipher.getAuthTag();
  if (tag.length !== TAG_SIZE) {
    throw new EncryptError(`auth tag must be ${TAG_SIZE} bytes`);
  }

  return Buffer.concat([
    Buffer.from(MAGIC, "ascii"),
    Buffer.from([version]),
    salt,
    nonce,
    tag,
    ciphertext,
  ]);
}

/**
 * @param data ATBE wire bytes.
 * @param password Password used for PBKDF2 key derivation.
 * @returns Decrypted plaintext bytes.
 * @throws EncryptError when format validation or authentication fails.
 */
export function decryptRaw(data: Uint8Array, password: string): Uint8Array {
  const bytes = Buffer.from(data);
  if (bytes.length < HEADER_SIZE) {
    throw new EncryptError("invalid format");
  }
  if (bytes.subarray(0, MAGIC.length).toString("ascii") !== MAGIC) {
    throw new EncryptError("invalid format");
  }
  const version = bytes[MAGIC.length];
  if (!password) {
    throw new EncryptError("password cannot be empty");
  }
  const iterations = kdfIterationsForVersion(version);

  let offset = MAGIC.length + 1;
  const salt = bytes.subarray(offset, offset + SALT_SIZE);
  offset += SALT_SIZE;
  const nonce = bytes.subarray(offset, offset + NONCE_SIZE);
  offset += NONCE_SIZE;
  const tag = bytes.subarray(offset, offset + TAG_SIZE);
  offset += TAG_SIZE;
  const ciphertext = bytes.subarray(offset);

  const key = deriveKey(password, salt, iterations);
  const decipher = createDecipheriv("aes-256-gcm", key, nonce);
  decipher.setAuthTag(tag);
  try {
    return Buffer.concat([decipher.update(ciphertext), decipher.final()]);
  } catch {
    throw new EncryptError("authentication failed");
  }
}

function toWirePayload(records: readonly ATBRecord[]): WirePayload {
  const headHash =
    records.length > 0
      ? records[records.length - 1].hash
      : GENESIS_HASH;
  return {
    head_hash: headHash,
    records: records.map((record) => ({
      event: parseEvent(record.event),
      hash: record.hash,
    })),
  };
}

function fromWirePayload(payload: unknown): DecryptedBundlePayload {
  if (!payload || typeof payload !== "object") {
    throw new EncryptError("decrypted payload must be an object");
  }
  const maybe = payload as Record<string, unknown>;
  const headHash = maybe.head_hash;
  const recordsValue = maybe.records;
  if (typeof headHash !== "string" || !Array.isArray(recordsValue)) {
    throw new EncryptError("decrypted payload missing required fields");
  }

  const parsedRecords: ATBRecord[] = recordsValue.map((item) => {
    if (!item || typeof item !== "object") {
      throw new EncryptError("record must be an object");
    }
    const record = item as Record<string, unknown>;
    if (typeof record.hash !== "string" || !record.event || typeof record.event !== "object") {
      throw new EncryptError("record must include event object and hash string");
    }
    let event: Event;
    try {
      event = parseEvent(record.event);
    } catch {
      throw new EncryptError("record event must include seq, prev_hash, and type");
    }
    return {
      event,
      hash: record.hash,
    };
  });

  const hasManifest =
    parsedRecords.length > 0 &&
    parsedRecords[0].event.type === "atb.bundle.manifest";
  let prev = GENESIS_HASH;
  for (let i = 0; i < parsedRecords.length; i++) {
    const record = parsedRecords[i];
    const expectedSeq = hasManifest ? (i === 0 ? 0 : i) : i + 1;
    if (record.event.seq !== expectedSeq || record.event.prev_hash !== prev) {
      throw new EncryptError("decrypted payload failed hash-chain verification");
    }
    const computed = computeEventHash(record.event, prev);
    if (computed !== record.hash) {
      throw new EncryptError("decrypted payload failed hash-chain verification");
    }
    prev = computed;
  }
  if (headHash !== prev) {
    throw new EncryptError(
      `decrypted payload head_hash mismatch: expected ${headHash}, got ${prev}`
    );
  }

  return {
    headHash,
    records: parsedRecords,
  };
}

/**
 * @param records Bundle records to verify and encrypt.
 * @param password Password used for PBKDF2 key derivation.
 * @param options Optional deterministic salt and nonce.
 * @returns Promise resolving to ATBE wire bytes.
 * @throws EncryptError when payload verification or encryption fails.
 */
export async function encryptBundle(
  records: readonly ATBRecord[],
  password: string,
  options: EncryptOptions = {}
): Promise<Uint8Array> {
  const payload = toWirePayload(records);
  const canonical = canonicalize(payload);
  return encryptRaw(Buffer.from(canonical, "utf8"), password, options);
}

/**
 * @param password Password used for PBKDF2 key derivation.
 * @param data ATBE wire bytes.
 * @returns Promise resolving to a verified decrypted payload.
 * @throws EncryptError when decryption, JSON decoding, or verification fails.
 */
export async function decryptBundle(
  password: string,
  data: Uint8Array
): Promise<DecryptedBundlePayload> {
  const plaintext = decryptRaw(data, password);
  let parsed: unknown;
  try {
    parsed = JSON.parse(Buffer.from(plaintext).toString("utf8")) as unknown;
  } catch {
    throw new EncryptError("decrypted payload is not valid JSON");
  }
  return fromWirePayload(parsed);
}
