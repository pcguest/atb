import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import { ATBVerificationError, Bundle } from "./bundle.js";
import { GENESIS_HASH } from "./hash.js";

const tempDirs: string[] = [];

afterEach(() => {
  for (const dir of tempDirs.splice(0)) {
    rmSync(dir, { recursive: true, force: true });
  }
});

function tempBundlePath(): string {
  const dir = mkdtempSync(join(tmpdir(), "atb-ts-test-"));
  tempDirs.push(dir);
  return join(dir, "bundle.atb");
}

describe("Bundle", () => {
  it("appends, saves, loads, and verifies records", () => {
    const path = tempBundlePath();
    const bundle = new Bundle({ path });

    bundle.append("dev.session", { date: "2026-03-04" });
    bundle.append("snapshot.build", { gate: "pass" });
    bundle.save();

    const loaded = Bundle.load(path);
    expect(loaded.length).toBe(2);
    expect(loaded.records[0].event.prev_hash).toBe(GENESIS_HASH);
    expect(loaded.records[1].event.prev_hash).toBe(loaded.records[0].hash);
    expect(() => loaded.verify()).not.toThrow();
  });

  it("raises ATBVerificationError on tampered data", () => {
    const bundle = new Bundle();
    bundle.append("dev.session", { ok: true });
    bundle.append("decision", { choice: "ship" });

    bundle.records[1].event.type = "decision.tampered";

    expect(() => bundle.verify()).toThrowError(ATBVerificationError);
  });
});
