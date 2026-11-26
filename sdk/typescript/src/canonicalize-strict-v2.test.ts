/**
 * Conformance tests for the additive `atb-jcs-rfc8785-2` TypeScript profile.
 *
 * The v1 profile remains frozen. These tests execute the authored v2 vectors,
 * re-run the v1 negative vectors through v2, check raw round-trip closure, and
 * pin the canonical-output policy boundaries.
 */

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  StrictCanonicalizationError,
  canonicalizeStrict,
  canonicalizeStrictRaw,
  canonicalizeStrictRawV2,
  canonicalizeStrictV2,
} from "./canonicalize-strict.js";

interface PositiveCase {
  id: string;
  category: string;
  input_utf8_hex: string;
  canonical_utf8_hex: string;
}

interface PositiveFixture {
  schema: string;
  profile: string;
  status: string;
  cases: PositiveCase[];
}

interface NegativeCase {
  id: string;
  category: string;
  input_utf8_hex: string;
  expected_error: string;
}

interface NegativeFixture {
  schema: string;
  profile: string;
  status: string;
  cases: NegativeCase[];
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const JCS_DIR = resolve(__dirname, "..", "..", "..", "test", "jcs");

function loadFixture<T>(name: string): T {
  return JSON.parse(readFileSync(resolve(JCS_DIR, name), "utf8")) as T;
}

function expectStrictError(fn: () => unknown, errorClass: string): void {
  try {
    fn();
  } catch (err) {
    expect(err).toBeInstanceOf(StrictCanonicalizationError);
    expect((err as StrictCanonicalizationError).errorClass).toBe(errorClass);
    return;
  }
  throw new Error(`expected StrictCanonicalizationError(${errorClass})`);
}

function canonicalHex(input: string): string {
  return Buffer.from(input, "utf8").toString("hex");
}

const V2_POSITIVE = loadFixture<PositiveFixture>(
  "strict_v2_positive_vectors.json"
);
const V2_NEGATIVE = loadFixture<NegativeFixture>(
  "strict_v2_negative_vectors.json"
);
const V1_NEGATIVE = loadFixture<NegativeFixture>("strict_negative_vectors.json");

describe("v2 fixture metadata", () => {
  it("declares the v2 positive and negative profiles", () => {
    expect(V2_POSITIVE.schema).toBe("atb.jcs.strict-positive-vectors.v1");
    expect(V2_POSITIVE.profile).toBe("atb-jcs-rfc8785-2");
    expect(V2_POSITIVE.status).toBe("executable");
    expect(V2_POSITIVE.cases).toHaveLength(24);

    expect(V2_NEGATIVE.schema).toBe("atb.jcs.strict-negative-vectors.v1");
    expect(V2_NEGATIVE.profile).toBe("atb-jcs-rfc8785-2");
    expect(V2_NEGATIVE.status).toBe("executable");
    expect(V2_NEGATIVE.cases).toHaveLength(9);

    expect(new Set(V2_POSITIVE.cases.map((c) => c.id)).size).toBe(24);
    expect(new Set(V2_NEGATIVE.cases.map((c) => c.id)).size).toBe(9);
  });
});

describe("v2 positive vectors", () => {
  it.each(V2_POSITIVE.cases)("$id", (testCase) => {
    const input = Buffer.from(testCase.input_utf8_hex, "hex");
    const got = canonicalizeStrictRawV2(input);
    expect(canonicalHex(got)).toBe(testCase.canonical_utf8_hex);

    expect(canonicalizeStrictRawV2(got)).toBe(got);
  });
});

describe("v2 negative vectors", () => {
  it.each(V2_NEGATIVE.cases)("$id", (testCase) => {
    const input = Buffer.from(testCase.input_utf8_hex, "hex");
    expectStrictError(
      () => canonicalizeStrictRawV2(input),
      testCase.expected_error
    );
  });
});

describe("v2 fraction interval both signs", () => {
  it("rejects 2^55 and negative 2^53 fraction inputs", () => {
    expectStrictError(
      () => canonicalizeStrictRawV2('{"n":36028797018963968.0}'),
      "number_out_of_policy"
    );
    expectStrictError(
      () => canonicalizeStrictRawV2('{"n":-9007199254740992.0}'),
      "number_out_of_policy"
    );
  });
});

describe("v1 negative vectors through v2", () => {
  it.each(V1_NEGATIVE.cases)("$id", (testCase) => {
    const input = Buffer.from(testCase.input_utf8_hex, "hex");
    expectStrictError(
      () => canonicalizeStrictRawV2(input),
      testCase.expected_error
    );
  });
});

describe("v2 value mode", () => {
  it("rejects 1e20", () => {
    expectStrictError(() => canonicalizeStrictV2(1e20), "number_out_of_policy");
  });

  it("accepts 1e21 with exponential output", () => {
    expect(canonicalizeStrictV2(1e21)).toBe("1e+21");
  });

  it("accepts 1e-6 with fixed output", () => {
    expect(canonicalizeStrictV2(1e-6)).toBe("0.000001");
  });

  it("accepts the maximum safe integer", () => {
    expect(canonicalizeStrictV2(9007199254740991n)).toBe(
      "9007199254740991"
    );
  });

  it("rejects plain 10^21", () => {
    expectStrictError(
      () => canonicalizeStrictV2(10n ** 21n),
      "number_out_of_policy"
    );
  });
});

describe("v2 plain 10^21 lexical asymmetry", () => {
  it("rejects the plain raw token but accepts 1e21", () => {
    expectStrictError(
      () => canonicalizeStrictRawV2("1000000000000000000000"),
      "number_out_of_policy"
    );
    expect(canonicalizeStrictRawV2("1e21")).toBe("1e+21");
  });
});

describe("v1 profile remains frozen", () => {
  it("keeps the v1 1e20 outputs", () => {
    expect(canonicalizeStrictRaw('{"n":1e20}')).toBe(
      '{"n":100000000000000000000}'
    );
    expect(canonicalizeStrict(1e20)).toBe("100000000000000000000");
    expect(canonicalizeStrict(1e-6)).toBe("0.000001");
  });
});
