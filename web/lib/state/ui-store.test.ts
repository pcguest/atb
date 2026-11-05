import { beforeEach, describe, expect, it } from "vitest";

import { useUIStore } from "@/lib/state/ui-store";

describe("UI theme store", () => {
  beforeEach(() => {
    localStorage.clear();
    useUIStore.setState({ theme: "system" });
  });

  it("defaults to the system preference", () => {
    expect(useUIStore.getInitialState().theme).toBe("system");
  });

  it("persists an explicit theme choice", () => {
    useUIStore.getState().setTheme("light");
    expect(useUIStore.getState().theme).toBe("light");
    const persisted = JSON.parse(localStorage.getItem("atb-ui-store-v1") ?? "{}");
    expect(persisted.state.theme).toBe("light");
  });

  it("migrates the legacy default dark theme to system", async () => {
    localStorage.setItem("atb-ui-store-v1", JSON.stringify({ state: { theme: "dark" }, version: 2 }));
    await useUIStore.persist.rehydrate();
    expect(useUIStore.getState().theme).toBe("system");
  });

  it("falls back to system for an invalid persisted theme", async () => {
    localStorage.setItem("atb-ui-store-v1", JSON.stringify({ state: { theme: "sepia" }, version: 3 }));
    await useUIStore.persist.rehydrate();
    expect(useUIStore.getState().theme).toBe("system");
  });
});
