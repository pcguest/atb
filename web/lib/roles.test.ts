import { describe, expect, it } from "vitest";

import {
  canExportEvidence,
  canRevealMaskedFields,
  canViewRawData,
  dashboardRoles,
  isDensePresentation,
} from "@/lib/roles";

describe("presentation modes", () => {
  it("defines the agreed Engineer, Security, and Auditor modes", () => {
    expect(dashboardRoles).toEqual(["engineer", "security", "auditor"]);
  });

  it("does not use presentation modes as access control", () => {
    for (const role of dashboardRoles) {
      expect(canViewRawData(role)).toBe(true);
      expect(canExportEvidence(role)).toBe(true);
      expect(canRevealMaskedFields(role)).toBe(true);
    }
  });

  it("uses Engineer as the dense presentation", () => {
    expect(isDensePresentation("engineer")).toBe(true);
    expect(isDensePresentation("security")).toBe(false);
    expect(isDensePresentation("auditor")).toBe(false);
  });
});
