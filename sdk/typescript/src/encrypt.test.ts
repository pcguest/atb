import { createCipheriv, pbkdf2Sync } from "node:crypto";
import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import type { Event } from "./event.js";
import {
  EncryptError,
  MAGIC,
  KEY_SIZE,
  LEGACY_PBKDF2_ITERATIONS,
  LEGACY_VERSION,
  VERSION,
  HEADER_SIZE,
  decryptRaw,
  encryptBundle,
  encryptRaw,
} from "./encrypt.js";
import { GENESIS_HASH, computeHash } from "./hash.js";

const EXPECTED_GO_VECTOR_HEX =
  "4154424502101112131415161718191a1b1c1d1e1f202122232425262728292a2bf688fd757a322898adb2343fb39d4a7ddda25540f2da650d5b271d17f6cbd9f6ac74cdf11fccaf54d4604971572033830fd35fa1b3fd443a50108981b5fd452fc2d94ca72fbea39302f45110fc49360bdba052ba58aebdff25e32caa6d513d1cd7449e8a16a4e520f7605280b4a20dd76a255c6db73e3fc9fed990e1297618dcfbc9cd000368cc7b5a13cf1894532ff97ff5de70fb5f3de6fae36e8ed799bdc0465b7b321ae187588d5828b284feb22ff91bd02126d69c6030032686ea73b58692fe142737ca63bd9d4c6e8610433701b0b29f97c4e3e0604adf68b1298dda1f4b57d851a75eb210fd7daba7c3cb497a4d5b6b07090d550b666d1a55c7b9a41b77041ceabc607307a4ac7e0e4e40535292aaa0e8e73d745b8cc57480b31b391ffeb0efa3432df96ae9183db9b18f4144a8968c4565823ddce7f803993a8de349dcb38e31e142c69c22a31b9ec79779b2b672d8653aded11cc86476989eaea5d87bef3ee1e997676d26ed27e02b192202abe83a88a3598aa3d40e85bc2100ccfe433de10cfa8535f78a67c098e960a4ee3986486b506473bcf7329b7e8f36e045a1c136c3d9a169ca875bb8663ee4178105a466ab557ac960378a3c43ffe44b55309de627c3df2427d6c38ef3c45821c6878854520de96f88a4fe134774e5ac6aa773ffa52d4686e1d8c834fed2ae7897258edb299b9943a280ef82ac010bc3b73275fa1201791c01456d90cfbd3f81ad0d795397a346dc38c85e3f1046a3cca49845cbf343e4c599a74f313e9dc041a3d054c7cbed5280582f408d3fce74eabc6479d7726a76acabf8a918f12d3b717204e4a06d54f93e3468cb0b34ba26fad8b0c54792b719e11799006a08efced58177eda5d85ce0197dd14158526a4f5b4ad574c93916ebbb7f30a441dd8b5279556b703444b428a017a9a46c344223ace9479dbce73df5c8b8ddb4947966f1feccb0518f15bc03c15443af40ee88b7a5a1c5ff9fca9955b5e4a7ea5550ec28656d55f8a1c009a86d86215cfd73c234d0aedf8f10487eb0439f6302576327e693b0ea96013a8d3cda30dfa4a446da10c57cba3df06653e46fb6a080802b4f195796a4620e01d14130c819eb6bdb0b689cabec69973c22cbae9fa63d026dc5db46d7214a4e60c4095a744cc8434c3663fa8ceae1821f55598f509c00c204295d42c4371d5a5e30922c6f2dee8fdaf3882ada245877a013f80e36eb990eb9cbdcb77e821b20205ea5f00836ec70c783c91be158277ba5baf1ffe1251ceb161ef1ea23acb6b6aa36d2c60fe2b151df64e0d547a2f0ddf2ca3ad48005ecfec88dc147e369773341d796317c84d2253569f3113427";

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

function encryptRawLegacy(
  plaintext: Uint8Array,
  password: string
): Uint8Array {
  const salt = Buffer.from(fixedSalt());
  const nonce = Buffer.from(fixedNonce());
  const key = pbkdf2Sync(
    password,
    salt,
    LEGACY_PBKDF2_ITERATIONS,
    KEY_SIZE,
    "sha256"
  );
  const cipher = createCipheriv("aes-256-gcm", key, nonce);
  const ciphertext = Buffer.concat([
    cipher.update(Buffer.from(plaintext)),
    cipher.final(),
  ]);
  const tag = cipher.getAuthTag();

  return Buffer.concat([
    Buffer.from(MAGIC, "ascii"),
    Buffer.from([LEGACY_VERSION]),
    salt,
    nonce,
    tag,
    ciphertext,
  ]);
}

function sampleBundle(): Bundle {
  const bundle = new Bundle();
  const manifestCreatedAt = "2026-04-01T00:00:00Z";
  const manifestEvent: Event = {
    seq: 0,
    prev_hash: GENESIS_HASH,
    type: "atb.bundle.manifest",
    hash_algo: "sha256",
    data: JSON.stringify({
      version: "1",
      created_at: manifestCreatedAt,
      bundle_id: "00112233445566778899aabbccddeeff",
    }),
    timestamp: manifestCreatedAt,
  };
  bundle.records[0] = {
    event: manifestEvent,
    hash: computeHash(manifestEvent, GENESIS_HASH),
  };
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
  }, 30_000);

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
    encrypted[MAGIC.length] = 0x03;
    expect(() => decryptRaw(encrypted, "test123")).toThrow("unsupported version");
  });

  it("decrypts legacy version payloads", () => {
    const plaintext = Buffer.from('{"head_hash":"abc","records":[]}', "utf8");
    const encrypted = encryptRawLegacy(plaintext, "test123");

    expect(encrypted[MAGIC.length]).toBe(LEGACY_VERSION);
    expect(Buffer.from(decryptRaw(encrypted, "test123"))).toEqual(plaintext);
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
  }, 30_000);

  it("bundle methods roundtrip", async () => {
    const bundle = sampleBundle();
    const encrypted = await bundle.encrypt("test123");
    const decrypted = await Bundle.decrypt("test123", encrypted);
    expect(decrypted.records.map((r) => r.hash)).toEqual(
      bundle.records.map((r) => r.hash)
    );
  }, 30_000);

  it("parity with Go/Python fixed vector", async () => {
    const bundle = sampleBundle();
    const encrypted = await encryptBundle(bundle.records, "test123", {
      salt: fixedSalt(),
      nonce: fixedNonce(),
    });
    expect(Buffer.from(encrypted).toString("hex")).toBe(EXPECTED_GO_VECTOR_HEX);
  }, 30_000);
});
