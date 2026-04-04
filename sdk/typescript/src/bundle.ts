/**
 * ATB Bundle — the primary interface for creating and managing ATB trace bundles.
 */

import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { randomBytes } from "node:crypto";
import { dirname } from "node:path";
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
const MANIFEST_VERSION = "1";

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
 * import { Bundle } from "@pcguest/atb-sdk";
 *
 * const bundle = new Bundle();
 * bundle.append("dev.session", { date: "2025-01-15" });
 * bundle.save();
 *
 * const loaded = Bundle.load();
 * loaded.verify();
 * ```
 */
export class Bundle {
  readonly records: ATBRecord[] = [];
  private readonly path: string;

  constructor(options: BundleOptions = {}) {
    this.path = options.path ?? DEFAULT_PATH;
    this.records.push(this.createManifestRecord());
  }

  /** Append a new event to the bundle. */
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

  /** Verify the integrity of the entire bundle. Throws on tampering. */
  verify(): void {
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
  }

  /** Save the bundle to disk in NDJSON format. */
  save(path?: string): void {
    const target = path ?? this.path;
    mkdirSync(dirname(target), { recursive: true });
    const lines = this.records.map((r) => JSON.stringify(r));
    writeFileSync(target, lines.join("\n") + "\n", "utf8");
  }

  /** Load a bundle from disk. */
  static load(path?: string): Bundle {
    const target = path ?? DEFAULT_PATH;
    const bundle = new Bundle({ path: target });
    bundle.records.length = 0;
    const content = readFileSync(target, "utf8");
    const lines = content.split("\n");
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const trimmed = line.trim();
      if (!trimmed) continue;
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
    return bundle;
  }

  /** Encrypt this bundle to ATBE bytes. */
  async encrypt(password: string): Promise<Uint8Array> {
    this.verify();
    return encryptBundle(this.records, password);
  }

  /** Decrypt ATBE bytes into a verified bundle. */
  static async decrypt(password: string, data: Uint8Array): Promise<Bundle> {
    const decoded = await decryptBundle(password, data);
    const bundle = new Bundle();
    bundle.records.length = 0;
    bundle.records.push(...decoded.records);
    bundle.verify();
    return bundle;
  }

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
      data: JSON.stringify({
        version: MANIFEST_VERSION,
        created_at: createdAt,
        bundle_id: randomBytes(16).toString("hex"),
      }),
      timestamp: createdAt,
    };
    return { event, hash: computeHash(event, GENESIS_HASH) };
  }
}

function nowRFC3339(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}
