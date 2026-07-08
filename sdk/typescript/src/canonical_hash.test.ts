/**
 * Cross-language canonical-hash golden test for the TypeScript SDK.
 *
 * Loads internal/hash/testdata/canonical_vectors.json (the same fixture
 * used by the Go runtime and the Python SDK) and asserts the TypeScript
 * SDK's canonicaliser and hash function produce identical output.
 *
 * Drift here means the TypeScript SDK has diverged from the Go source of
 * truth and bundles signed by Go will not re-verify in TypeScript.
 */

import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { canonicalize } from "./canonicalize.js";
import { WorkflowContext } from "./workflow-common.js";
import type { WorkflowPolicyDecision } from "./workflow-common.js";

interface CanonicalVector {
  description: string;
  event: unknown;
  prev_hash: string;
  expected_canonical_bytes: string;
  expected_hash: string;
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const VECTORS_PATH = resolve(
  __dirname,
  "..",
  "..",
  "..",
  "internal",
  "hash",
  "testdata",
  "canonical_vectors.json"
);

function loadVectors(): CanonicalVector[] {
  const raw = readFileSync(VECTORS_PATH, "utf8");
  return JSON.parse(raw) as CanonicalVector[];
}

function firstDiff(a: Uint8Array, b: Uint8Array): number {
  const n = Math.min(a.length, b.length);
  for (let i = 0; i < n; i++) {
    if (a[i] !== b[i]) return i;
  }
  return n;
}

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

describe("canonical hash golden vectors", () => {
  const vectors = loadVectors();

  it.each(vectors)("$description", (vector) => {
    const expectedCanonical = Buffer.from(
      vector.expected_canonical_bytes,
      "base64"
    );
    const gotCanonicalString = canonicalize(vector.event);
    const gotCanonical = Buffer.from(gotCanonicalString, "utf8");

    if (Buffer.compare(gotCanonical, expectedCanonical) !== 0) {
      const idx = firstDiff(gotCanonical, expectedCanonical);
      throw new Error(
        `canonical bytes drift\n` +
          `  case: ${vector.description}\n` +
          `  first diff at byte ${idx}\n` +
          `  expected: ${toHex(expectedCanonical)}\n` +
          `  got:      ${toHex(gotCanonical)}`
      );
    }

    const gotHash = createHash("sha256")
      .update(vector.prev_hash, "utf8")
      .update(gotCanonical)
      .digest("hex");

    expect(gotHash).toBe(vector.expected_hash);
  });
});

interface PolicyDefaultsFixture {
  action_id: string;
  input: WorkflowPolicyDecision;
  expected_payload: Record<string, unknown>;
}

describe("policy decision defaults", () => {
  it("match the shared cross-SDK fixture", () => {
    const fixturePath = resolve(
      __dirname,
      "..",
      "..",
      "..",
      "internal",
      "hash",
      "testdata",
      "policy_decision_defaults.json"
    );
    const fixture = JSON.parse(
      readFileSync(fixturePath, "utf8")
    ) as PolicyDefaultsFixture;
    const payload = new WorkflowContext().policyPayload(
      fixture.action_id,
      fixture.input
    );
    expect(payload).toEqual(fixture.expected_payload);
  });
});
