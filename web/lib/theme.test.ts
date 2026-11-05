import { describe, expect, it } from "vitest";

import { isThemePreference, resolveTheme } from "@/lib/theme";

describe("theme preference", () => {
  it("follows the system preference by default", () => {
    expect(resolveTheme("system", true)).toBe("dark");
    expect(resolveTheme("system", false)).toBe("light");
  });

  it("honours an explicit choice", () => {
    expect(resolveTheme("light", true)).toBe("light");
    expect(resolveTheme("dark", false)).toBe("dark");
  });

  it("rejects unsupported persisted values", () => {
    expect(isThemePreference("system")).toBe(true);
    expect(isThemePreference("sepia")).toBe(false);
    expect(isThemePreference(null)).toBe(false);
  });
});
