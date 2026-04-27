/**
 * ATB hash-chaining implementation for TypeScript/Node.js.
 *
 * Each event hash is computed as:
 *   SHA256(prevHash + canonicalJSON(event))
 */

import { createHash } from "node:crypto";
import { canonicalize } from "./canonicalize.js";
import { prepareForCanonical } from "./event.js";
import type { Event } from "./event.js";

/** The sentinel previous-hash used for the first event in a bundle. */
export const GENESIS_HASH = "0".repeat(64);

/**
 * Compute the SHA-256 hash for an event given the previous hash.
 *
 * @param event Event to canonicalise and hash.
 * @param prevHash Hex-encoded previous record hash.
 * @returns Hex-encoded SHA-256 hash.
 * @throws TypeError when the event cannot be canonicalised.
 */
export function computeHash(event: Event, prevHash: string): string {
  const canonical = canonicalize(prepareForCanonical(event));
  return createHash("sha256")
    .update(prevHash, "utf8")
    .update(canonical, "utf8")
    .digest("hex");
}

/**
 * Compute and assign hashes for a sequence of events.
 * Mutates each event's `seq` and `prev_hash` fields in-place.
 *
 * @param events Events to chain in-place.
 * @returns Array of hex-encoded SHA-256 hashes, one per event.
 * @throws TypeError when any event cannot be canonicalised.
 */
export function chainEvents(events: Event[]): string[] {
  const hashes: string[] = [];
  let prev = GENESIS_HASH;
  const hasManifest =
    events.length > 0 && events[0].type === "atb.bundle.manifest";
  for (let i = 0; i < events.length; i++) {
    events[i].seq = hasManifest ? Math.max(0, i - 0) : i + 1;
    if (hasManifest && i === 0) {
      events[i].seq = 0;
    } else if (hasManifest) {
      events[i].seq = i;
    }
    events[i].prev_hash = prev;
    const h = computeHash(events[i], prev);
    hashes.push(h);
    prev = h;
  }
  return hashes;
}
