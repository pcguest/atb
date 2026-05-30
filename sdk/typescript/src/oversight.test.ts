import { describe, expect, it } from "vitest";
import {
  DATA_EXPORT,
  HUMAN_APPROVAL,
  HUMAN_OVERRIDE,
  TOOL_CALL,
} from "./eventTypes.js";
import {
  DataExportEmitter,
  HumanApprovalEmitter,
  HumanOverrideEmitter,
  ToolCallEmitter,
} from "./oversight.js";

function makeStub(sessionId = "test-session-1") {
  const events: Array<{ type: string; payload: Record<string, unknown> }> = [];
  return {
    sessionId,
    emit(type: string, payload: Record<string, unknown>): void {
      events.push({ type, payload });
    },
    get events() {
      return events;
    },
  };
}

describe("oversight emitters", () => {
  it("ToolCallEmitter throws when toolName is empty", () => {
    const stub = makeStub();
    const emitter = new ToolCallEmitter(stub);
    expect(() => emitter.emit({ toolName: "" })).toThrow("toolName");
    expect(stub.events).toHaveLength(0);
  });

  it("ToolCallEmitter emits correct type and payload", () => {
    const stub = makeStub();
    const emitter = new ToolCallEmitter(stub);
    emitter.emit({ toolName: "web_search", actorId: "alice" });
    expect(stub.events).toHaveLength(1);
    expect(stub.events[0].type).toBe(TOOL_CALL);
    expect(stub.events[0].payload.tool_name).toBe("web_search");
    expect(stub.events[0].payload.actor_id).toBe("alice");
  });

  it("DataExportEmitter emits correct type and payload", () => {
    const stub = makeStub();
    const emitter = new DataExportEmitter(stub);
    emitter.emit({ exportTarget: "csv", recordCount: 42 });
    expect(stub.events[0].type).toBe(DATA_EXPORT);
    expect(stub.events[0].payload.export_target).toBe("csv");
    expect(stub.events[0].payload.record_count).toBe(42);
  });

  it("HumanOverrideEmitter emits correct type and payload", () => {
    const stub = makeStub();
    const emitter = new HumanOverrideEmitter(stub);
    emitter.emit({
      overrideReason: "emergency stop",
      overriddenActionId: "act-99",
    });
    expect(stub.events[0].type).toBe(HUMAN_OVERRIDE);
    expect(stub.events[0].payload.override_reason).toBe("emergency stop");
  });

  it("HumanApprovalEmitter throws when approvedActionId empty", () => {
    const stub = makeStub();
    const emitter = new HumanApprovalEmitter(stub);
    expect(() => emitter.emit({ approvedActionId: "" })).toThrow(
      "approvedActionId"
    );
    expect(stub.events).toHaveLength(0);
  });

  it("HumanApprovalEmitter emits correct type and payload", () => {
    const stub = makeStub();
    const emitter = new HumanApprovalEmitter(stub);
    emitter.emit({
      approvedActionId: "action-42",
      approverId: "bob",
      note: "LGTM",
    });
    expect(stub.events[0].type).toBe(HUMAN_APPROVAL);
    expect(stub.events[0].payload.approved_action_id).toBe("action-42");
    expect(stub.events[0].payload.approver_id).toBe("bob");
  });

  it("sessionId propagated from context into payload", () => {
    const stub = makeStub("sess-xyz");
    const emitter = new ToolCallEmitter(stub);
    emitter.emit({ toolName: "search" });
    expect(stub.events[0].payload.session_id).toBe("sess-xyz");
  });
});
