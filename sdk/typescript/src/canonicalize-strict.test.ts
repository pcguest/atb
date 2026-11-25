/**
 * Conformance tests for the strict `atb-jcs-rfc8785-1` TypeScript
 * implementation, plus number-formatting and parser unit tests.
 *
 * Positive and negative vectors are the authored step-1 fixtures in
 * `test/jcs/`. The legacy `canonicalize` tests live elsewhere and are untouched.
 */

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  StrictCanonicalizationError,
  canonicalizeStrict,
  canonicalizeStrictRaw,
} from "./canonicalize-strict.js";

interface PositiveCase {
  id: string;
  category: string;
  input_utf8_hex: string;
  canonical_utf8_hex: string;
}

interface PositiveFixture {
  cases: PositiveCase[];
}

interface NegativeCase {
  id: string;
  category: string;
  input_utf8_hex: string;
  expected_error: string;
}

interface NegativeFixture {
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
  return Buffer.from(canonicalizeStrictRaw(input), "utf8").toString("hex");
}

const POSITIVE = loadFixture<PositiveFixture>("strict_positive_vectors.json");
const NEGATIVE = loadFixture<NegativeFixture>("strict_negative_vectors.json");

describe("strict positive vectors", () => {
  it.each(POSITIVE.cases)("$id", (testCase) => {
    const input = Buffer.from(testCase.input_utf8_hex, "hex");
    const got = Buffer.from(canonicalizeStrictRaw(input), "utf8").toString(
      "hex"
    );
    expect(got).toBe(testCase.canonical_utf8_hex);
  });
});

describe("strict negative vectors", () => {
  it.each(NEGATIVE.cases)("$id", (testCase) => {
    const input = Buffer.from(testCase.input_utf8_hex, "hex");
    expectStrictError(() => canonicalizeStrictRaw(input), testCase.expected_error);
  });
});

describe("number formatting (ECMA-262 Number::toString)", () => {
  const cases: Array<[number, string]> = [
    [0, "0"],
    [-0, "0"],
    [1.5, "1.5"],
    [0.1, "0.1"],
    [1e-7, "1e-7"],
    [1e-6, "0.000001"],
    [1e-5, "0.00001"],
    [1e20, "100000000000000000000"],
    [1e21, "1e+21"],
    [1e22, "1e+22"],
    [1e23, "1e+23"],
    [9007199254740991, "9007199254740991"],
    [1.2345678901234567, "1.2345678901234567"],
    [5e-324, "5e-324"],
    [1.7976931348623157e308, "1.7976931348623157e+308"],
  ];

  for (const [value, expected] of cases) {
    it(`formats ${String(value)} as ${expected}`, () => {
      expect(canonicalizeStrict(value)).toBe(expected);
    });
  }

  it("formats raw number tokens the same way", () => {
    expect(canonicalizeStrictRaw("-0")).toBe("0");
    expect(canonicalizeStrictRaw("1e-7")).toBe("1e-7");
    expect(canonicalizeStrictRaw("1e21")).toBe("1e+21");
  });
});

describe("raw mode parser rules", () => {
  it("rejects duplicate keys", () => {
    expectStrictError(() => canonicalizeStrictRaw('{"a":1,"a":2}'), "duplicate_key");
  });

  it("rejects escaped-equivalent duplicate keys", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1,"\\u0061":2}'),
      "duplicate_key"
    );
  });

  it("rejects duplicate keys at any depth", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"x":{"k":1,"k":2}}'),
      "duplicate_key"
    );
  });

  it("rejects invalid UTF-8 bytes rather than substituting U+FFFD", () => {
    const bytes = new Uint8Array([
      0x7b, 0x22, 0x73, 0x22, 0x3a, 0x22, 0xc3, 0x28, 0x22, 0x7d,
    ]);
    expectStrictError(() => canonicalizeStrictRaw(bytes), "invalid_utf8");
  });

  it("does not strip a leading UTF-8 BOM", () => {
    const bytes = new Uint8Array([0xef, 0xbb, 0xbf, 0x7b, 0x7d]);
    expectStrictError(() => canonicalizeStrictRaw(bytes), "invalid_json");
  });

  it("rejects lone high and low surrogate escapes", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"s":"\\uD800"}'),
      "lone_surrogate"
    );
    expectStrictError(
      () => canonicalizeStrictRaw('{"s":"\\uDC00"}'),
      "lone_surrogate"
    );
  });

  it("rejects raw lone surrogate code units", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('"' + "\uD800" + '"'),
      "lone_surrogate"
    );
    expectStrictError(
      () => canonicalizeStrictRaw('"' + "\uDC00" + '"'),
      "lone_surrogate"
    );
    expectStrictError(
      () => canonicalizeStrictRaw("\uD800"),
      "lone_surrogate"
    );
  });

  it("accepts surrogate pairs and emits the literal code point", () => {
    expect(canonicalHex('{"s":"\\uD83D\\uDE00"}')).toBe(
      Buffer.from('{"s":"\u{1F600}"}', "utf8").toString("hex")
    );
    expect(canonicalizeStrictRaw('{"s":"\u{1F600}"}')).toBe(
      '{"s":"\u{1F600}"}'
    );
  });

  it("rejects trailing data", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1} {"b":2}'),
      "trailing_data"
    );
  });

  it("rejects overflowing exponents as unrepresentable", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"n":1e999}'),
      "number_unrepresentable"
    );
  });

  it("enforces the plain-integer policy boundaries", () => {
    expect(canonicalizeStrictRaw("9007199254740991")).toBe(
      "9007199254740991"
    );
    expect(canonicalizeStrictRaw("-9007199254740991")).toBe(
      "-9007199254740991"
    );
    expectStrictError(
      () => canonicalizeStrictRaw("9007199254740992"),
      "number_out_of_policy"
    );
    expectStrictError(
      () => canonicalizeStrictRaw("-9007199254740992"),
      "number_out_of_policy"
    );
    expectStrictError(
      () => canonicalizeStrictRaw("9007199254740993"),
      "number_out_of_policy"
    );
  });

  it("orders NUL-containing and astral keys by UTF-16 code unit", () => {
    expect(canonicalizeStrictRaw('{"a\\u0000":1,"a":2}')).toBe(
      '{"a":2,"a\\u0000":1}'
    );
    expect(
      canonicalizeStrictRaw('{"z":1,"\\uFFFF":2,"\\uD83D\\uDE00":3}')
    ).toBe('{"z":1,"\u{1F600}":3,"\uFFFF":2}');
  });
});

describe("value mode", () => {
  it("serialises supported values recursively", () => {
    expect(canonicalizeStrict({ b: [null, true], a: "x" })).toBe(
      '{"a":"x","b":[null,true]}'
    );
  });

  it("rejects non-finite numbers", () => {
    expectStrictError(() => canonicalizeStrict(NaN), "non_finite_number");
    expectStrictError(() => canonicalizeStrict(Infinity), "non_finite_number");
    expectStrictError(() => canonicalizeStrict(-Infinity), "non_finite_number");
  });

  it("applies the safe-integer policy to bigint only", () => {
    expect(canonicalizeStrict(9007199254740991n)).toBe("9007199254740991");
    expectStrictError(
      () => canonicalizeStrict(9007199254740992n),
      "number_out_of_policy"
    );
    // Finite binary64 values are not subject to the plain-integer token policy.
    expect(canonicalizeStrict(1e20)).toBe("100000000000000000000");
  });

  it("rejects strings containing lone surrogates", () => {
    expectStrictError(() => canonicalizeStrict("\uD800"), "lone_surrogate");
  });

  it("rejects unsupported value types with TypeError", () => {
    expect(() => canonicalizeStrict(undefined)).toThrow(TypeError);
    expect(() => canonicalizeStrict(() => 0)).toThrow(TypeError);
    expect(() => canonicalizeStrict(Symbol("s"))).toThrow(TypeError);
  });
});

function nestedArrays(depth: number): unknown {
  let value: unknown = [];
  for (let i = 1; i < depth; i++) {
    value = [value];
  }
  return value;
}

function nestedObjects(depth: number): unknown {
  let value: unknown = {};
  for (let i = 1; i < depth; i++) {
    value = { a: value };
  }
  return value;
}

function rawNestedObjects(depth: number): string {
  return '{"a":'.repeat(depth - 1) + "{}" + "}".repeat(depth - 1);
}

describe("nesting depth", () => {
  it("accepts 512 raw nested arrays and rejects 513", () => {
    const accepted = "[".repeat(512) + "]".repeat(512);
    expect(canonicalizeStrictRaw(accepted)).toBe(accepted);
    expectStrictError(
      () => canonicalizeStrictRaw("[".repeat(513) + "]".repeat(513)),
      "depth_limit"
    );
  });

  it("accepts 512 raw nested objects and rejects 513", () => {
    const accepted = rawNestedObjects(512);
    expect(canonicalizeStrictRaw(accepted)).toBe(accepted);
    expectStrictError(
      () => canonicalizeStrictRaw(rawNestedObjects(513)),
      "depth_limit"
    );
  });

  it("rejects very deep raw nesting without a stack overflow", () => {
    expectStrictError(
      () => canonicalizeStrictRaw("[".repeat(5120) + "]".repeat(5120)),
      "depth_limit"
    );
  });

  it("ignores brackets inside strings during the depth scan", () => {
    expect(canonicalizeStrictRaw('{"s":"[[[{{{]]]}}}"}')).toBe(
      '{"s":"[[[{{{]]]}}}"}'
    );
  });

  it("accepts 512 value-mode nested arrays and rejects 513", () => {
    expect(canonicalizeStrict(nestedArrays(512))).toBe(
      "[".repeat(512) + "]".repeat(512)
    );
    expectStrictError(() => canonicalizeStrict(nestedArrays(513)), "depth_limit");
  });

  it("accepts 512 value-mode nested objects and rejects 513", () => {
    expect(canonicalizeStrict(nestedObjects(512))).toBe(
      '{"a":'.repeat(511) + "{}" + "}".repeat(511)
    );
    expectStrictError(
      () => canonicalizeStrict(nestedObjects(513)),
      "depth_limit"
    );
  });
});

describe("cyclic value safety", () => {
  it("rejects array cycles as invalid JSON", () => {
    const cyclic: unknown[] = [];
    cyclic.push(cyclic);
    expect(() => canonicalizeStrict(cyclic)).toThrow(
      StrictCanonicalizationError
    );
    try {
      canonicalizeStrict(cyclic);
    } catch (err) {
      expect((err as StrictCanonicalizationError).errorClass).toBe("invalid_json");
      expect((err as Error).message).toBe("cyclic value");
      return;
    }
    throw new Error("expected cyclic array rejection");
  });

  it("rejects object cycles as invalid JSON", () => {
    const cyclic: Record<string, unknown> = {};
    cyclic.self = cyclic;
    expectStrictError(() => canonicalizeStrict(cyclic), "invalid_json");
  });

  it("rejects indirect cycles as invalid JSON", () => {
    const first: unknown[] = [];
    const second: unknown[] = [first];
    first.push(second);
    expectStrictError(() => canonicalizeStrict(first), "invalid_json");
  });

  it("accepts shared non-cyclic references", () => {
    const sharedObject = { a: 1 };
    expect(canonicalizeStrict([sharedObject, sharedObject])).toBe(
      '[{"a":1},{"a":1}]'
    );
    const sharedArray = [1];
    expect(canonicalizeStrict([sharedArray, sharedArray])).toBe("[[1],[1]]");
  });
});

describe("duplicate key precedence", () => {
  it("reports a value error before a duplicate key", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1,"a":1e999}'),
      "number_unrepresentable"
    );
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1,"a":}'),
      "invalid_json"
    );
  });

  it("reports a lone surrogate before a duplicate key", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1,"a":"\\uD800"}'),
      "lone_surrogate"
    );
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1,"\\uD800":2,"\\uD800":3}'),
      "lone_surrogate"
    );
  });

  it("still rejects a plain duplicate key", () => {
    expectStrictError(
      () => canonicalizeStrictRaw('{"a":1,"a":2}'),
      "duplicate_key"
    );
  });
});

describe("non-finite token boundaries", () => {
  it("rejects exact non-finite literals", () => {
    for (const input of [
      "NaN",
      "Infinity",
      "-Infinity",
      "[NaN]",
      '{"n":Infinity}',
    ]) {
      expectStrictError(
        () => canonicalizeStrictRaw(input),
        "non_finite_number"
      );
    }
  });

  it("rejects malformed non-finite tokens as invalid JSON", () => {
    for (const input of ["NaNx", "Infinityx", "-Infinityx", "NaN1"]) {
      expectStrictError(() => canonicalizeStrictRaw(input), "invalid_json");
    }
  });
});

describe("underflow", () => {
  it("canonicalises raw underflow to zero", () => {
    expect(canonicalizeStrictRaw('{"n":1e-999}')).toBe('{"n":0}');
    expect(canonicalizeStrictRaw('{"n":-1e-999}')).toBe('{"n":0}');
    expect(canonicalizeStrictRaw("1e-999")).toBe("0");
  });

  it("canonicalises value-mode underflow to zero", () => {
    expect(canonicalizeStrict(1e-999)).toBe("0");
  });
});
