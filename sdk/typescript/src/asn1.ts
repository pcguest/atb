/**
 * Minimal ASN.1 DER decoder for ECDSA-P256 signatures.
 *
 * KMS backends (AWS, GCP, Vault) emit ECDSA signatures in DER form (an
 * ASN.1 SEQUENCE of two INTEGERs r and s). WebCrypto's SubtleCrypto.verify
 * for {name: "ECDSA"} expects the IEEE P1363 form (raw r||s, each padded
 * to the field size — 32 bytes for P-256). This module converts between
 * the two representations.
 *
 * Scope: P-256 only (32-byte field). Sufficient for the algorithms we
 * support; do not generalise without adding tests for larger curves.
 */

const FIELD_SIZE_P256 = 32;

/**
 * Convert a DER-encoded ECDSA signature (ASN.1 SEQUENCE of two INTEGERs)
 * into IEEE P1363 form (raw r||s, each left-padded to 32 bytes for P-256).
 *
 * Handles both the canonical 32-byte INTEGER and the 33-byte form where a
 * leading 0x00 byte was prepended to keep the high bit clear (DER's rule
 * for unsigned integers stored in a signed encoding).
 *
 * @param der DER-encoded ECDSA-P256 signature bytes.
 * @returns IEEE P1363 signature bytes.
 * @throws Error when the DER structure is malformed or not P-256 sized.
 */
export function derToP1363(der: Uint8Array): Uint8Array {
  if (der.length < 8) {
    throw new Error(`asn1: DER signature too short (${der.length} bytes)`);
  }
  let offset = 0;
  if (der[offset++] !== 0x30) {
    throw new Error("asn1: expected SEQUENCE tag (0x30)");
  }
  const seqLen = readLength(der, offset);
  offset = seqLen.offset;
  if (offset + seqLen.length > der.length) {
    throw new Error("asn1: SEQUENCE length exceeds buffer");
  }

  if (der[offset++] !== 0x02) {
    throw new Error("asn1: expected INTEGER tag for r");
  }
  const rLen = readLength(der, offset);
  offset = rLen.offset;
  const r = stripDerIntegerPadding(der.subarray(offset, offset + rLen.length));
  offset += rLen.length;

  if (der[offset++] !== 0x02) {
    throw new Error("asn1: expected INTEGER tag for s");
  }
  const sLen = readLength(der, offset);
  offset = sLen.offset;
  const s = stripDerIntegerPadding(der.subarray(offset, offset + sLen.length));

  if (r.length > FIELD_SIZE_P256 || s.length > FIELD_SIZE_P256) {
    throw new Error(
      `asn1: integer exceeds P-256 field size (r=${r.length}, s=${s.length})`
    );
  }

  const out = new Uint8Array(FIELD_SIZE_P256 * 2);
  out.set(r, FIELD_SIZE_P256 - r.length);
  out.set(s, FIELD_SIZE_P256 * 2 - s.length);
  return out;
}

function readLength(
  buf: Uint8Array,
  offset: number
): { length: number; offset: number } {
  if (offset >= buf.length) {
    throw new Error("asn1: truncated length field");
  }
  const first = buf[offset];
  if ((first & 0x80) === 0) {
    return { length: first, offset: offset + 1 };
  }
  // Long-form length: low 7 bits encode the byte count of the length value.
  const numBytes = first & 0x7f;
  if (numBytes === 0 || numBytes > 4) {
    throw new Error(`asn1: unsupported long-form length (${numBytes} bytes)`);
  }
  if (offset + 1 + numBytes > buf.length) {
    throw new Error("asn1: truncated long-form length");
  }
  let length = 0;
  for (let i = 0; i < numBytes; i++) {
    length = (length << 8) | buf[offset + 1 + i];
  }
  return { length, offset: offset + 1 + numBytes };
}

function stripDerIntegerPadding(int: Uint8Array): Uint8Array {
  // DER encodes unsigned integers with a leading 0x00 byte when the high
  // bit of the most-significant byte would otherwise be set (which would
  // mark the value as negative in two's complement). Strip exactly one
  // such byte; preserve all other bytes verbatim.
  if (int.length > 1 && int[0] === 0x00 && (int[1] & 0x80) !== 0) {
    return int.subarray(1);
  }
  return int;
}
