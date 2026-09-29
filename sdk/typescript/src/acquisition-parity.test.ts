/**
 * Acquisition envelope parity: PRODUCER = Go, CONSUMER/VERIFIER = TypeScript.
 *
 * The canonical Go runtime serialises and hashes an optional top-level
 * `acquisition` field. Before this repair the TypeScript SDK rebuilt events from
 * a fixed field list, silently dropping `acquisition`, which changed the
 * canonical representation and made `Bundle.load().verify()` fail on a
 * Go-produced acquisition-bearing bundle.
 *
 * The end-to-end case generates a bundle with the authoritative Go CLI and
 * verifies it with the TypeScript SDK. The vector cases pin canonical bytes and
 * record hashes computed by the Go implementation.
 */
import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import { parseEvent, prepareForCanonical } from "./event.js";
import { computeHash, GENESIS_HASH } from "./hash.js";

const NINE = "9".repeat(64);
const A64 = "a".repeat(64);

const FULL_ACQUISITION = {
  mode: "retrospective",
  source_system: "chatlog",
  source_record_id: "r1",
  source_timestamp: "2026-01-01T10:00:00Z",
  acquired_at: "2026-01-02T09:00:00Z",
  source_digest: `sha256:${NINE}`,
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

// Vectors computed by the authoritative Go implementation
// (internal/canonicalize + internal/hash) on an identical event shape.
const VECTORS = [
  {
    name: "no acquisition",
    prev: GENESIS_HASH,
    raw: { seq: 1, prev_hash: GENESIS_HASH, type: "dev.session", data: {} },
    canonical: `{"data":{},"prev_hash":"${GENESIS_HASH}","seq":1,"type":"dev.session"}`,
    hash: "451747960eadec3c609185007db5bfab72d5223c3f9cf4e1c93e915bf3a0b96a",
  },
  {
    name: "minimal acquisition",
    prev: GENESIS_HASH,
    raw: {
      seq: 1,
      prev_hash: GENESIS_HASH,
      type: "dev.session",
      data: {},
      acquisition: { mode: "retrospective", source_system: "chatlog" },
    },
    canonical: `{"acquisition":{"mode":"retrospective","source_system":"chatlog"},"data":{},"prev_hash":"${GENESIS_HASH}","seq":1,"type":"dev.session"}`,
    hash: "b2a4c0a96add6ddd810049e8b7b59c8e7d2372b053eaa0a96239050e9fc59706",
  },
  {
    name: "full acquisition",
    prev: A64,
    raw: {
      seq: 2,
      prev_hash: A64,
      type: "ai.request.received",
      hash_algo: "sha256",
      timestamp: "2026-01-01T10:00:00Z",
      data: { request_id: "r1" },
      acquisition: FULL_ACQUISITION,
    },
    canonical: `{"acquisition":{"acquired_at":"2026-01-02T09:00:00Z","adapter":"atb.chatlog.generic-jsonl","adapter_version":"1.0.0","checkpoint":{"acquisition_stream":"chatlog.jsonl","adapter":"atb.chatlog.generic-jsonl","adapter_version":"1.0.0","observed_at":"2026-01-02T09:00:00Z","position":"r1","source_system":"chatlog"},"mode":"retrospective","source_digest":"sha256:${NINE}","source_record_id":"r1","source_system":"chatlog","source_timestamp":"2026-01-01T10:00:00Z"},"data":{"request_id":"r1"},"hash_algo":"sha256","prev_hash":"${A64}","seq":2,"timestamp":"2026-01-01T10:00:00Z","type":"ai.request.received"}`,
    hash: "26cce31f91da29439272334dd02c45d2f4f386992da5ef8e8027dbc8e5001d71",
  },
];

describe("acquisition canonical vectors (Go-produced expectations)", () => {
  for (const vector of VECTORS) {
    it(`preserves canonical bytes and hash for ${vector.name}`, () => {
      const event = parseEvent(vector.raw);
      expect(canonicalize(prepareForCanonical(event))).toBe(vector.canonical);
      expect(computeHash(event, vector.prev)).toBe(vector.hash);
    });
  }

  it("does not introduce an empty acquisition object when absent", () => {
    const event = parseEvent({
      seq: 1,
      prev_hash: GENESIS_HASH,
      type: "dev.session",
      data: {},
    });
    expect(event.acquisition).toBeUndefined();
    expect(prepareForCanonical(event)).not.toHaveProperty("acquisition");
  });

  it("removing acquisition changes the canonical bytes and hash", () => {
    const withAcq = parseEvent(VECTORS[2].raw);
    const withoutAcq = parseEvent({
      seq: 2,
      prev_hash: A64,
      type: "ai.request.received",
      hash_algo: "sha256",
      timestamp: "2026-01-01T10:00:00Z",
      data: { request_id: "r1" },
    });
    expect(canonicalize(prepareForCanonical(withAcq))).not.toBe(
      canonicalize(prepareForCanonical(withoutAcq))
    );
    expect(computeHash(withAcq, A64)).not.toBe(computeHash(withoutAcq, A64));
  });

  it("is insensitive to acquisition key insertion order", () => {
    const ordered = {
      seq: 1,
      prev_hash: GENESIS_HASH,
      type: "dev.session",
      data: {},
      acquisition: { mode: "retrospective", source_system: "chatlog" },
    };
    const reordered = {
      acquisition: { source_system: "chatlog", mode: "retrospective" },
      data: {},
      type: "dev.session",
      prev_hash: GENESIS_HASH,
      seq: 1,
    };
    expect(canonicalize(prepareForCanonical(parseEvent(reordered)))).toBe(
      canonicalize(prepareForCanonical(parseEvent(ordered)))
    );
  });

  it("rejects malformed acquisition rather than coercing it", () => {
    const base = { seq: 1, prev_hash: GENESIS_HASH, type: "dev.session", data: {} };
    expect(() => parseEvent({ ...base, acquisition: "chatlog" })).toThrow(TypeError);
    expect(() => parseEvent({ ...base, acquisition: [1, 2] })).toThrow(TypeError);
    expect(() => parseEvent({ ...base, acquisition: { mode: 5 } })).toThrow(TypeError);
    expect(() => parseEvent({ ...base, acquisition: { checkpoint: "x" } })).toThrow(
      TypeError
    );
    expect(() =>
      parseEvent({ ...base, acquisition: { checkpoint: { source_system: "chatlog" } } })
    ).toThrow(TypeError);
  });
});

describe("Go -> TypeScript acquisition bundle verification", () => {
  it("loads and verifies a Go-produced acquisition-bearing bundle", () => {
    const dir = mkdtempSync(join(tmpdir(), "atb-ts-acq-"));
    const chatlogPath = join(dir, "chatlog.jsonl");
    const bundlePath = join(dir, "bundle.atb");
    const checkpointPath = join(dir, "checkpoint.json");
    const gocache = join(dir, "gocache");

    writeFileSync(
      chatlogPath,
      [
        JSON.stringify({
          role: "user",
          content: "hello",
          timestamp: "2026-01-01T10:00:00Z",
          model: "gpt-4",
          model_provider: "openai",
          conversation_id: "c1",
          session_id: "s1",
          request_id: "r1",
        }),
        JSON.stringify({
          role: "assistant",
          content: "hi there",
          timestamp: "2026-01-01T10:00:01Z",
          model: "gpt-4",
        }),
      ].join("\n") + "\n",
      "utf8"
    );

    const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
    const result = spawnSync(
      "go",
      [
        "run",
        "./cmd/atb",
        "import",
        "chatlog",
        "--from",
        "generic-jsonl",
        "--input",
        chatlogPath,
        "--bundle",
        bundlePath,
        "--checkpoint",
        checkpointPath,
      ],
      { cwd: repoRoot, encoding: "utf8", env: { ...process.env, GOCACHE: gocache } }
    );
    expect(result.status, result.stderr).toBe(0);

    // CONSUMER/VERIFIER: TypeScript. This failed before the repair because
    // parseEvent dropped acquisition, changing the recomputed hash.
    const bundle = Bundle.load(bundlePath);
    expect(() => bundle.verify()).not.toThrow();

    const acquisitionEvents = bundle.records.filter(
      (record) => record.event.acquisition !== undefined
    );
    expect(acquisitionEvents.length).toBeGreaterThan(0);

    const first = acquisitionEvents[0].event.acquisition;
    expect(first?.mode).toBe("retrospective");
    expect(first?.source_system).toBe("chatlog");
    expect(first?.checkpoint?.position).toBeTruthy();
    expect(first?.adapter).toBe("atb.chatlog.generic-jsonl");
  }, 180_000);
});
