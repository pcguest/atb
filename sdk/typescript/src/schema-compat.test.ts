import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import { prepareForCanonical, type Event } from "./event.js";
import { GENESIS_HASH, computeHash } from "./hash.js";

const tempDirs: string[] = [];

afterEach(() => {
  for (const dir of tempDirs.splice(0)) {
    rmSync(dir, { recursive: true, force: true });
  }
});

function tempBundlePath(): string {
  const dir = mkdtempSync(join(tmpdir(), "atb-ts-schema-"));
  tempDirs.push(dir);
  return join(dir, "bundle.atb");
}

describe("schema compatibility", () => {
  it("canonical JSON is backward-compatible when optional fields are undefined", () => {
    const legacyEvent = {
      seq: 1,
      prev_hash: GENESIS_HASH,
      type: "schema.compat",
      data: { x: 1 },
    };
    const nextEvent: Event = {
      seq: 1,
      prev_hash: GENESIS_HASH,
      type: "schema.compat",
      data: { x: 1 },
      actor_id: undefined,
      org_id: undefined,
      workspace_id: undefined,
    };

    expect(canonicalize(legacyEvent)).toBe(
      canonicalize(prepareForCanonical(nextEvent))
    );
  });

  it("append with undefined/empty identity values is backward-compatible", () => {
    const baseline = new Bundle();
    baseline.append("schema.compat", { x: 1 });

    const withUndefined = new Bundle();
    withUndefined.append("schema.compat", { x: 1 }, {
      actorId: undefined,
      orgId: undefined,
      workspaceId: undefined,
    });

    const withEmpty = new Bundle();
    withEmpty.append("schema.compat", { x: 1 }, {
      actorId: "",
      orgId: "   ",
      workspaceId: "",
    });

    expect(withUndefined.records[0].event).toEqual(baseline.records[0].event);
    expect(withEmpty.records[0].event).toEqual(baseline.records[0].event);
    expect(withUndefined.records[0].hash).toBe(baseline.records[0].hash);
    expect(withEmpty.records[0].hash).toBe(baseline.records[0].hash);
  });

  it("append with identity fields changes hash and still verifies", () => {
    const baseline = new Bundle();
    baseline.append("schema.compat", { x: 1 });

    const withIdentity = new Bundle();
    withIdentity.append("schema.compat", { x: 1 }, {
      actorId: "paddy",
      orgId: "pcguest",
      workspaceId: "local",
    });

    expect(withIdentity.records[0].hash).not.toBe(baseline.records[0].hash);
    expect(withIdentity.records[0].event.actor_id).toBe("paddy");
    expect(withIdentity.records[0].event.org_id).toBe("pcguest");
    expect(withIdentity.records[0].event.workspace_id).toBe("local");
    expect(() => withIdentity.verify()).not.toThrow();
  });

  it("legacy bundle verifies with new SDK", () => {
    const path = tempBundlePath();
    const event: Event = {
      seq: 1,
      prev_hash: GENESIS_HASH,
      type: "legacy.test",
      data: { x: 1 },
    };
    const hash = computeHash(event, GENESIS_HASH);
    writeFileSync(
      path,
      JSON.stringify({ event, hash }) + "\n",
      "utf8"
    );

    const loaded = Bundle.load(path);
    expect(() => loaded.verify()).not.toThrow();
    expect(loaded.records[0].event.actor_id).toBeUndefined();
    expect(loaded.records[0].event.org_id).toBeUndefined();
    expect(loaded.records[0].event.workspace_id).toBeUndefined();
  });
});
