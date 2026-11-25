/**
 * Strict `atb-jcs-rfc8785-1` JSON canonicalisation.
 *
 * This is a new, non-production entry point. It implements RFC 8785
 * canonicalisation plus the ATB safe-integer interoperability restriction.
 * The legacy `canonicalize` function is intentionally unchanged.
 */

const MAX_SAFE_INTEGER_BIGINT = 9007199254740991n;
const MIN_SAFE_INTEGER_BIGINT = -9007199254740991n;

/** Maximum array/object nesting accepted by the strict profile. */
const MAX_NESTING_DEPTH = 512;

/** Stable strict-profile error classes. */
export class StrictCanonicalizationError extends Error {
  readonly errorClass: string;

  constructor(errorClass: string, message?: string) {
    super(message ?? errorClass);
    this.name = "StrictCanonicalizationError";
    this.errorClass = errorClass;
  }
}

/**
 * Canonicalise a single raw JSON value.
 *
 * @param input UTF-8 bytes or an already-decoded JavaScript string.
 * @returns the strict canonical JSON string.
 * @throws StrictCanonicalizationError when the input is rejected.
 */
export function canonicalizeStrictRaw(input: string | Uint8Array): string {
  const text = typeof input === "string" ? input : decodeUtf8(input);
  // Decoding valid UTF-8 bytes can never yield a lone surrogate, but a caller
  // may pass a JavaScript string that contains one directly. Reject it before
  // parsing, mirroring the Go byte path's up-front UTF-8 validity check.
  if (hasLoneSurrogate(text)) {
    throw new StrictCanonicalizationError(
      "lone_surrogate",
      "input contains an unpaired UTF-16 surrogate"
    );
  }
  const parser = new StrictParser(text);
  const output = parser.parseValue();
  parser.skipWhitespace();
  if (!parser.atEnd()) {
    throw new StrictCanonicalizationError(
      "trailing_data",
      "non-whitespace data follows the top-level JSON value"
    );
  }
  return output;
}

/**
 * Canonicalise an in-memory JavaScript value.
 *
 * @param value the value to canonicalise.
 * @returns the strict canonical JSON string.
 * @throws StrictCanonicalizationError for non-finite numbers, out-of-policy
 *   bigints, or strings containing lone surrogates.
 * @throws TypeError for unsupported value types.
 */
export function canonicalizeStrict(value: unknown): string {
  return serializeValue(value, 0, new WeakSet<object>());
}

function serializeValue(
  value: unknown,
  depth: number,
  seen: WeakSet<object>
): string {
  if (value === null) return "null";
  switch (typeof value) {
    case "boolean":
      return value ? "true" : "false";
    case "number":
      return serializeNumber(value);
    case "bigint":
      return serializeBigInt(value);
    case "string":
      return serializeString(value);
    case "object":
      return serializeObjectOrArray(value, depth, seen);
    default:
      throw new TypeError(`canonicalizeStrict: unsupported type ${typeof value}`);
  }
}

function decodeUtf8(input: Uint8Array): string {
  try {
    // `ignoreBOM: true` keeps a leading BOM in the decoded text rather than
    // silently stripping it, so it is rejected as invalid JSON just as the Go
    // implementation rejects the U+FEFF code point.
    return new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(
      input
    );
  } catch {
    throw new StrictCanonicalizationError(
      "invalid_utf8",
      "input is not valid UTF-8"
    );
  }
}

function hasLoneSurrogate(s: string): boolean {
  for (let i = 0; i < s.length; i++) {
    const code = s.charCodeAt(i);
    if (code >= 0xd800 && code <= 0xdbff) {
      const next = i + 1 < s.length ? s.charCodeAt(i + 1) : -1;
      if (next < 0xdc00 || next > 0xdfff) return true;
      i++;
    } else if (code >= 0xdc00 && code <= 0xdfff) {
      return true;
    }
  }
  return false;
}

function serializeNumber(n: number): string {
  if (!Number.isFinite(n)) {
    throw new StrictCanonicalizationError(
      "non_finite_number",
      "value contains a non-finite number"
    );
  }
  // ES6 Number::toString; this already serialises both zeros as "0".
  return String(n);
}

function serializeBigInt(n: bigint): string {
  if (n > MAX_SAFE_INTEGER_BIGINT || n < MIN_SAFE_INTEGER_BIGINT) {
    throw new StrictCanonicalizationError(
      "number_out_of_policy",
      "integer value exceeds 2^53 - 1"
    );
  }
  return String(Number(n));
}

function serializeObjectOrArray(
  value: object,
  depth: number,
  seen: WeakSet<object>
): string {
  const isArray = Array.isArray(value);
  if (!isArray && !isPlainObject(value)) {
    throw new TypeError("canonicalizeStrict: unsupported object type");
  }

  const nextDepth = depth + 1;
  if (nextDepth > MAX_NESTING_DEPTH) {
    throw new StrictCanonicalizationError(
      "depth_limit",
      "maximum nesting depth exceeded"
    );
  }
  if (seen.has(value)) {
    throw new StrictCanonicalizationError("invalid_json", "cyclic value");
  }
  seen.add(value);

  try {
    if (isArray) {
      const parts: string[] = [];
      for (const item of value as unknown[]) {
        parts.push(serializeValue(item, nextDepth, seen));
      }
      return "[" + parts.join(",") + "]";
    }

    const obj = value as Record<string, unknown>;
    const keys = Object.keys(obj).sort(utf16Compare);
    const parts = keys.map(
      (key) =>
        `${serializeString(key)}:${serializeValue(obj[key], nextDepth, seen)}`
    );
    return "{" + parts.join(",") + "}";
  } finally {
    seen.delete(value);
  }
}

function isPlainObject(value: object): boolean {
  const proto = Object.getPrototypeOf(value);
  return proto === Object.prototype || proto === null;
}

const ESCAPE_MAP: Record<number, string> = {
  0x22: '\\"',
  0x5c: "\\\\",
  0x08: "\\b",
  0x0c: "\\f",
  0x0a: "\\n",
  0x0d: "\\r",
  0x09: "\\t",
};

function serializeString(s: string): string {
  let result = '"';
  for (let i = 0; i < s.length; i++) {
    const code = s.charCodeAt(i);

    if (code >= 0xd800 && code <= 0xdbff) {
      const next = i + 1 < s.length ? s.charCodeAt(i + 1) : -1;
      if (next < 0xdc00 || next > 0xdfff) {
        throw new StrictCanonicalizationError(
          "lone_surrogate",
          "unpaired UTF-16 surrogate"
        );
      }
      result += s[i] + s[i + 1];
      i++;
      continue;
    }
    if (code >= 0xdc00 && code <= 0xdfff) {
      throw new StrictCanonicalizationError(
        "lone_surrogate",
        "unpaired UTF-16 surrogate"
      );
    }

    const escape = ESCAPE_MAP[code];
    if (escape !== undefined) {
      result += escape;
    } else if (code < 0x20) {
      result += `\\u${code.toString(16).padStart(4, "0")}`;
    } else {
      result += s[i];
    }
  }
  return result + '"';
}

/**
 * Compare strings by their UTF-16 code-unit sequences. A NUL code unit is
 * compared as 0 (not treated as a terminator), and a proper prefix sorts first.
 */
function utf16Compare(a: string, b: string): number {
  const shared = Math.min(a.length, b.length);
  for (let i = 0; i < shared; i++) {
    const ca = a.charCodeAt(i);
    const cb = b.charCodeAt(i);
    if (ca !== cb) return ca - cb;
  }
  return a.length - b.length;
}

function isDigit(code: number): boolean {
  return code >= 0x30 && code <= 0x39;
}

class StrictParser {
  private pos = 0;
  private depth = 0;

  constructor(private readonly data: string) {}

  atEnd(): boolean {
    return this.pos >= this.data.length;
  }

  skipWhitespace(): void {
    while (this.pos < this.data.length) {
      const c = this.data.charCodeAt(this.pos);
      if (c === 0x20 || c === 0x09 || c === 0x0a || c === 0x0d) {
        this.pos++;
      } else {
        return;
      }
    }
  }

  parseValue(): string {
    this.skipWhitespace();
    if (this.pos >= this.data.length) {
      this.fail("invalid_json", "unexpected end of input");
    }

    const c = this.data.charCodeAt(this.pos);
    if (c === 0x6e) {
      if (this.consumeLiteral("null")) return "null";
    } else if (c === 0x74) {
      if (this.consumeLiteral("true")) return "true";
    } else if (c === 0x66) {
      if (this.consumeLiteral("false")) return "false";
    } else if (c === 0x22) {
      return serializeString(this.parseString());
    } else if (c === 0x5b) {
      return this.parseArray();
    } else if (c === 0x7b) {
      return this.parseObject();
    }

    if (
      (this.hasPrefix("NaN") && this.literalBoundary("NaN".length)) ||
      (this.hasPrefix("Infinity") &&
        this.literalBoundary("Infinity".length)) ||
      (this.hasPrefix("-Infinity") &&
        this.literalBoundary("-Infinity".length))
    ) {
      this.fail("non_finite_number", "non-finite number literal");
    }
    if (c === 0x2d || isDigit(c)) {
      return this.parseNumber();
    }
    this.fail("invalid_json", "unexpected token");
  }

  /**
   * Report whether a literal of length `n` starting at the current position
   * ends at a valid JSON token boundary. This keeps `NaNx`, `Infinityx` and
   * similar malformed tokens classified as invalid JSON rather than as
   * non-finite number literals.
   */
  private literalBoundary(n: number): boolean {
    const index = this.pos + n;
    if (index >= this.data.length) return true;
    const c = this.data.charCodeAt(index);
    return (
      c === 0x20 ||
      c === 0x09 ||
      c === 0x0a ||
      c === 0x0d ||
      c === 0x2c ||
      c === 0x5d ||
      c === 0x7d
    );
  }

  private enterContainer(): void {
    this.depth += 1;
    if (this.depth > MAX_NESTING_DEPTH) {
      this.fail("depth_limit", "maximum nesting depth exceeded");
    }
  }

  private leaveContainer(): void {
    this.depth -= 1;
  }

  private parseArray(): string {
    this.enterContainer();
    try {
      this.pos++; // '['
      this.skipWhitespace();
      if (this.peek() === 0x5d) {
        this.pos++;
        return "[]";
      }

      let output = "[";
      for (let i = 0; ; i++) {
        if (i > 0) output += ",";
        output += this.parseValue();
        this.skipWhitespace();

        const c = this.peek();
        if (c === 0x2c) {
          this.pos++;
          continue;
        }
        if (c === 0x5d) {
          this.pos++;
          return output + "]";
        }
        this.fail("invalid_json", "expected ',' or ']'");
      }
    } finally {
      this.leaveContainer();
    }
  }

  private parseObject(): string {
    this.enterContainer();
    try {
      this.pos++; // '{'
      this.skipWhitespace();
      if (this.peek() === 0x7d) {
        this.pos++;
        return "{}";
      }

      const members: Array<{ key: string; value: string }> = [];
      const seen = new Set<string>();

      for (;;) {
        this.skipWhitespace();
        if (this.peek() !== 0x22) {
          this.fail("invalid_json", "expected object member name");
        }

        const key = this.parseString();

        this.skipWhitespace();
        if (this.peek() !== 0x3a) {
          this.fail("invalid_json", "expected ':' after object member name");
        }
        this.pos++;

        const value = this.parseValue();
        // Duplicate detection runs after the member value has been parsed so
        // that value errors (malformed JSON, unrepresentable numbers, lone
        // surrogates) take precedence, matching Go and Python.
        if (seen.has(key)) {
          this.fail("duplicate_key", "duplicate object member name");
        }
        seen.add(key);
        members.push({ key, value });

        this.skipWhitespace();
        const c = this.peek();
        if (c === 0x2c) {
          this.pos++;
          continue;
        }
        if (c === 0x7d) {
          this.pos++;
          return this.writeObject(members);
        }
        this.fail("invalid_json", "expected ',' or '}'");
      }
    } finally {
      this.leaveContainer();
    }
  }

  private writeObject(members: Array<{ key: string; value: string }>): string {
    members.sort((a, b) => utf16Compare(a.key, b.key));
    let output = "{";
    for (let i = 0; i < members.length; i++) {
      if (i > 0) output += ",";
      output += `${serializeString(members[i].key)}:${members[i].value}`;
    }
    return output + "}";
  }

  private parseString(): string {
    this.pos++; // opening '"'
    let output = "";

    while (this.pos < this.data.length) {
      const c = this.data.charCodeAt(this.pos);

      if (c === 0x22) {
        this.pos++;
        return output;
      }

      if (c === 0x5c) {
        this.pos++;
        if (this.pos >= this.data.length) {
          this.fail("invalid_json", "truncated escape sequence");
        }
        const esc = this.data.charCodeAt(this.pos);
        this.pos++;
        switch (esc) {
          case 0x22:
            output += '"';
            break;
          case 0x5c:
            output += "\\";
            break;
          case 0x2f:
            output += "/";
            break;
          case 0x62:
            output += "\b";
            break;
          case 0x66:
            output += "\f";
            break;
          case 0x6e:
            output += "\n";
            break;
          case 0x72:
            output += "\r";
            break;
          case 0x74:
            output += "\t";
            break;
          case 0x75:
            output += this.parseUnicodeEscape();
            break;
          default:
            this.fail("invalid_json", "invalid escape sequence");
        }
        continue;
      }

      if (c < 0x20) {
        this.fail("invalid_json", "unescaped control character in string");
      }

      if (c >= 0xd800 && c <= 0xdbff) {
        const next =
          this.pos + 1 < this.data.length
            ? this.data.charCodeAt(this.pos + 1)
            : -1;
        if (next < 0xdc00 || next > 0xdfff) {
          this.fail("lone_surrogate", "unpaired UTF-16 surrogate");
        }
        output += this.data[this.pos] + this.data[this.pos + 1];
        this.pos += 2;
        continue;
      }

      if (c >= 0xdc00 && c <= 0xdfff) {
        this.fail("lone_surrogate", "unpaired UTF-16 surrogate");
      }

      output += this.data[this.pos];
      this.pos++;
    }

    this.fail("invalid_json", "unterminated string");
  }

  /**
   * Consume the four hex digits of a `\uXXXX` escape (the leading `\u` has
   * already been consumed). A high surrogate must be followed by a low
   * surrogate escape; a low surrogate is never valid on its own.
   */
  private parseUnicodeEscape(): string {
    const first = this.readHex4();
    if (first < 0xd800 || first > 0xdfff) {
      return String.fromCharCode(first);
    }

    if (
      !(
        this.pos + 1 < this.data.length &&
        this.data.charCodeAt(this.pos) === 0x5c &&
        this.data.charCodeAt(this.pos + 1) === 0x75
      )
    ) {
      this.fail("lone_surrogate", "unpaired UTF-16 surrogate escape");
    }

    this.pos += 2;
    const second = this.readHex4();
    if (
      first < 0xd800 ||
      first > 0xdbff ||
      second < 0xdc00 ||
      second > 0xdfff
    ) {
      this.fail("lone_surrogate", "unpaired UTF-16 surrogate escape");
    }

    const codePoint = 0x10000 + ((first - 0xd800) << 10) + (second - 0xdc00);
    return String.fromCodePoint(codePoint);
  }

  private readHex4(): number {
    if (this.pos + 4 > this.data.length) {
      this.fail("invalid_json", "truncated \\u escape");
    }

    let value = 0;
    for (let i = 0; i < 4; i++) {
      const c = this.data.charCodeAt(this.pos + i);
      let digit = 0;
      if (c >= 0x30 && c <= 0x39) {
        digit = c - 0x30;
      } else if (c >= 0x61 && c <= 0x66) {
        digit = c - 0x61 + 10;
      } else if (c >= 0x41 && c <= 0x46) {
        digit = c - 0x41 + 10;
      } else {
        this.fail("invalid_json", "invalid \\u escape");
      }
      value = (value << 4) | digit;
    }
    this.pos += 4;
    return value & 0xffff;
  }

  private parseNumber(): string {
    const start = this.pos;

    if (this.peek() === 0x2d) {
      this.pos++;
    }

    if (this.pos >= this.data.length) {
      this.fail("invalid_json", "truncated number");
    }

    const firstDigit = this.data.charCodeAt(this.pos);
    if (firstDigit === 0x30) {
      this.pos++;
      if (isDigit(this.peek())) {
        this.fail("invalid_json", "leading zero in number");
      }
    } else if (firstDigit >= 0x31 && firstDigit <= 0x39) {
      this.pos++;
      while (isDigit(this.peek())) this.pos++;
    } else {
      this.fail("invalid_json", "invalid number");
    }

    let plainInteger = true;

    if (this.peek() === 0x2e) {
      plainInteger = false;
      this.pos++;
      if (!isDigit(this.peek())) {
        this.fail("invalid_json", "missing fraction digits");
      }
      while (isDigit(this.peek())) this.pos++;
    }

    const exponent = this.peek();
    if (exponent === 0x65 || exponent === 0x45) {
      plainInteger = false;
      this.pos++;
      const sign = this.peek();
      if (sign === 0x2b || sign === 0x2d) this.pos++;
      if (!isDigit(this.peek())) {
        this.fail("invalid_json", "missing exponent digits");
      }
      while (isDigit(this.peek())) this.pos++;
    }

    const token = this.data.slice(start, this.pos);

    if (plainInteger) {
      let magnitude = 0n;
      try {
        magnitude = BigInt(token);
      } catch {
        this.fail("invalid_json", "invalid integer literal");
      }
      if (
        magnitude > MAX_SAFE_INTEGER_BIGINT ||
        magnitude < MIN_SAFE_INTEGER_BIGINT
      ) {
        this.fail(
          "number_out_of_policy",
          "plain integer literal exceeds 2^53 - 1"
        );
      }
      // Within the safe-integer range the conversion is exact.
      return String(Number(token));
    }

    const n = Number(token);
    if (!Number.isFinite(n)) {
      this.fail(
        "number_unrepresentable",
        "number cannot be represented as a finite binary64 value"
      );
    }
    return String(n);
  }

  private hasPrefix(prefix: string): boolean {
    return this.data.startsWith(prefix, this.pos);
  }

  private consumeLiteral(literal: string): boolean {
    if (this.hasPrefix(literal)) {
      this.pos += literal.length;
      return true;
    }
    return false;
  }

  private peek(): number {
    return this.pos < this.data.length ? this.data.charCodeAt(this.pos) : -1;
  }

  private fail(errorClass: string, message: string): never {
    throw new StrictCanonicalizationError(errorClass, message);
  }
}
