import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { ATBVerificationError, Bundle } from "./bundle.js";
import { GENESIS_HASH, computeHash } from "./hash.js";
import type { Event } from "./event.js";

const tempDirs: string[] = [];

afterEach(() => {
  for (const dir of tempDirs.splice(0)) {
    rmSync(dir, { recursive: true, force: true });
  }
});

function tempBundlePath(): string {
  const dir = mkdtempSync(join(tmpdir(), "atb-chain-field-"));
  tempDirs.push(dir);
  return join(dir, "bundle.atb");
}

function makeBundle(withManifest: boolean): Bundle {
  const bundle = new Bundle();
  if (!withManifest) {
    bundle.records.length = 0;
  }
  bundle.append("ai.tool.exec", { ok: true });
  bundle.append("ai.tool.exec", { ok: false });
  return bundle;
}

function targetIndex(withManifest: boolean): number {
  return withManifest ? 1 : 0;
}

function rewriteTargetWithDuplicateKey(
  path: string,
  field: "seq" | "prev_hash",
  duplicateValue: number | string
): void {
  const lines = readFileSync(path, "utf8").split("\n");
  const record = JSON.parse(lines[1]) as { event: Event; hash: string };
  const expectedPrevHash = record.event.prev_hash;
  const expectedSeq = 1;
  const event: Event = { ...record.event, seq: expectedSeq };
  const expectedHash = computeHash(event, expectedPrevHash);
  const eventJSON = JSON.stringify(event);

  const needle =
    field === "seq"
      ? `"seq":${expectedSeq}`
      : `"prev_hash":"${expectedPrevHash}"`;
  const replacement =
    field === "seq"
      ? `${needle},"seq":${duplicateValue}`
      : `${needle},"prev_hash":"${duplicateValue}"`;
  const tamperedEventJSON = eventJSON.replace(needle, replacement);
  expect(tamperedEventJSON).not.toBe(eventJSON);
  lines[1] = `{"event":${tamperedEventJSON},"hash":"${expectedHash}"}`;
  writeFileSync(path, lines.join("\n"), "utf8");
}

describe("Bundle recorded chain fields", () => {
  it.each([
    ["manifest", true],
    ["no-manifest", false],
  ])("still verifies a valid %s bundle", (_label, withManifest) => {
    const bundle = makeBundle(withManifest as boolean);

    expect(bundle.verify().signatures).toEqual([]);
  });

  it.each([
    ["manifest", true],
    ["no-manifest", false],
  ])("rejects a mutated seq in a %s bundle", (_label, withManifest) => {
    const bundle = makeBundle(withManifest as boolean);
    bundle.records[targetIndex(withManifest as boolean)].event.seq = 999;

    // A plain mutation also breaks the stored record hash, so the pre-fix
    // verifier would report a hash mismatch. The duplicate-key test below is
    // the actual acceptance delta.
    expect(() => bundle.verify()).toThrowError(/sequence mismatch/);
    expect(() => bundle.verify()).toThrowError(ATBVerificationError);
  });

  it.each([
    ["manifest", true],
    ["no-manifest", false],
  ])("rejects a mutated prev_hash in a %s bundle", (_label, withManifest) => {
    const bundle = makeBundle(withManifest as boolean);
    bundle.records[targetIndex(withManifest as boolean)].event.prev_hash =
      "f".repeat(64);

    expect(() => bundle.verify()).toThrowError(/prev_hash mismatch/);
    expect(() => bundle.verify()).toThrowError(ATBVerificationError);
  });

  it.each([
    ["manifest", true],
    ["no-manifest", false],
  ])("rejects a missing seq in a %s bundle", (_label, withManifest) => {
    const bundle = makeBundle(withManifest as boolean);
    const event = bundle.records[targetIndex(withManifest as boolean)]
      .event as unknown as Record<string, unknown>;
    delete event.seq;

    expect(() => bundle.verify()).toThrowError(/sequence mismatch/);
  });

  it.each([
    ["manifest", true],
    ["no-manifest", false],
  ])("rejects a missing prev_hash in a %s bundle", (_label, withManifest) => {
    const bundle = makeBundle(withManifest as boolean);
    const event = bundle.records[targetIndex(withManifest as boolean)]
      .event as unknown as Record<string, unknown>;
    delete event.prev_hash;

    expect(() => bundle.verify()).toThrowError(/prev_hash mismatch/);
  });

  it.each([
    ["manifest", true],
    ["no-manifest", false],
  ])("rejects both mutated chain fields in a %s bundle", (_label, withManifest) => {
    const bundle = makeBundle(withManifest as boolean);
    const event: Event =
      bundle.records[targetIndex(withManifest as boolean)].event;
    event.seq = 999;
    event.prev_hash = "f".repeat(64);

    expect(() => bundle.verify()).toThrowError(/sequence mismatch/);
  });

  it("requires genesis as the first prev_hash", () => {
    const bundle = makeBundle(true);
    expect(bundle.records[0].event.prev_hash).toBe(GENESIS_HASH);

    bundle.records[0].event.prev_hash = "f".repeat(64);

    expect(() => bundle.verify()).toThrowError(/prev_hash mismatch/);
  });

  it.each([
    ["seq", 999, /sequence mismatch/],
    ["prev_hash", "f".repeat(64), /prev_hash mismatch/],
  ])(
    "rejects a duplicate %s key: the acceptance delta",
    (field, duplicateValue, message) => {
      // JSON.parse keeps the last duplicate key, while the stored hash was
      // computed over the expected value. The pre-fix verifier overwrote the
      // parsed value and recomputed the matching hash, so it accepted this.
      const path = tempBundlePath();
      makeBundle(true).save(path);
      rewriteTargetWithDuplicateKey(
        path,
        field as "seq" | "prev_hash",
        duplicateValue as number | string
      );

      const loaded = Bundle.load(path);
      const recorded = loaded.records[1].event as unknown as Record<
        string,
        unknown
      >;
      expect(recorded[field]).toEqual(duplicateValue);

      expect(() => loaded.verify()).toThrowError(message as RegExp);
    }
  );
});
