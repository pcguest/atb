/**
 * Core type definitions for the ATB TypeScript SDK.
 */

/** A single auditable event in an ATB bundle. */
export interface ATBEvent {
  /** 1-based sequence number within the bundle. */
  seq: number;
  /** Hex-encoded SHA-256 hash of the preceding event (or GENESIS_HASH). */
  prevHash: string;
  /** Dot-namespaced event type identifier. */
  type: string;
  /** Arbitrary JSON-serialisable payload. */
  data: unknown;
}

/** A single record in an ATB bundle file (event + its hash). */
export interface ATBRecord {
  event: ATBEvent;
  hash: string;
}

/** Options for creating a Bundle. */
export interface BundleOptions {
  /** Path to the bundle file. Defaults to `run.atb/bundle.atb`. */
  path?: string;
}
