/**
 * RFC 8785 JSON Canonicalization Scheme (JCS) implementation.
 *
 * Produces a deterministic, canonical UTF-8 string for any JSON value,
 * suitable for use in cryptographic hash computation.
 */

/**
 * @param value JSON-compatible value to canonicalise.
 * @returns RFC 8785 canonical JSON string.
 * @throws TypeError when `value` contains an unsupported type.
 * @throws RangeError when `value` contains a non-finite number.
 */
export function canonicalize(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return serializeNumber(value);
  if (typeof value === "string") return serializeString(value);
  if (Array.isArray(value)) {
    return "[" + value.map(canonicalize).join(",") + "]";
  }
  if (typeof value === "object") {
    const obj = value as Record<string, unknown>;
    // RFC 8785 §3.2.3 — sort keys by UTF-16 code unit sequence.
    const sortedKeys = Object.keys(obj).sort(utf16Compare);
    const pairs = sortedKeys.map(
      (k) => `${serializeString(k)}:${canonicalize(obj[k])}`
    );
    return "{" + pairs.join(",") + "}";
  }
  throw new TypeError(`canonicalize: unsupported type ${typeof value}`);
}

function serializeNumber(n: number): string {
  if (!isFinite(n)) throw new RangeError(`canonicalize: non-finite number ${n}`);
  // Use JavaScript's default number serialisation, which matches ES6.
  return String(n);
}

const ESCAPE_MAP: Record<string, string> = {
  '"': '\\"',
  "\\": "\\\\",
  "\b": "\\b",
  "\f": "\\f",
  "\n": "\\n",
  "\r": "\\r",
  "\t": "\\t",
};

function serializeString(s: string): string {
  let result = '"';
  for (const ch of s) {
    const code = ch.charCodeAt(0);
    if (ch in ESCAPE_MAP) {
      result += ESCAPE_MAP[ch];
    } else if (code < 0x20) {
      result += `\\u${code.toString(16).padStart(4, "0")}`;
    } else {
      result += ch;
    }
  }
  return result + '"';
}

function utf16Compare(a: string, b: string): number {
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const ca = a.charCodeAt(i) || -1;
    const cb = b.charCodeAt(i) || -1;
    if (ca !== cb) return ca - cb;
  }
  return 0;
}
