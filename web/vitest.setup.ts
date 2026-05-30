import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { cleanup } from "@testing-library/react";

// vitest is configured without `globals`, so @testing-library/react cannot
// auto-register its DOM cleanup. Without this, rendered output accumulates
// across `it` blocks and queries collide ("found multiple elements").
afterEach(() => {
  cleanup();
});
