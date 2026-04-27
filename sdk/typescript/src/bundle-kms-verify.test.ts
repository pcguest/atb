import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { afterEach, beforeAll, describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";

const TMP_DIRS: string[] = [];

afterEach(() => {
  while (TMP_DIRS.length) {
    const d = TMP_DIRS.pop();
    if (d) rmSync(d, { recursive: true, force: true });
  }
});

function mkTmp(): string {
  const d = mkdtempSync(join(tmpdir(), "atb-kms-verify-"));
  TMP_DIRS.push(d);
  return d;
}

async function makeKmsSignedBundle(opts: {
  algorithm: string;
  flipPayloadByte?: boolean;
}): Promise<{ path: string; bundleBytes: Buffer; pubKey: Uint8Array }> {
  const dir = mkTmp();
  const path = join(dir, "bundle.atb");

  // Build a small bundle the normal way.
  const b = new Bundle({ path });
  b.append("ai.tool.exec", { ok: true });
  b.save();

  // Read raw bytes — these are the pre-signature NDJSON the KMS would sign.
  let raw = readFileSync(path);

  // Generate an ephemeral P-256 keypair and sign the raw bytes.
  const subtle = globalThis.crypto.subtle;
  const kp = await subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    true,
    ["sign", "verify"]
  );
  const spki = new Uint8Array(await subtle.exportKey("spki", kp.publicKey));

  // SubtleCrypto.sign returns IEEE P1363 (raw r||s); convert to DER for the
  // wire format expected by the verifier.
  const p1363 = new Uint8Array(
    await subtle.sign(
      { name: "ECDSA", hash: "SHA-256" },
      kp.privateKey,
      raw
    )
  );
  const der = p1363ToDer(p1363);

  if (opts.flipPayloadByte) {
    // Flip a byte in the payload AFTER signing so verification must fail.
    raw = Buffer.from(raw);
    // Find the first '"' inside the data field and flip a byte nearby.
    const idx = raw.indexOf("ok");
    raw[idx] ^= 0x01;
    writeFileSync(path, raw);
  }

  const digest = await subtle.digest("SHA-256", new Uint8Array(raw));
  const bundleHash = Buffer.from(new Uint8Array(digest)).toString("hex");

  const sigRecord = {
    event: {
      seq: 99, // not validated by signaturesAsync's signature check
      prev_hash: "0".repeat(64),
      type: "atb.bundle.signature",
      hash_algo: "sha256",
      timestamp: new Date().toISOString(),
      data: {
        bundle_hash: bundleHash,
        signature: Buffer.from(der).toString("base64"),
        pubkey: Buffer.from(spki).toString("base64"),
        algorithm: opts.algorithm,
        backend: "aws-kms",
        key_id: "arn:aws:kms:us-east-1:111122223333:key/abcd-1234",
        signed_at: new Date().toISOString(),
      },
    },
    hash: "0".repeat(64), // not relied on by signaturesAsync
  };

  // Append the signature record to the file so the loader picks it up.
  const newline = raw.length > 0 && raw[raw.length - 1] !== 0x0a ? "\n" : "";
  const appended = Buffer.concat([
    raw,
    Buffer.from(newline + JSON.stringify(sigRecord) + "\n", "utf8"),
  ]);
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, appended);

  return { path, bundleBytes: appended, pubKey: spki };
}

// Minimal P1363 → DER for test fixtures only. Adds the leading 0x00 padding
// when r or s have the high bit set.
function p1363ToDer(p1363: Uint8Array): Uint8Array {
  const r = p1363.slice(0, 32);
  const s = p1363.slice(32);
  const rEnc = encodeUnsignedInt(r);
  const sEnc = encodeUnsignedInt(s);
  const seqLen = rEnc.length + sEnc.length;
  const out = new Uint8Array(2 + seqLen);
  out[0] = 0x30;
  out[1] = seqLen;
  out.set(rEnc, 2);
  out.set(sEnc, 2 + rEnc.length);
  return out;
}

function encodeUnsignedInt(value: Uint8Array): Uint8Array {
  // Strip leading zero bytes...
  let start = 0;
  while (start < value.length - 1 && value[start] === 0) start++;
  let trimmed = value.subarray(start);
  // ...then re-add a single 0x00 if the high bit is set so the value stays
  // unambiguously positive when re-encoded.
  if (trimmed[0] & 0x80) {
    const padded = new Uint8Array(trimmed.length + 1);
    padded[0] = 0x00;
    padded.set(trimmed, 1);
    trimmed = padded;
  }
  return new Uint8Array([0x02, trimmed.length, ...trimmed]);
}

describe("Bundle.signaturesAsync", () => {
  beforeAll(() => {
    if (!globalThis.crypto?.subtle) {
      throw new Error("WebCrypto not available — Node >= 18 required");
    }
  });

  it("verifies an ECDSA-P256 KMS-signed bundle", async () => {
    const { path } = await makeKmsSignedBundle({ algorithm: "ecdsa-p256" });
    const loaded = Bundle.load(path);
    const sigs = await loaded.signaturesAsync();
    expect(sigs.length).toBeGreaterThan(0);
    const kms = sigs.find((s) => s.algorithm === "ecdsa-p256");
    expect(kms).toBeDefined();
    expect(kms!.valid).toBe(true);
    expect(kms!.backend).toBe("aws-kms");
    expect(kms!.key_id).toContain("arn:aws:kms:");
    expect(kms!.signed_at).toBeTruthy();
  });

  it("returns valid:false for an ECDSA-P256 signature when the payload is tampered", async () => {
    const { path } = await makeKmsSignedBundle({
      algorithm: "ecdsa-p256",
      flipPayloadByte: true,
    });
    const loaded = Bundle.load(path);
    const sigs = await loaded.signaturesAsync();
    const kms = sigs.find((s) => s.algorithm === "ecdsa-p256");
    expect(kms).toBeDefined();
    expect(kms!.valid).toBe(false);
  });

  it("throws on an unsupported algorithm rather than silently returning false", async () => {
    const { path } = await makeKmsSignedBundle({ algorithm: "rsa-pss-sha512" });
    const loaded = Bundle.load(path);
    await expect(loaded.signaturesAsync()).rejects.toThrow(
      /unsupported signature algorithm/i
    );
  });

  it("verifies a locally-signed Ed25519 bundle (regression)", async () => {
    const dir = mkTmp();
    const path = join(dir, "bundle.atb");
    const b = new Bundle({ path });
    b.append("ai.tool.exec", { ok: true });
    b.save();

    const { generateKeyPairSync } = await import("node:crypto");
    const { privateKey } = generateKeyPairSync("ed25519");
    const pem = privateKey.export({ format: "pem", type: "pkcs8" }).toString();
    b.signLocal(pem);

    const loaded = Bundle.load(path);
    const sigs = await loaded.signaturesAsync();
    expect(sigs.length).toBe(1);
    expect(sigs[0].algorithm).toBe("ed25519");
    expect(sigs[0].valid).toBe(true);
  });

  it("surfaces provenance fields on the parsed signature evidence", async () => {
    const { path } = await makeKmsSignedBundle({ algorithm: "ecdsa-p256" });
    const loaded = Bundle.load(path);
    const sigs = await loaded.signaturesAsync();
    const ev = sigs.find((s) => s.algorithm === "ecdsa-p256")!;
    expect(ev.algorithm).toBe("ecdsa-p256");
    expect(ev.backend).toBe("aws-kms");
    expect(ev.key_id).toBe(
      "arn:aws:kms:us-east-1:111122223333:key/abcd-1234"
    );
    expect(typeof ev.signed_at).toBe("string");
    expect(ev.signed_at.length).toBeGreaterThan(0);
    expect(typeof ev.bundle_hash).toBe("string");
    expect(ev.bundle_hash).toMatch(/^[0-9a-f]{64}$/);
  });
});
