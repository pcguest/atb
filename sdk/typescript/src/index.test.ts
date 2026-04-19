import { describe, expect, it } from "vitest";
import { SDK_VERSION, version } from "./index.js";

describe("version()", () => {
  it("returns the SDK version string", () => {
    const v = version();
    expect(v.version).toBe(SDK_VERSION);
    expect(typeof v.version).toBe("string");
    expect(v.version.length).toBeGreaterThan(0);
  });

  it("returns the algorithm field", () => {
    const v = version();
    expect(typeof v.algorithm).toBe("string");
    expect(v.algorithm.length).toBeGreaterThan(0);
  });

  it("returned object has exactly version and algorithm keys", () => {
    const v = version();
    expect(Object.keys(v).sort()).toEqual(["algorithm", "version"]);
  });
});
