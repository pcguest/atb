import { describe, expect, it } from "vitest";

import { eventFamily, eventFamilyClass, eventSummary } from "./event-family";

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
    expect(eventFamily("data.retention.enforced")).toBe("retention");
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

describe("eventSummary", () => {
  it("summarises forensic events", () => {
    expect(eventSummary("atb.tool.call", { tool_name: "wipe_db" })).toBe("tool=wipe_db");
    expect(eventSummary("ai.action.error", { action_id: "toolu_1", error_class: "failed" })).toBe(
      "action=toolu_1 error_class=failed",
    );
    expect(eventSummary("atb.llm.response", { method: "POST", path: "/v1/messages", status_code: 200 })).toBe(
      "POST /v1/messages → 200",
    );
    expect(eventSummary("ai.policy.decision", { decision: "deny" })).toBe("decision=deny");
    expect(eventSummary("data.export.executed", { execution_outcome: "success" })).toBe(
      "export outcome=success",
    );
    expect(eventSummary("atb.data.export", { export_target: "s3://bucket" })).toBe(
      "export=s3://bucket",
    );
    expect(eventSummary("ai.action.precommit", { action_type: "deploy" })).toBe("action=deploy");
    expect(eventSummary("data.retention.policy_set", { days: 183 })).toBe("retention=183d");
    expect(
      eventSummary("data.retention.enforced", { operation: "s3_object_lock_request", outcome: "request_accepted" }),
    ).toBe("s3_object_lock_request outcome=request_accepted");
    expect(
      eventSummary("atb.human.override", {
        overridden_action_id: "act-1",
        identity_evidence: {
          identity_provider: "idp.example",
          subject: "reviewer-1",
        },
      }),
    ).toBe("override=act-1 reviewer=idp.example/reviewer-1");
    expect(
      eventSummary("ai.action.executed", {
        execution_outcome: "success",
        effective_scope: "role:prod-operator",
      }),
    ).toBe("executed outcome=success scope=role:prod-operator");
    expect(
      eventSummary("ai.action.precommit", {
        action_type: "deploy",
        principal: { type: "agent", id_hash: "sha256:a1", on_behalf_of: "sha256:u9" },
      }),
    ).toBe("action=deploy by agent:sha256:a1 on_behalf_of sha256:u9");
  });

  it("returns empty string when there is nothing concise to say", () => {
    expect(eventSummary("dev.session", { x: 1 })).toBe("");
    expect(eventSummary("atb.tool.call", {})).toBe("");
    expect(eventSummary("ai.action.error", { action_id: "x" })).toBe(""); // no error_class
    expect(eventSummary("atb.llm.request", "not-an-object")).toBe("");
  });
});
