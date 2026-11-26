import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import {
  ATBVerificationError,
  Bundle,
  MAX_MANIFEST_VERSION,
  UnsupportedManifestVersionError,
} from "./bundle.js";
import { GENESIS_HASH, computeHash } from "./hash.js";
import type { Event } from "./event.js";
import type { ATBRecord } from "./types.js";

const MANIFEST = "atb.bundle.manifest";
const SIGNATURE = "atb.bundle.signature";
const CREATED_AT = "2026-03-27T12:00:00Z";
const BUNDLE_ID = "a".repeat(32);

const tempDirs: string[] = [];

afterEach(() => {
  for (const dir of tempDirs.splice(0)) {
    rmSync(dir, { recursive: true, force: true });
  }
});

function tempPath(): string {
  const dir = mkdtempSync(join(tmpdir(), "atb-version-guard-"));
  tempDirs.push(dir);
  return join(dir, "bundle.atb");
}

function chain(events: Event[]): ATBRecord[] {
  const hasManifest = events.length > 0 && events[0].type === MANIFEST;
  let prev = GENESIS_HASH;
  const records: ATBRecord[] = [];
  for (let i = 0; i < events.length; i++) {
    const event = events[i];
    event.seq = hasManifest ? (i === 0 ? 0 : i) : i + 1;
    event.prev_hash = prev;
    event.hash_algo = "sha256";
    const hash = computeHash(event, prev);
    records.push({ event, hash });
    prev = hash;
  }
  return records;
}

function manifest(data: unknown): Event {
  return {
    seq: 0,
    prev_hash: GENESIS_HASH,
    type: MANIFEST,
    data,
  };
}

function event(): Event {
  return {
    seq: 0,
    prev_hash: GENESIS_HASH,
    type: "ai.tool.exec",
    data: { ok: true },
  };
}

function signature(): Event {
  return {
    seq: 0,
    prev_hash: GENESIS_HASH,
    type: SIGNATURE,
    data: {
      bundle_hash: "0".repeat(64),
      signature: "not-a-signature",
      pubkey: "not-a-pubkey",
      backend: "test",
    },
  };
}

function writeRecords(records: ATBRecord[], path = tempPath()): string {
  const payload = records.map((record) => JSON.stringify(record)).join("\n");
  writeFileSync(path, payload + "\n", "utf8");
  return path;
}

function bundleFrom(records: ATBRecord[]): Bundle {
  const bundle = new Bundle();
  bundle.records.length = 0;
  bundle.records.push(...records);
  return bundle;
}

function manifestVersion(version: unknown): Record<string, unknown> {
  return { version, created_at: CREATED_AT, bundle_id: BUNDLE_ID };
}

describe("Bundle manifest version guard", () => {
  it.each([
    ["v1 string", JSON.stringify(manifestVersion("1"))],
    [
      "v1 absent version",
      JSON.stringify({ created_at: CREATED_AT, bundle_id: BUNDLE_ID }),
    ],
    ["v2 structured", manifestVersion(2)],
    ["v3 structured", manifestVersion(3)],
    [
      "v3 canonical profile",
      { ...manifestVersion(3), canonical_profile: "atb.profile.agent" },
    ],
  ])("accepts %s manifests as legacy", (_label, data) => {
    const path = writeRecords(chain([manifest(data), event()]));

    const loaded = Bundle.load(path);

    expect(loaded.verify().signatures).toEqual([]);
  });

  it("accepts a bundle with no manifest", () => {
    const path = writeRecords(chain([event()]));

    const loaded = Bundle.load(path);

    expect(loaded.verify().signatures).toEqual([]);
  });

  it.each([
    ["integer", 4],
    ["digit string", "4"],
    ["fractional number", 4.5],
    ["fraction just above max", 3.5],
    ["fraction string", "4.0"],
    ["leading whitespace", " 4"],
    ["trailing junk", "4abc"],
    ["zero padded", "0004"],
    ["oversized digits", "9".repeat(30)],
  ])("rejects v4 from a %s before verification", (_label, version) => {
    const records = chain([manifest(manifestVersion(version)), event()]);
    const path = writeRecords(records);

    let error: unknown;
    try {
      Bundle.load(path);
    } catch (caught) {
      error = caught;
    }

    expect(error).toBeInstanceOf(UnsupportedManifestVersionError);
    expect(error).not.toBeInstanceOf(ATBVerificationError);
    const unsupported = error as UnsupportedManifestVersionError;
    expect(unsupported.name).toBe("UnsupportedManifestVersionError");
    expect(unsupported.version).toBe(4);
    expect(unsupported.message).toContain(
      "atb: unsupported manifest version 4: this bundle requires a newer version of the SDK"
    );
  });

  it("rejects v4 written as an integral float", () => {
    const records = chain([manifest(manifestVersion(4)), event()]);
    const path = tempPath();
    const payload = records
      .map((record) => JSON.stringify(record))
      .join("\n")
      .replace('"version":4', '"version":4.0');
    writeFileSync(path, payload + "\n", "utf8");

    expect(() => Bundle.load(path)).toThrowError(
      UnsupportedManifestVersionError
    );
  });

  it("rejects v4 with a signature record before any signature evidence", () => {
    const records = chain([
      manifest(manifestVersion(4)),
      event(),
      signature(),
    ]);
    const path = writeRecords(records);
    expect(() => Bundle.load(path)).toThrowError(
      UnsupportedManifestVersionError
    );

    const inMemory = bundleFrom(records);
    expect(() => inMemory.signatures()).toThrowError(
      UnsupportedManifestVersionError
    );
    expect(() => inMemory.verify()).toThrowError(
      UnsupportedManifestVersionError
    );
  });

  it("rejects signaturesAsync for an in-memory v4 bundle", async () => {
    const inMemory = bundleFrom(
      chain([manifest(manifestVersion(4)), event(), signature()])
    );

    await expect(inMemory.signaturesAsync()).rejects.toBeInstanceOf(
      UnsupportedManifestVersionError
    );
  });

  it("rejects in-memory v4 verify and encrypt", async () => {
    const inMemory = bundleFrom(
      chain([manifest(manifestVersion(4)), event()])
    );

    expect(() => inMemory.verify()).toThrowError(
      UnsupportedManifestVersionError
    );
    await expect(inMemory.encrypt("password")).rejects.toBeInstanceOf(
      UnsupportedManifestVersionError
    );
  });

  it("rejects non-finite manifest numbers", () => {
    const bundle = new Bundle();
    bundle.records[0].event.data = {
      ...manifestVersion(Number.POSITIVE_INFINITY),
    };

    let error: unknown;
    try {
      bundle.verify();
    } catch (caught) {
      error = caught;
    }

    expect(error).toBeInstanceOf(UnsupportedManifestVersionError);
    expect((error as UnsupportedManifestVersionError).version).toBe(
      MAX_MANIFEST_VERSION + 1
    );
  });

  it("does not determinably reject booleans or strings without a leading digit", () => {
    const booleanPath = writeRecords(
      chain([manifest(manifestVersion(true)), event()])
    );
    expect(() => Bundle.load(booleanPath)).not.toThrowError(
      UnsupportedManifestVersionError
    );

    const wordPath = writeRecords(
      chain([manifest(manifestVersion("abc")), event()])
    );
    expect(() => Bundle.load(wordPath)).not.toThrowError(
      UnsupportedManifestVersionError
    );
  });

  it("pins MAX_MANIFEST_VERSION to Go's ManifestVersionMax", () => {
    expect(MAX_MANIFEST_VERSION).toBe(3);
  });

  it("still writes manifest version 1", () => {
    const path = tempPath();
    new Bundle({ path }).save();

    const first = JSON.parse(readFileSync(path, "utf8").split("\n")[0]) as {
      event: { data: string };
    };
    const data = JSON.parse(first.event.data) as { version: unknown };

    expect(data.version).toBe("1");
  });
});
