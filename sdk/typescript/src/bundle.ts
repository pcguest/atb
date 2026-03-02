/**
 * ATB Bundle — the primary interface for creating and managing ATB trace bundles.
 */

import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname } from "node:path";
import { GENESIS_HASH, computeHash } from "./hash.js";
import type { ATBEvent, ATBRecord, BundleOptions } from "./types.js";

const DEFAULT_PATH = "run.atb/bundle.atb";

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
 * import { Bundle } from "@atb-dev/sdk";
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
  }

  /** Append a new event to the bundle. */
  append(type: string, data: unknown): ATBRecord {
    const prevHash =
      this.records.length > 0
        ? this.records[this.records.length - 1].hash
        : GENESIS_HASH;

    const event: ATBEvent = {
      seq: this.records.length + 1,
      prevHash,
      type,
      data,
    };

    const hash = computeHash(event, prevHash);
    const record: ATBRecord = { event, hash };
    this.records.push(record);
    return record;
  }

  /** Verify the integrity of the entire bundle. Throws on tampering. */
  verify(): void {
    let prev = GENESIS_HASH;
    for (let i = 0; i < this.records.length; i++) {
      const record = this.records[i];
      const event = { ...record.event, seq: i + 1, prevHash: prev };
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
    const content = readFileSync(target, "utf8");
    for (const line of content.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      const record = JSON.parse(trimmed) as ATBRecord;
      bundle.records.push(record);
    }
    return bundle;
  }

  get length(): number {
    return this.records.length;
  }
}
