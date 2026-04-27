/**
 * Smoke tests for the async verification path added to the TypeScript SDK.
 *
 * Two of the four cases exercise `verifySignatureRecordAsync` indirectly:
 * the function is internal to ./bundle.ts and is not part of the public
 * export surface, so we drive it through `Bundle.signaturesAsync()`, which
 * is the documented entry point that fans out to it for every signature
 * record. The "unknown algorithm" case asserts the rejection bubbles up
 * through `signaturesAsync()` with the same "upgrade" message that
 * `verifySignatureRecordAsync` throws.
 */

import { createHash, generateKeyPairSync } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import type { BundleSignature } from "./types.js";

const TMP_DIRS: string[] = [];

afterEach(() => {
  while (TMP_DIRS.length) {
    const d = TMP_DIRS.pop();
    if (d) rmSync(d, { recursive: true, force: true });
  }
});

function mkTmp(): string {
  const d = mkdtempSync(join(tmpdir(), "atb-smoke-"));
  TMP_DIRS.push(d);
  return d;
}

function makeEd25519SignedBundle(): { path: string } {
  const path = join(mkTmp(), "bundle.atb");
  const b = new Bundle({ path });
  b.append("ai.tool.exec", { ok: true });
  b.save();

  const { privateKey } = generateKeyPairSync("ed25519");
  const pem = privateKey.export({ type: "pkcs8", format: "pem" }) as string;
  b.signLocal(pem, path, { keyId: "smoke-test-key" });
  return { path };
}

describe("Bundle async verification smoke", () => {
  it("ed25519 bundle round-trips through signaturesAsync", async () => {
    const { path } = makeEd25519SignedBundle();
    const loaded = Bundle.load(path);
    const results = await loaded.signaturesAsync();
    expect(results).toHaveLength(1);
    expect(results[0].valid).toBe(true);
    expect(results[0].algorithm).toBe("ed25519");
  });

  it("tampered payload returns valid:false via signaturesAsync", async () => {
    const { path } = makeEd25519SignedBundle();

    // Corrupt one byte of the non-signature record's payload, leaving the
    // signature record intact. The bundle_hash check must fail.
    const raw = readFileSync(path);
    const idx = raw.indexOf("ok");
    expect(idx).toBeGreaterThan(0);
    raw[idx] ^= 0x01;
    writeFileSync(path, raw);

    const loaded = Bundle.load(path);
    const results = await loaded.signaturesAsync();
    expect(results).toHaveLength(1);
    expect(results[0].valid).toBe(false);
  });

  it("unknown algorithm rejects from signaturesAsync with 'upgrade' message", async () => {
    const path = join(mkTmp(), "bundle.atb");
    const b = new Bundle({ path });
    b.append("ai.tool.exec", { ok: true });
    b.save();

    const raw = readFileSync(path);
    // bundle_hash must be sha256(prefix-before-this-record) so that the
    // hash check inside verifySignatureRecordAsync passes and execution
    // reaches the algorithm switch — that's where the "upgrade" throw lives.
    const prefixDigest = createHash("sha256").update(raw).digest("hex");
    const sigRecord = {
      event: {
        seq: 99,
        prev_hash: "0".repeat(64),
        type: "atb.bundle.signature",
        hash_algo: "sha256",
        timestamp: new Date().toISOString(),
        data: {
          bundle_hash: prefixDigest,
          signature: Buffer.from("not-a-real-sig").toString("base64"),
          pubkey: Buffer.from("not-a-real-key").toString("base64"),
          algorithm: "rsa-4096",
          backend: "aws-kms",
          key_id: "fake",
          signed_at: new Date().toISOString(),
        },
      },
      hash: "0".repeat(64),
    };
    const newline = raw.length > 0 && raw[raw.length - 1] !== 0x0a ? "\n" : "";
    const appended = Buffer.concat([
      raw,
      Buffer.from(newline + JSON.stringify(sigRecord) + "\n", "utf8"),
    ]);
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, appended);

    const loaded = Bundle.load(path);
    await expect(loaded.signaturesAsync()).rejects.toThrow(/upgrade/);
  });

  it("BundleSignature interface fields are present on a parsed signature record", async () => {
    const { path } = makeEd25519SignedBundle();
    const loaded = Bundle.load(path);
    const results = await loaded.signaturesAsync();
    expect(results).toHaveLength(1);
    const evidence = results[0];
    // Algorithm is mandatory on SignatureEvidence; the BundleSignature wire
    // type makes algorithm/key_id/backend/signed_at optional, so we check
    // that the parser surfaced them (and tolerate empty strings for the
    // optional ones, matching the current normalisation behaviour).
    expect(typeof evidence.algorithm).toBe("string");
    expect(typeof evidence.key_id).toBe("string");
    expect(typeof evidence.backend).toBe("string");
    expect(typeof evidence.signed_at).toBe("string");
    expect(evidence.signed_at.length).toBeGreaterThan(0);

    // Type-level assertion that the BundleSignature wire interface is
    // re-exported and structurally usable from the public entry point.
    const wire: BundleSignature = {
      bundle_hash: evidence.bundle_hash,
      signature: "",
      pub_key: evidence.pubkey,
      algorithm: "ed25519",
      key_id: evidence.key_id,
      backend: "local",
      signed_at: evidence.signed_at,
    };
    expect(wire.algorithm).toBe("ed25519");
  });
});
