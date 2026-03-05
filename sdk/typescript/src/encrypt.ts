import {
  createHash,
  createCipheriv,
  createDecipheriv,
  pbkdf2Sync,
  randomBytes,
} from "node:crypto";
import { canonicalize } from "./canonicalize.js";
import { parseEvent, prepareForCanonical, type Event } from "./event.js";
import { computeHash, GENESIS_HASH } from "./hash.js";
import type { ATBRecord } from "./types.js";

export const MAGIC = "ATBE";
export const VERSION = 0x01;
export const SALT_SIZE = 16;
export const NONCE_SIZE = 12;
export const TAG_SIZE = 16;
export const KEY_SIZE = 32;
export const PBKDF2_ITERATIONS = 100_000;
export const HEADER_SIZE =
  MAGIC.length + 1 + SALT_SIZE + NONCE_SIZE + TAG_SIZE;

export class EncryptError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "EncryptError";
  }
}

export interface EncryptOptions {
  salt?: Uint8Array;
  nonce?: Uint8Array;
}

interface WireRecord {
  event: Event;
  hash: string;
}

interface WirePayload {
  head_hash: string;
  records: WireRecord[];
}

export interface DecryptedBundlePayload {
  headHash: string;
  records: ATBRecord[];
}

function deriveKey(password: string, salt: Uint8Array): Buffer {
  if (!password) {
    throw new EncryptError("password cannot be empty");
  }
  if (salt.length !== SALT_SIZE) {
    throw new EncryptError(`salt must be ${SALT_SIZE} bytes`);
  }
  return pbkdf2Sync(password, salt, PBKDF2_ITERATIONS, KEY_SIZE, "sha256");
}

function computeWireHash(event: Event, prevHash: string): string {
  return createHash("sha256")
    .update(prevHash, "utf8")
    .update(canonicalize(prepareForCanonical(event)), "utf8")
    .digest("hex");
}

export function encryptRaw(
  plaintext: Uint8Array,
  password: string,
  options: EncryptOptions = {}
): Uint8Array {
  const salt = options.salt ? Buffer.from(options.salt) : randomBytes(SALT_SIZE);
  const nonce = options.nonce
    ? Buffer.from(options.nonce)
    : randomBytes(NONCE_SIZE);
  if (nonce.length !== NONCE_SIZE) {
    throw new EncryptError(`nonce must be ${NONCE_SIZE} bytes`);
  }

  const key = deriveKey(password, salt);
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
    Buffer.from([VERSION]),
    salt,
    nonce,
    tag,
    ciphertext,
  ]);
}

export function decryptRaw(data: Uint8Array, password: string): Uint8Array {
  const bytes = Buffer.from(data);
  if (bytes.length < HEADER_SIZE) {
    throw new EncryptError("invalid format");
  }
  if (bytes.subarray(0, MAGIC.length).toString("ascii") !== MAGIC) {
    throw new EncryptError("invalid format");
  }
  const version = bytes[MAGIC.length];
  if (version !== VERSION) {
    throw new EncryptError(
      `unsupported version: 0x${version.toString(16).padStart(2, "0")}`
    );
  }
  if (!password) {
    throw new EncryptError("password cannot be empty");
  }

  let offset = MAGIC.length + 1;
  const salt = bytes.subarray(offset, offset + SALT_SIZE);
  offset += SALT_SIZE;
  const nonce = bytes.subarray(offset, offset + NONCE_SIZE);
  offset += NONCE_SIZE;
  const tag = bytes.subarray(offset, offset + TAG_SIZE);
  offset += TAG_SIZE;
  const ciphertext = bytes.subarray(offset);

  const key = deriveKey(password, salt);
  const decipher = createDecipheriv("aes-256-gcm", key, nonce);
  decipher.setAuthTag(tag);
  try {
    return Buffer.concat([decipher.update(ciphertext), decipher.final()]);
  } catch {
    throw new EncryptError("authentication failed");
  }
}

function toWirePayload(records: readonly ATBRecord[]): WirePayload {
  const wireRecords: WireRecord[] = [];
  let prev = GENESIS_HASH;
  for (let i = 0; i < records.length; i++) {
    const source = records[i].event;
    if (typeof source.type !== "string") {
      throw new EncryptError("record event is missing type");
    }
    const wireEvent: Event = {
      seq: i + 1,
      prev_hash: prev,
      type: source.type,
      data: source.data,
    };
    if (source.actor_id !== undefined) {
      wireEvent.actor_id = source.actor_id;
    }
    if (source.org_id !== undefined) {
      wireEvent.org_id = source.org_id;
    }
    if (source.workspace_id !== undefined) {
      wireEvent.workspace_id = source.workspace_id;
    }
    const wireHash = computeWireHash(wireEvent, prev);
    wireRecords.push({
      event: wireEvent,
      hash: wireHash,
    });
    prev = wireHash;
  }

  const headHash =
    wireRecords.length > 0
      ? wireRecords[wireRecords.length - 1].hash
      : GENESIS_HASH;
  return {
    head_hash: headHash,
    records: wireRecords,
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

  const parsedRecords: WireRecord[] = recordsValue.map((item) => {
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

  // Verify the decrypted wire payload chain (cross-language canonical form).
  let prevWire = GENESIS_HASH;
  for (let i = 0; i < parsedRecords.length; i++) {
    const record = parsedRecords[i];
    const event: Event = {
      seq: i + 1,
      prev_hash: prevWire,
      type: record.event.type,
      data: record.event.data,
    };
    if (record.event.actor_id !== undefined) {
      event.actor_id = record.event.actor_id;
    }
    if (record.event.org_id !== undefined) {
      event.org_id = record.event.org_id;
    }
    if (record.event.workspace_id !== undefined) {
      event.workspace_id = record.event.workspace_id;
    }
    if (record.event.seq !== i + 1 || record.event.prev_hash !== prevWire) {
      throw new EncryptError("decrypted payload failed hash-chain verification");
    }
    const computed = computeWireHash(event, prevWire);
    if (computed !== record.hash) {
      throw new EncryptError("decrypted payload failed hash-chain verification");
    }
    prevWire = computed;
  }
  if (headHash !== prevWire) {
    throw new EncryptError(
      `decrypted payload head_hash mismatch: expected ${headHash}, got ${prevWire}`
    );
  }

  // Convert verified wire records into current SDK in-memory shape.
  const records: ATBRecord[] = [];
  let prevTS = GENESIS_HASH;
  for (let i = 0; i < parsedRecords.length; i++) {
    const wire = parsedRecords[i];
    const event: Event = {
      seq: i + 1,
      prev_hash: prevTS,
      type: wire.event.type,
      data: wire.event.data,
    };
    if (wire.event.actor_id !== undefined) {
      event.actor_id = wire.event.actor_id;
    }
    if (wire.event.org_id !== undefined) {
      event.org_id = wire.event.org_id;
    }
    if (wire.event.workspace_id !== undefined) {
      event.workspace_id = wire.event.workspace_id;
    }
    const hash = computeHash(event, prevTS);
    records.push({ event, hash });
    prevTS = hash;
  }

  return {
    headHash,
    records,
  };
}

export async function encryptBundle(
  records: readonly ATBRecord[],
  password: string,
  options: EncryptOptions = {}
): Promise<Uint8Array> {
  const payload = toWirePayload(records);
  const canonical = canonicalize(payload);
  return encryptRaw(Buffer.from(canonical, "utf8"), password, options);
}

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
