import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import {
  HumanOverrideGate,
  type HumanOverrideActionInput,
  type HumanOverrideApprovalInput,
} from "./human-override-gate.js";

function userTypes(bundle: Bundle): string[] {
  return bundle.records
    .filter((record) => record.event.type !== "atb.bundle.manifest")
    .map((record) => record.event.type);
}

const action: HumanOverrideActionInput = {
  actionType: "delete_records",
  targetResourceId: "svc-prod",
  intendedEffect: "purge accounts",
  actionParameters: { scope: "all" },
};

const approval: HumanOverrideApprovalInput = {
  approverIdHash: "sha256:approver",
};

describe("HumanOverrideGate", () => {
  it("records ai.action.error when the overridden action throws", async () => {
    const bundle = new Bundle();
    const gate = new HumanOverrideGate({ bundle, actorId: "actor-1" });

    await expect(
      gate.run(action, approval, async () => {
        throw new Error("sink refused");
      })
    ).rejects.toThrow("sink refused");

    const types = userTypes(bundle);
    expect(types[types.length - 1]).toBe("ai.action.error");
    expect(types).not.toContain("ai.action.executed");

    const events = bundle.records
      .filter((record) => record.event.type !== "atb.bundle.manifest")
      .map((record) => record.event);
    const error = events[events.length - 1].data as Record<string, unknown>;
    expect(error.error_class).toBe("exception");
    expect(String(error.error_detail_digest)).toMatch(/^sha256:/);
  });
});
