import { beforeEach, describe, expect, it } from "vitest";

import { useUIStore } from "@/lib/state/ui-store";

describe("UI theme store", () => {
  beforeEach(() => {
    localStorage.clear();
    useUIStore.setState({ theme: "system" });
  });

  it("defaults to the system preference", () => {
    expect(useUIStore.getState().theme).toBe("system");
  });

  it("persists an explicit theme choice", () => {
    useUIStore.getState().setTheme("light");
    expect(useUIStore.getState().theme).toBe("light");
    const persisted = JSON.parse(localStorage.getItem("atb-ui-store-v1") ?? "{}");
    expect(persisted.state.theme).toBe("light");
  });
});
