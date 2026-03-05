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
