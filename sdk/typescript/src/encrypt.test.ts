import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import {
  EncryptError,
  MAGIC,
  VERSION,
  HEADER_SIZE,
  decryptRaw,
  encryptBundle,
  encryptRaw,
} from "./encrypt.js";

const EXPECTED_GO_VECTOR_HEX =
  "4154424501101112131415161718191a1b1c1d1e1f202122232425262728292a2b6281773a2bf9c95f7f0a51a057f7ea5a5584ed630df2312b4c45ed3c104553e2312f48ba948ba4790d75858ae40a518b4289c7cde8b775504d2e0727a6ee891a09d53d8b4a1467dc4a74c9e7a8012c270b3c4107621caa8eea6ec779bd31073c1b336c3da5f0e5d4a34770dc5109cfb93a4d90d9cfd6307b6a487089c69d5f50c1714bfc26b3107f8540ac31e366fe389e77ad8c5fe78aea46dedd7d0c2d235053a62625e67c094075055cdb077a9f81f22e8dd5d0f01e927a8307032ebe0df5139ec5e319eac1cfefbd3ac3f8a76c9cd19c92278f984f2804da2a660a23c0a15115b939fcd095ebac3b2662e50f2adb683a842a66a8557e4ef26beb07600a35ef6f86d2768f2f0ceee0b465a5e13fbb14a26c74c0df04cf069209815c502fe9f92b628ecf104e719e9b931bd196989952f87399af49446b7ac0a188338c46f05a781b4a30cfdf59a7b5537f53d2263b4ae7192398ff920162e4c071214138e83ddae16a3b3dd5ee35e1593c4d8c3ce4cf591f95c5a9f81a4d43bfcf78c4e54a8398cd57e2741e14a1c97da69b09160224026745b66e312e80900e6205b06cd72aa6026b2b85b734452e73da56f2c6f99b62d9b98e19c1c87d868fdf678cef5119eae4d324234214cba7726f1981af65aebe36f37542ab8d0f94007a1765a12116bc9abfb7b833fe89333da6bf268126b2f2689b7561b2ba0d7b174bf089b7d49612ded3cb0e4ea30166331382370b31d6f5381c490871035e5f632ce6";

function fixedSalt(): Uint8Array {
  return Uint8Array.from([
    0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b,
    0x1c, 0x1d, 0x1e, 0x1f,
  ]);
}

function fixedNonce(): Uint8Array {
  return Uint8Array.from([
    0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b,
  ]);
}

function sampleBundle(): Bundle {
  const bundle = new Bundle();
  bundle.append("dev.session", { msg: "hello" });
  bundle.append("decision", { choice: "ship" });
  return bundle;
}

describe("encrypt", () => {
  it("encrypt/decrypt roundtrip with fixed salt/nonce", async () => {
    const bundle = sampleBundle();
    const first = await encryptBundle(bundle.records, "test123", {
      salt: fixedSalt(),
      nonce: fixedNonce(),
    });
    const second = await encryptBundle(bundle.records, "test123", {
      salt: fixedSalt(),
      nonce: fixedNonce(),
    });
    expect(Buffer.from(first).equals(Buffer.from(second))).toBe(true);
    expect(Buffer.from(first.subarray(0, MAGIC.length)).toString("ascii")).toBe(
      MAGIC
    );
    expect(first[MAGIC.length]).toBe(VERSION);
    expect(first.length).toBeGreaterThan(HEADER_SIZE);

    const decrypted = await Bundle.decrypt("test123", first);
    expect(decrypted.records.map((r) => r.hash)).toEqual(
      bundle.records.map((r) => r.hash)
    );
  });

  it("decrypt fails with wrong password", () => {
    const encrypted = encryptRaw(
      Buffer.from('{"head_hash":"abc","records":[]}', "utf8"),
      "test123",
      { salt: fixedSalt(), nonce: fixedNonce() }
    );
    expect(() => decryptRaw(encrypted, "wrong-pass")).toThrow(EncryptError);
    expect(() => decryptRaw(encrypted, "wrong-pass")).toThrow(
      "authentication failed"
    );
  });

  it("decrypt fails with tampered ciphertext", () => {
    const encrypted = Buffer.from(
      encryptRaw(Buffer.from('{"head_hash":"abc","records":[]}', "utf8"), "test123", {
        salt: fixedSalt(),
        nonce: fixedNonce(),
      })
    );
    encrypted[encrypted.length - 1] ^= 0x01;
    expect(() => decryptRaw(encrypted, "test123")).toThrow("authentication failed");
  });

  it("decrypt rejects unsupported version", () => {
    const encrypted = Buffer.from(
      encryptRaw(Buffer.from('{"head_hash":"abc","records":[]}', "utf8"), "test123", {
        salt: fixedSalt(),
        nonce: fixedNonce(),
      })
    );
    encrypted[MAGIC.length] = 0x02;
    expect(() => decryptRaw(encrypted, "test123")).toThrow("unsupported version");
  });

  it("decrypt then verify head hash mismatch", async () => {
    const bundle = sampleBundle();
    const encrypted = await encryptBundle(bundle.records, "test123", {
      salt: fixedSalt(),
      nonce: fixedNonce(),
    });
    const decrypted = JSON.parse(
      Buffer.from(decryptRaw(encrypted, "test123")).toString("utf8")
    ) as { head_hash: string; records: unknown[] };
    const tamperedPayload = {
      head_hash: "0".repeat(64),
      records: decrypted.records,
    };
    const tampered = encryptRaw(
      Buffer.from(canonicalize(tamperedPayload), "utf8"),
      "test123",
      { salt: fixedSalt(), nonce: fixedNonce() }
    );
    await expect(Bundle.decrypt("test123", tampered)).rejects.toThrow(
      "head_hash mismatch"
    );
  });

  it("bundle methods roundtrip", async () => {
    const bundle = sampleBundle();
    const encrypted = await bundle.encrypt("test123");
    const decrypted = await Bundle.decrypt("test123", encrypted);
    expect(decrypted.records.map((r) => r.hash)).toEqual(
      bundle.records.map((r) => r.hash)
    );
  });

  it("parity with Go/Python fixed vector", async () => {
    const bundle = sampleBundle();
    const encrypted = await encryptBundle(bundle.records, "test123", {
      salt: fixedSalt(),
      nonce: fixedNonce(),
    });
    expect(Buffer.from(encrypted).toString("hex")).toBe(EXPECTED_GO_VECTOR_HEX);
  });
});
