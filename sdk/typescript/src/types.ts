/**
 * Core type definitions for the ATB TypeScript SDK.
 */

import type { Event } from "./event.js";

/** A single auditable event in an ATB bundle. */
export type ATBEvent = Event;

/** A single record in an ATB bundle file (event + its hash). */
export interface ATBRecord {
  event: Event;
  hash: string;
}

/** Options for creating a Bundle. */
export interface BundleOptions {
  /** Path to the bundle file. Defaults to `run.atb/bundle.atb`. */
  path?: string;
}

/**
 * The wire shape of an `atb.bundle.signature` record's `data` payload.
 * Mandatory fields are present on every signature; the optional provenance
 * fields are emitted by KMS backends and by current local writers, but are
 * absent on bundles signed before they were introduced. The reader treats
 * a missing `algorithm` as `"ed25519"` for backward compatibility.
 */
export interface BundleSignature {
  bundle_hash: string;
  signature: string;
  pub_key: string;
  algorithm?: "ed25519" | "ecdsa-p256";
  key_id?: string;
  backend?: "local" | "aws-kms" | "gcp-kms" | "vault";
  signed_at?: string;
}
