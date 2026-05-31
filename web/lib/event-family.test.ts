import { describe, expect, it } from "vitest";

import { eventFamily, eventFamilyClass } from "./event-family";

describe("eventFamily", () => {
  it("groups capture (atb.*) events with their runtime (ai.*) families", () => {
    // Regression: these previously fell through to "other"/grey.
    expect(eventFamily("atb.tool.call")).toBe("tool");
    expect(eventFamily("atb.llm.request")).toBe("llm");
    expect(eventFamily("atb.llm.response")).toBe("llm");
    expect(eventFamily("atb.human.override")).toBe("human");
  });

  it("categorises the forensic accountability events", () => {
    expect(eventFamily("ai.action.error")).toBe("action");
    expect(eventFamily("ai.action.executed")).toBe("action");
    expect(eventFamily("ai.policy.decision")).toBe("policy");
  });

  it("keeps the existing runtime families", () => {
    expect(eventFamily("ai.llm.call")).toBe("llm");
    expect(eventFamily("ai.tool.exec")).toBe("tool");
    expect(eventFamily("ai.chain.run")).toBe("chain");
    expect(eventFamily("ai.job.scheduled")).toBe("job");
    expect(eventFamily("atb.corroboration.external")).toBe("corroboration");
  });

  it("falls back to other for unknown types", () => {
    expect(eventFamily("dev.session")).toBe("other");
    expect(eventFamily("")).toBe("other");
  });
});

describe("eventFamilyClass", () => {
  it("maps forensic events to distinct colour classes", () => {
    expect(eventFamilyClass("ai.action.error")).toBe("ev-action");
    expect(eventFamilyClass("atb.tool.call")).toBe("ev-tool");
    expect(eventFamilyClass("atb.llm.response")).toBe("ev-llm");
  });

  it("always returns a defined class", () => {
    for (const t of ["ai.job.started", "atb.corroboration.external", "unknown.type"]) {
      expect(eventFamilyClass(t)).toMatch(/^ev-/);
    }
  });
});
