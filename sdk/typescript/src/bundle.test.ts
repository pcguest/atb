import { generateKeyPairSync } from "node:crypto";
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { spawnSync } from "node:child_process";
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
    bundle.append("dev.session", { gate: "pass" });
    bundle.save();

    const loaded = Bundle.load(path);
    expect(loaded.length).toBe(3);
    expect(loaded.records[0].event.prev_hash).toBe(GENESIS_HASH);
    expect(loaded.records[1].event.prev_hash).toBe(loaded.records[0].hash);
    expect(loaded.records[2].event.prev_hash).toBe(loaded.records[1].hash);
    expect(() => loaded.verify()).not.toThrow();
  });

  it("raises ATBVerificationError on tampered data", () => {
    const bundle = new Bundle();
    bundle.append("dev.session", { ok: true });
    bundle.append("decision", { choice: "ship" });

    bundle.records[1].event.type = "decision.tampered";

    expect(() => bundle.verify()).toThrowError(ATBVerificationError);
  });

  it("sign_local records local provenance fields", () => {
    const path = tempBundlePath();
    const bundle = new Bundle({ path });
    bundle.append("ai.tool.exec", { ok: true });
    bundle.save();

    const record = bundle.sign_local(privateKeyPEM(), path);
    const data = record.event.data as Record<string, unknown>;

    expect(data.backend).toBe("local");
    expect(data.key_id).toBe("");
    expect(isRFC3339UTC(data.signed_at)).toBe(true);
    const report = bundle.verify();
    expect(report.signatures[0].backend).toBe("local");
    expect(report.signatures[0].valid).toBe(true);
  });

  // Compiles the CLI from scratch (empty GOCACHE), which can blow the 60s
  // default timeout on cold Windows CI runners.
  it("verify returns local signature evidence for a Go-signed bundle", { timeout: 180_000 }, () => {
    const path = tempBundlePath();
    const keyPath = join(dirname(path), "atb-key.pem");
    const bundle = new Bundle({ path });
    bundle.append("ai.tool.exec", { source: "typescript-fixture" });
    bundle.save();
    writeFileSync(keyPath, privateKeyPEM(), "utf8");
    const goCache = join(dirname(path), "gocache");
    mkdirSync(goCache, { recursive: true });

    const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "../../..");
    const result = spawnSync(
      "go",
      [
        "run",
        "./cmd/atb",
        "sign",
        "--bundle",
        path,
        "--key",
        keyPath,
      ],
      { cwd: repoRoot, encoding: "utf8", env: { ...process.env, GOCACHE: goCache } }
    );
    expect(result.status, result.stderr).toBe(0);

    const loaded = Bundle.load(path);
    const report = loaded.verify();

    expect(report.signatures).toHaveLength(1);
    expect(report.signatures[0].backend).toBe("local");
    expect(report.signatures[0].key_id).toBe("");
    expect(isRFC3339UTC(report.signatures[0].signed_at)).toBe(true);
    expect(report.signatures[0].pubkey).not.toBe("");
    expect(report.signatures[0].bundle_hash).not.toBe("");
    expect(report.signatures[0].valid).toBe(true);
  }, 60_000);

  it("keeps manifest data double encoded", () => {
    const path = tempBundlePath();
    const bundle = new Bundle({ path });
    bundle.save();

    const [line] = readFileSync(path, "utf8").trim().split("\n");
    const record = JSON.parse(line) as { event: { data: unknown } };

    expect(typeof record.event.data).toBe("string");
    const manifest = JSON.parse(record.event.data as string) as Record<string, unknown>;
    expect(manifest.version).toBe("1");
    expect(typeof manifest.bundle_id).toBe("string");
    expect(typeof manifest.created_at).toBe("string");
  });
});

function privateKeyPEM(): string {
  const { privateKey } = generateKeyPairSync("ed25519");
  return privateKey.export({ type: "pkcs8", format: "pem" }).toString();
}

function isRFC3339UTC(value: unknown): boolean {
  if (typeof value !== "string" || !value.endsWith("Z")) {
    return false;
  }
  return !Number.isNaN(Date.parse(value));
}
