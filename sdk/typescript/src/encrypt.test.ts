import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import type { Event } from "./event.js";
import {
  EncryptError,
  MAGIC,
  VERSION,
  HEADER_SIZE,
  decryptRaw,
  encryptBundle,
  encryptRaw,
} from "./encrypt.js";
import { GENESIS_HASH, computeHash } from "./hash.js";

const EXPECTED_GO_VECTOR_HEX =
  "4154424501101112131415161718191a1b1c1d1e1f202122232425262728292a2bcf72c3b29cea1d714f7840a5a84eb0545584ed630df2312b4c45ed3c104557b16a274dbecc8ca72d0c2687dfe406548c458895c5bab5220c1b7e5772f7ba8e1f088230de441532de4e7098e2ad5178700c3e455c621bab8db23d962db03c073c1b336c3da5f0e5d4a34770dc5109cfb93a4d90d9cfd6307b6a4870899fc46e01d03603ad27b9124fc8588d3ff04aae71b40aa69f49eedcb500b18c39603f293c41a42627e061094468055dbf077a9581f2248dd5ba9c0c8e1691554670ea51a07cc7918f0be0adddefbd3bc2faa56f9fd5989722899e482f0cc03f7d1827d3e10847f17fedcc83e8d2237924ac5b6cc97e21a82265a1182158f270e2153a1d39ef26969631d96935e3e7f43bb5ec28bb1cf76a2794d857c80fc008845a552bebfc2331dac9434a779e93c512d7c2cd9b01ad7090a94c406f7ec9a98a60da10f4596a56567bdd8a1ff2eb176d5999347d07b10b64d3a68a5226a9dd7b290671be2fdff8382362cbfc77a5117a26c96da6f34b158d90abf61f4e29fb863c83aa08d18fc74de6361f5ef4df24fb955d0d5970583c16eb7e213a9e881a3a5ca53dc374e1013f2cd2bb61482c728b56f2c3fe9c678eec9d56d1837ac79fd03091a94151afb6c17b60454090af63314fd9b92ee1fe3faa671cbfd519c35d741062f27b13ee89f0a9a727fdc66136b4ad7ec237e0a5249f346efffc066e475bab9fe887ce1a8593851059f955703e2dd46a556bc3ad290e420c7202515f3253ebbc42d40bb1f74d5c49c96e1b4ec7a2c40bd71847fa63e9b37a07b02f1f70cbc12a14de874fdac42e5ea8fc210bcfec2b1061a659c56b0132fea746f5e36cb22005053a75504b47fa628e20acf458369169a87448e8d4537925589173926ce7aa14e870d3a386be6010461ebbe86ca7acb709f3047b2b20626c15b30e574bcc62c37802033f5be0d9d1ee43336f9475a7db1b0b20bb8c73b0e41a505d167b50a4988e119be29130683e8145840c4217dcfab7709b7649e1406c9569423bb7dca31db64ec12c83a67ad34dd40a8ac407c81a0e4a0c29d55dda364aa66ae1e8d229f6794ecdf2ae1d368589237f0f3b0626fbe0045dec329db64e2058543fce7cabfd1106aabaeddf34a3526f7d633268135f05a97f3612436d0e0866857e9be18f7fb342190ae3dae637fb8c8b571d33b7f3df34ade4641e4343db0b069c0d365af501809191d9c003b80da44d480b16eab8e5ea34282bcbeb07855e3cd78b016664965eda7b63423a55cc80b7d8b4114eba1ce5319b2820c3643adb3a7e2399146170ac6a98c2ce37493a10b96be0fc9c542314a37ff8083dd53183906f5a34ec4da37e8e";

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
