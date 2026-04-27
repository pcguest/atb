import { describe, expect, it } from "vitest";
import { derToP1363 } from "./asn1.js";

describe("derToP1363", () => {
  it("decodes a canonical 32-byte r and 32-byte s", () => {
    // SEQUENCE(0x30) len(0x44=68)
    //   INTEGER(0x02) len(0x20=32) r
    //   INTEGER(0x02) len(0x20=32) s
    const r = new Uint8Array(32).fill(0x11);
    const s = new Uint8Array(32).fill(0x22);
    const der = new Uint8Array([
      0x30,
      0x44,
      0x02,
      0x20,
      ...r,
      0x02,
      0x20,
      ...s,
    ]);
    const out = derToP1363(der);
    expect(out.length).toBe(64);
    expect(Array.from(out.slice(0, 32))).toEqual(Array.from(r));
    expect(Array.from(out.slice(32))).toEqual(Array.from(s));
  });

  it("strips a leading 0x00 padding byte from a 33-byte DER INTEGER", () => {
    // r is 33 bytes with high-bit-set value preceded by 0x00 sign byte;
    // s is a normal 32-byte value.
    const rPadded = new Uint8Array(33);
    rPadded[0] = 0x00;
    rPadded[1] = 0x80; // sets the high bit, so the 0x00 is required in DER
    rPadded.fill(0xab, 2);
    const s = new Uint8Array(32).fill(0x33);
    const der = new Uint8Array([
      0x30,
      33 + 32 + 4, // SEQUENCE length: rTagLen(2)+r(33)+sTagLen(2)+s(32)
      0x02,
      0x21,
      ...rPadded,
      0x02,
      0x20,
      ...s,
    ]);
    const out = derToP1363(der);
    expect(out.length).toBe(64);
    // Expected r: rPadded with leading 0x00 stripped -> 32 bytes starting with 0x80.
    const expectedR = rPadded.slice(1);
    expect(Array.from(out.slice(0, 32))).toEqual(Array.from(expectedR));
    expect(Array.from(out.slice(32))).toEqual(Array.from(s));
  });

  it("left-pads short INTEGERs to 32 bytes", () => {
    // r is only 1 byte (small value 0x07), s is 30 bytes.
    const rShort = new Uint8Array([0x07]);
    const sShort = new Uint8Array(30).fill(0x55);
    const der = new Uint8Array([
      0x30,
      1 + 30 + 4,
      0x02,
      0x01,
      ...rShort,
      0x02,
      0x1e,
      ...sShort,
    ]);
    const out = derToP1363(der);
    expect(out.length).toBe(64);
    // r should be 31 zero bytes followed by 0x07.
    expect(Array.from(out.slice(0, 31))).toEqual(new Array(31).fill(0));
    expect(out[31]).toBe(0x07);
    // s should be 2 zero bytes followed by 30 x 0x55.
    expect(out[32]).toBe(0);
    expect(out[33]).toBe(0);
    expect(Array.from(out.slice(34))).toEqual(Array.from(sShort));
  });

  it("rejects malformed input that is not a SEQUENCE", () => {
    const bogus = new Uint8Array([0x02, 0x01, 0x01, 0x02, 0x01, 0x01, 0, 0]);
    expect(() => derToP1363(bogus)).toThrow();
  });
});
