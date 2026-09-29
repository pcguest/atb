/**
 * Canonical envelope parity guard: the TypeScript canonical representation must
 * cover every property the normative envelope schema declares.
 *
 * This protects against the class of defect where a future optional canonical
 * envelope field is added to the Go runtime and schema but silently omitted
 * from the TypeScript fixed-field reconstruction, causing hash divergence.
 *
 * The guard covers both the flat envelope `properties` and the nested
 * `$defs.acquisition` / `$defs.acquisition_checkpoint` properties, because the
 * acquisition parser is also a hand-maintained field list.
 */
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { parseAcquisition, prepareForCanonical, type Event } from "./event.js";
import { GENESIS_HASH } from "./hash.js";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
const schemaPath = resolve(repoRoot, "schemas", "event.v1.json");

type JsonSchema = {
  properties: Record<string, unknown>;
  $defs: Record<string, { properties?: Record<string, unknown> }>;
};

function loadSchema(): JsonSchema {
  return JSON.parse(readFileSync(schemaPath, "utf8")) as JsonSchema;
}

function envelopePropertyKeys(): string[] {
  return Object.keys(loadSchema().properties).sort();
}

function defPropertyKeys(defName: string): string[] {
  const def = loadSchema().$defs[defName];
  if (!def || !def.properties) {
    throw new Error(`schemas/event.v1.json is missing $defs.${defName}.properties`);
  }
  return Object.keys(def.properties).sort();
}

// Populated with every declared optional field so the guard can detect a
// schema property that the TypeScript representation does not cover.
const FULLY_POPULATED_ACQUISITION = {
  mode: "retrospective",
  source_system: "chatlog",
  source_record_id: "r1",
  source_timestamp: "2026-01-01T10:00:00Z",
  acquired_at: "2026-01-02T09:00:00Z",
  source_digest: `sha256:${"9".repeat(64)}`,
  adapter: "atb.chatlog.generic-jsonl",
  adapter_version: "1.0.0",
  checkpoint: {
    source_system: "chatlog",
    acquisition_stream: "chatlog.jsonl",
    position: "r1",
    observed_at: "2026-01-02T09:00:00Z",
    adapter: "atb.chatlog.generic-jsonl",
    adapter_version: "1.0.0",
  },
};

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
  acquisition: FULLY_POPULATED_ACQUISITION,
};

describe("canonical envelope parity with schemas/event.v1.json", () => {
  it("represents every schema envelope property in canonical preparation", () => {
    const represented = Object.keys(prepareForCanonical(FULLY_POPULATED_EVENT)).sort();
    expect(represented).toEqual(envelopePropertyKeys());
  });

  it("represents every $defs.acquisition property", () => {
    const acquisition = parseAcquisition(FULLY_POPULATED_ACQUISITION);
    expect(Object.keys(acquisition).sort()).toEqual(defPropertyKeys("acquisition"));
  });

  it("represents every $defs.acquisition_checkpoint property", () => {
    const acquisition = parseAcquisition(FULLY_POPULATED_ACQUISITION);
    expect(acquisition.checkpoint).toBeDefined();
    expect(Object.keys(acquisition.checkpoint ?? {}).sort()).toEqual(
      defPropertyKeys("acquisition_checkpoint")
    );
  });
});
