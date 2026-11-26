/**
 * Validate the executable JCS canonicalisation-profile fixtures.
 *
 * The fixtures cover the frozen legacy profile `atb-jcs-legacy-1`, the frozen
 * strict profile `atb-jcs-rfc8785-1`, the additive `atb-jcs-rfc8785-2`
 * profile, and the numeric-parity case. This test validates the fixtures and
 * pins the TypeScript legacy canonicaliser. The strict fixtures are executed
 * against the strict parser by `canonicalize-strict.test.ts` and
 * `canonicalize-strict-v2.test.ts`; this test checks their status, metadata,
 * declared error classes, and the frozen numeric-parity case.
 */

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { TextDecoder } from "node:util";
import { describe, expect, it } from "vitest";
import { canonicalize } from "./canonicalize.js";
import { canonicalizeStrictRawV2 } from "./canonicalize-strict.js";

interface Divergence {
  go: boolean;
  python: boolean;
  typescript: boolean;
}

interface PositiveCase {
  id: string;
  category: string;
  input_utf8_hex: string;
  canonical_utf8_hex: string;
  legacy_divergence: Divergence;
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

interface LegacyRawCase {
  id: string;
  input_utf8_hex: string;
  go_canonical_utf8_hex?: string;
  go_error?: string;
}

interface LegacyExpected {
  go: string;
  python: string;
  typescript: string;
}

interface LegacyValueCase {
  id: string;
  value_json: string;
  expected: LegacyExpected;
  equal_across_languages: boolean;
}

interface LegacyFixture {
  schema: string;
  profile: string;
  status: string;
  raw_cases: LegacyRawCase[];
  value_cases: LegacyValueCase[];
}

interface NumericParityExpected {
  go: string;
  python: string;
  typescript: string;
}

interface NumericParityCase {
  id: string;
  profile: string;
  input_utf8_hex: string;
  legacy_expected: NumericParityExpected;
  strict_expected: NumericParityExpected;
  notes: string;
}

interface NumericParityFixture {
  schema: string;
  profile: string;
  status: string;
  cases: NumericParityCase[];
}

interface ProfileSelectionCase {
  id: string;
  manifest_version: number;
  canonical_profile?: string;
  current_reader: string;
  strict_reader: string;
}

interface ProfileSelectionFixture {
  schema: string;
  status: string;
  profiles: { legacy: string; strict: string };
  manifest_field: string;
  manifest_version_strict: number;
  cases: ProfileSelectionCase[];
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const JCS_DIR = resolve(__dirname, "..", "..", "..", "test", "jcs");

const POSITIVE_SCHEMA = "atb.jcs.strict-positive-vectors.v1";
const NEGATIVE_SCHEMA = "atb.jcs.strict-negative-vectors.v1";
const NUMERIC_PARITY_SCHEMA = "atb.jcs.numeric-parity-cases.v1";
const LEGACY_SCHEMA = "atb.jcs.legacy-freeze-vectors.v1";
const PROFILE_SELECTION_SCHEMA = "atb.jcs.profile-selection-cases.v1";

const STRICT_PROFILE = "atb-jcs-rfc8785-1";
const V2_PROFILE = "atb-jcs-rfc8785-2";
const NUMERIC_PARITY_PROFILE = "atb-jcs-rfc8785-1-and-rfc8785-2";
const LEGACY_PROFILE = "atb-jcs-legacy-1";

const ALLOWED_ERROR_CLASSES = new Set([
  "duplicate_key",
  "invalid_utf8",
  "lone_surrogate",
  "trailing_data",
  "non_finite_number",
  "number_out_of_policy",
  "number_unrepresentable",
  "invalid_json",
  "depth_limit",
  "missing_profile",
  "unsupported_profile",
]);

const REQUIRED_POSITIVE_CATEGORIES = new Set([
  "key_ordering",
  "string_escaping",
  "numbers",
  "nesting",
]);

const REQUIRED_NEGATIVE_CATEGORIES = new Set([
  "duplicate_key",
  "invalid_utf8",
  "lone_surrogate",
  "trailing_data",
  "non_finite_number",
  "number_out_of_policy",
  "number_unrepresentable",
  "invalid_json",
]);

function loadFixture<T>(name: string): T {
  return JSON.parse(readFileSync(resolve(JCS_DIR, name), "utf8")) as T;
}

function decodeHex(hex: string, label: string): Buffer {
  const bytes = Buffer.from(hex, "hex");
  if (bytes.toString("hex") !== hex.toLowerCase()) {
    throw new Error(`${label}: not valid hex`);
  }
  return bytes;
}

function decodeHexUtf8(hex: string, label: string): string {
  // TextDecoder with fatal: true rejects malformed UTF-8 rather than
  // substituting U+FFFD, which is what the fixture assertions require.
  return new TextDecoder("utf-8", { fatal: true }).decode(decodeHex(hex, label));
}

const POSITIVE = loadFixture<PositiveFixture>("strict_positive_vectors.json");
const NEGATIVE = loadFixture<NegativeFixture>("strict_negative_vectors.json");
const V2_POSITIVE = loadFixture<PositiveFixture>(
  "strict_v2_positive_vectors.json"
);
const V2_NEGATIVE = loadFixture<NegativeFixture>(
  "strict_v2_negative_vectors.json"
);
const NUMERIC_PARITY = loadFixture<NumericParityFixture>(
  "numeric_parity_cases.json"
);
const LEGACY = loadFixture<LegacyFixture>("legacy_freeze_vectors.json");
const PROFILE_SELECTION = loadFixture<ProfileSelectionFixture>(
  "profile_selection_cases.json"
);

describe("JCS specification fixture metadata", () => {
  it("declares the strict positive profile and required categories", () => {
    expect(POSITIVE.schema).toBe(POSITIVE_SCHEMA);
    expect(POSITIVE.profile).toBe(STRICT_PROFILE);
    expect(POSITIVE.status).toBe("executable");
    expect(new Set(POSITIVE.cases.map((c) => c.id)).size).toBe(
      POSITIVE.cases.length
    );
    for (const category of REQUIRED_POSITIVE_CATEGORIES) {
      expect(POSITIVE.cases.some((c) => c.category === category)).toBe(true);
    }
  });

  it("declares the strict negative profile and required categories", () => {
    expect(NEGATIVE.schema).toBe(NEGATIVE_SCHEMA);
    expect(NEGATIVE.profile).toBe(STRICT_PROFILE);
    expect(NEGATIVE.status).toBe("executable");
    expect(new Set(NEGATIVE.cases.map((c) => c.id)).size).toBe(
      NEGATIVE.cases.length
    );
    for (const category of REQUIRED_NEGATIVE_CATEGORIES) {
      expect(NEGATIVE.cases.some((c) => c.category === category)).toBe(true);
    }
  });

  it("declares the v2 strict positive and negative profiles", () => {
    expect(V2_POSITIVE.schema).toBe(POSITIVE_SCHEMA);
    expect(V2_POSITIVE.profile).toBe(V2_PROFILE);
    expect(V2_POSITIVE.status).toBe("executable");
    expect(V2_POSITIVE.cases).toHaveLength(24);
    expect(new Set(V2_POSITIVE.cases.map((c) => c.id)).size).toBe(
      V2_POSITIVE.cases.length
    );
    for (const category of REQUIRED_POSITIVE_CATEGORIES) {
      expect(V2_POSITIVE.cases.some((c) => c.category === category)).toBe(true);
    }

    expect(V2_NEGATIVE.schema).toBe(NEGATIVE_SCHEMA);
    expect(V2_NEGATIVE.profile).toBe(V2_PROFILE);
    expect(V2_NEGATIVE.status).toBe("executable");
    expect(V2_NEGATIVE.cases).toHaveLength(9);
    expect(new Set(V2_NEGATIVE.cases.map((c) => c.id)).size).toBe(
      V2_NEGATIVE.cases.length
    );
    for (const testCase of V2_NEGATIVE.cases) {
      expect(ALLOWED_ERROR_CLASSES.has(testCase.expected_error)).toBe(true);
      expect(testCase.category).toBe(testCase.expected_error);
    }
  });

  it("declares the frozen numeric-parity case", () => {
    expect(NUMERIC_PARITY.schema).toBe(NUMERIC_PARITY_SCHEMA);
    expect(NUMERIC_PARITY.profile).toBe(NUMERIC_PARITY_PROFILE);
    expect(NUMERIC_PARITY.status).toBe("frozen");
    expect(NUMERIC_PARITY.cases).toHaveLength(1);

    const testCase = NUMERIC_PARITY.cases[0];
    expect(testCase.id).toBe("parity-1e-6-legacy-vs-strict");
    expect(testCase.profile).toBe(NUMERIC_PARITY_PROFILE);
    expect(testCase.input_utf8_hex).toBe("7b226e223a31652d367d");
    expect(testCase.legacy_expected).toEqual({
      go: "7b226e223a302e3030303030317d",
      python: "7b226e223a31652d30367d",
      typescript: "7b226e223a302e3030303030317d",
    });
    expect(testCase.strict_expected).toEqual({
      go: "7b226e223a302e3030303030317d",
      python: "7b226e223a302e3030303030317d",
      typescript: "7b226e223a302e3030303030317d",
    });
    expect(testCase.notes).toBeTruthy();
  });

  it("executes the numeric-parity case", () => {
    const testCase = NUMERIC_PARITY.cases[0];
    const input = decodeHexUtf8(testCase.input_utf8_hex, "input");
    const value = JSON.parse(input);

    expect(
      Buffer.from(canonicalize(value), "utf8").toString("hex")
    ).toBe(testCase.legacy_expected.typescript);
    expect(
      Buffer.from(canonicalizeStrictRawV2(input), "utf8").toString("hex")
    ).toBe(testCase.strict_expected.typescript);
  });

  it("declares the frozen legacy profile", () => {
    expect(LEGACY.schema).toBe(LEGACY_SCHEMA);
    expect(LEGACY.profile).toBe(LEGACY_PROFILE);
    expect(LEGACY.status).toBe("frozen");
    const ids = [
      ...LEGACY.raw_cases.map((c) => c.id),
      ...LEGACY.value_cases.map((c) => c.id),
    ];
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("declares the profile-selection contract", () => {
    expect(PROFILE_SELECTION.schema).toBe(PROFILE_SELECTION_SCHEMA);
    expect(PROFILE_SELECTION.profiles.legacy).toBe(LEGACY_PROFILE);
    expect(PROFILE_SELECTION.profiles.strict).toBe(STRICT_PROFILE);
    expect(PROFILE_SELECTION.manifest_field).toBe("canonical_profile");
    expect(PROFILE_SELECTION.manifest_version_strict).toBe(4);
    expect(new Set(PROFILE_SELECTION.cases.map((c) => c.id)).size).toBe(
      PROFILE_SELECTION.cases.length
    );
    for (const testCase of PROFILE_SELECTION.cases) {
      expect(testCase.current_reader).toBeTruthy();
      expect(testCase.strict_reader).toBeTruthy();
    }
  });

  it("uses unique case ids across fixtures", () => {
    const ids = [
      ...POSITIVE.cases.map((c) => c.id),
      ...NEGATIVE.cases.map((c) => c.id),
      ...LEGACY.raw_cases.map((c) => c.id),
      ...LEGACY.value_cases.map((c) => c.id),
      ...PROFILE_SELECTION.cases.map((c) => c.id),
      ...NUMERIC_PARITY.cases.map((c) => c.id),
    ];
    expect(new Set(ids).size).toBe(ids.length);
  });
});

describe("strict positive vectors", () => {
  it.each(POSITIVE.cases)("$id", (testCase) => {
    const inputText = decodeHexUtf8(testCase.input_utf8_hex, "input");
    const canonicalText = decodeHexUtf8(
      testCase.canonical_utf8_hex,
      "canonical"
    );

    const value = JSON.parse(inputText);
    JSON.parse(canonicalText);

    const got = Buffer.from(canonicalize(value), "utf8").toString("hex");
    const diverges = got !== testCase.canonical_utf8_hex;
    expect(diverges).toBe(testCase.legacy_divergence.typescript);
  });
});

describe("strict negative vectors", () => {
  it.each(NEGATIVE.cases)("$id", (testCase) => {
    // Fixture-integrity check: canonicalize-strict.test.ts executes these cases
    // against the strict parser and asserts the reported error class.
    decodeHex(testCase.input_utf8_hex, "input");
    expect(ALLOWED_ERROR_CLASSES.has(testCase.expected_error)).toBe(true);
    expect(REQUIRED_NEGATIVE_CATEGORIES.has(testCase.category)).toBe(true);
  });
});

describe("legacy freeze vectors", () => {
  it("declares exactly one Go outcome per raw case", () => {
    for (const rawCase of LEGACY.raw_cases) {
      decodeHex(rawCase.input_utf8_hex, rawCase.id);
      const hasCanonical = rawCase.go_canonical_utf8_hex !== undefined;
      const hasError = rawCase.go_error !== undefined;
      expect(hasCanonical).not.toBe(hasError);
    }
  });

  it.each(LEGACY.value_cases)("$id", (testCase) => {
    const value = JSON.parse(testCase.value_json);
    const got = Buffer.from(canonicalize(value), "utf8").toString("hex");
    expect(got).toBe(testCase.expected.typescript);

    if (testCase.equal_across_languages) {
      expect(testCase.expected.go).toBe(testCase.expected.python);
      expect(testCase.expected.go).toBe(testCase.expected.typescript);
    }
  });
});
