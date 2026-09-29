/**
 * Canonical envelope parity guard: the TypeScript canonical representation must
 * cover every property the normative envelope schema declares.
 *
 * This protects against the class of defect where a future optional canonical
 * envelope field is added to the Go runtime and schema but silently omitted
 * from the TypeScript fixed-field reconstruction, causing hash divergence.
 */
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { prepareForCanonical, type Event } from "./event.js";
import { GENESIS_HASH } from "./hash.js";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");

function envelopePropertyKeys(): string[] {
  const schemaPath = resolve(repoRoot, "schemas", "event.v1.json");
  const schema = JSON.parse(readFileSync(schemaPath, "utf8")) as {
    properties: Record<string, unknown>;
  };
  return Object.keys(schema.properties).sort();
}

const FULLY_POPULATED_EVENT: Event = {
  seq: 1,
  prev_hash: GENESIS_HASH,
  type: "dev.session",
  data: {},
  hash_algo: "sha256",
  actor_id: "actor-1",
  org_id: "org-1",
  workspace_id: "ws-1",
  timestamp: "2026-01-01T10:00:00Z",
  trace_id: "0".repeat(32),
  span_id: "0".repeat(16),
  parent_span_id: "0".repeat(16),
  acquisition: {
    mode: "retrospective",
    source_system: "chatlog",
    checkpoint: {
      source_system: "chatlog",
      acquisition_stream: "chatlog.jsonl",
      position: "r1",
      observed_at: "2026-01-01T10:00:00Z",
      adapter: "atb.chatlog.generic-jsonl",
    },
  },
};

describe("canonical envelope parity with schemas/event.v1.json", () => {
  it("represents every schema envelope property in canonical preparation", () => {
    const represented = Object.keys(prepareForCanonical(FULLY_POPULATED_EVENT)).sort();
    expect(represented).toEqual(envelopePropertyKeys());
  });
});
