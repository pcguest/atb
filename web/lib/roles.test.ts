import { describe, expect, it } from "vitest";

import {
  canExportEvidence,
  canRevealMaskedFields,
  canViewExecutiveSummary,
  canViewRawData,
} from "@/lib/roles";

describe("role permission helpers", () => {
  it("allows engineer-only raw event access", () => {
    expect(canViewRawData("engineer")).toBe(true);
    expect(canViewRawData("auditor")).toBe(false);
    expect(canViewRawData("executive")).toBe(false);
  });

  it("allows auditor and engineer evidence export", () => {
    expect(canExportEvidence("engineer")).toBe(true);
    expect(canExportEvidence("auditor")).toBe(true);
    expect(canExportEvidence("executive")).toBe(false);
  });

  it("allows executive-only trend summary", () => {
    expect(canViewExecutiveSummary("engineer")).toBe(false);
    expect(canViewExecutiveSummary("auditor")).toBe(false);
    expect(canViewExecutiveSummary("executive")).toBe(true);
  });

  it("allows engineer-only reveal actions", () => {
    expect(canRevealMaskedFields("engineer")).toBe(true);
    expect(canRevealMaskedFields("auditor")).toBe(false);
    expect(canRevealMaskedFields("executive")).toBe(false);
  });
});
