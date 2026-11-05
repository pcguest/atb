import { act, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ThemeSync } from "@/components/ThemeSync";
import { useUIStore } from "@/lib/state/ui-store";

describe("ThemeSync", () => {
  let listeners: Array<() => void>;
  let matches = false;

  beforeEach(() => {
    listeners = [];
    matches = false;
    localStorage.clear();
    useUIStore.setState({ theme: "system" });
    vi.stubGlobal("matchMedia", vi.fn(() => ({
      get matches() { return matches; },
      addEventListener: (_event: string, listener: () => void) => listeners.push(listener),
      removeEventListener: (_event: string, listener: () => void) => {
        listeners = listeners.filter((entry) => entry !== listener);
      },
    })));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("applies an explicit theme and stops following system changes", async () => {
    render(<ThemeSync />);
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("light"));
    act(() => useUIStore.setState({ theme: "dark" }));
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("dark"));
    expect(listeners).toHaveLength(0);
  });

  it("follows system changes while the preference is system", async () => {
    render(<ThemeSync />);
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("light"));
    matches = true;
    act(() => listeners.forEach((listener) => listener()));
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("dark"));
  });
});
