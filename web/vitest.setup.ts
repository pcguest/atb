import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

const values = new Map<string, string>();
const storage: Storage = {
  get length() {
    return values.size;
  },
  clear: () => values.clear(),
  getItem: (key) => values.get(key) ?? null,
  key: (index) => Array.from(values.keys())[index] ?? null,
  removeItem: (key) => {
    values.delete(key);
  },
  setItem: (key, value) => {
    values.set(key, value);
  },
};

Object.defineProperty(globalThis, "localStorage", {
  configurable: true,
  value: storage,
});

// vitest is configured without `globals`, so @testing-library/react cannot
// auto-register its DOM cleanup. Without this, rendered output accumulates
// across `it` blocks and queries collide ("found multiple elements").
afterEach(() => {
  cleanup();
});
