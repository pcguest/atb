/**
 * ATB hash-chaining implementation for TypeScript/Node.js.
 *
 * Each event hash is computed as:
 *   SHA256(prevHash + canonicalJSON(event))
 */

import { createHash } from "node:crypto";
import { canonicalize } from "./canonicalize.js";
import type { ATBEvent } from "./types.js";

/** The sentinel previous-hash used for the first event in a bundle. */
export const GENESIS_HASH = "0".repeat(64);

/**
 * Compute the SHA-256 hash for an event given the previous hash.
 */
export function computeHash(event: ATBEvent, prevHash: string): string {
  const canonical = canonicalize(event);
  return createHash("sha256")
    .update(prevHash, "utf8")
    .update(canonical, "utf8")
    .digest("hex");
}

/**
 * Compute and assign hashes for a sequence of events.
 * Mutates each event's `seq` and `prevHash` fields in-place.
 *
 * @returns Array of hex-encoded SHA-256 hashes, one per event.
 */
export function chainEvents(events: ATBEvent[]): string[] {
  const hashes: string[] = [];
  let prev = GENESIS_HASH;
  for (let i = 0; i < events.length; i++) {
    events[i].seq = i + 1;
    events[i].prevHash = prev;
    const h = computeHash(events[i], prev);
    hashes.push(h);
    prev = h;
  }
  return hashes;
}
