import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import {
  ActionGate,
  ActionGateDeniedError,
  type ActionGateInput,
} from "./action-gate.js";

function userTypes(bundle: Bundle): string[] {
  return bundle.records
    .filter((record) => record.event.type !== "atb.bundle.manifest")
    .map((record) => record.event.type);
}

function makeAction(input: Partial<ActionGateInput> = {}): ActionGateInput {
  return {
    actionType: "deploy_change",
    targetResourceId: "svc-prod",
    intendedEffect: "roll out release",
    actionParameters: { version: "0.9.2-beta" },
    ...input,
  };
}

describe("ActionGate", () => {
  it("records precommit before handler executes", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({ bundle, actorId: "actor-1" });
    let seenBeforeExecution: string[] = [];

    await gate.run(makeAction(), () => {
      seenBeforeExecution = userTypes(bundle);
      return "ok";
    });

    expect(seenBeforeExecution).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
    ]);
  });

  it("records policy decision and executed outcome after success", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({
      bundle,
      actorId: "actor-1",
      policy: async () => ({
        decision: "allow",
        reasonCodes: ["policy_pass"],
        policyId: "policy.allow",
        policyVersion: "2026-04-01",
      }),
    });

    const result = await gate.run(makeAction(), async () => ({ status: "done" }));

    expect(result).toEqual({ status: "done" });
    const events = bundle.records
      .filter((record) => record.event.type !== "atb.bundle.manifest")
      .map((record) => record.event);
    expect(events.map((event) => event.type)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
      "ai.action.executed",
    ]);

    const precommit = events[0].data as Record<string, unknown>;
    expect(precommit.action_id).toMatch(/^act_/);
    expect(precommit.action_type).toBe("deploy_change");
    expect(precommit.action_parameters_digest).toMatch(/^sha256:/);
    expect(precommit.target_resource_id).toBe("svc-prod");
    expect(precommit.intended_effect).toBe("roll out release");

    const policy = events[1].data as Record<string, unknown>;
    expect(policy.decision_id).toBe(precommit.action_id);
    expect(policy.action_id).toBe(precommit.action_id);
    expect(policy.policy_id).toBe("policy.allow");
    expect(policy.policy_version).toBe("2026-04-01");
    expect(policy.decision).toBe("allow");
    expect(policy.decision_reason_codes).toEqual(["policy_pass"]);
    expect(policy.subject_id_hash).toMatch(/^sha256:/);

    const executed = events[2].data as Record<string, unknown>;
    expect(executed.action_id).toBe(precommit.action_id);
    expect(executed.action_type).toBe("deploy_change");
    expect(executed.tool_receipt_digest).toMatch(/^sha256:/);
    expect(executed.execution_duration_ms).toBeGreaterThanOrEqual(0);
    expect(executed.execution_outcome).toBe("success");
  });

  it("enforce mode rejects and does not call handler when policy denies", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({
      bundle,
      mode: "enforce",
      actorId: "actor-1",
      policy: () => ({ decision: "deny", reasonCodes: ["blocked"] }),
    });
    let called = false;

    await expect(
      gate.run(makeAction({ actionType: "delete_records" }), async () => {
        called = true;
        return "should-not-run";
      })
    ).rejects.toBeInstanceOf(ActionGateDeniedError);

    expect(called).toBe(false);
    const events = bundle.records
      .filter((record) => record.event.type !== "atb.bundle.manifest")
      .map((record) => record.event);
    expect(events.map((event) => event.type)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
    ]);
    expect((events[1].data as Record<string, unknown>).decision).toBe("deny");
  });

  it("log-only mode never blocks on deny", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({
      bundle,
      mode: "log_only",
      actorId: "actor-1",
      policy: () => ({ decision: "deny", reasonCodes: ["observe_only"] }),
    });
    let called = false;

    const result = await gate.run(makeAction(), async () => {
      called = true;
      return "ran";
    });

    expect(result).toBe("ran");
    expect(called).toBe(true);
    const events = bundle.records
      .filter((record) => record.event.type !== "atb.bundle.manifest")
      .map((record) => record.event);
    expect(events.map((event) => event.type)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
      "ai.action.executed",
    ]);
    expect((events[1].data as Record<string, unknown>).decision).toBe("deny");
    expect((events[2].data as Record<string, unknown>).execution_outcome).toBe(
      "success"
    );
  });

  it("supports async handlers with the same event ordering", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({ bundle, actorId: "actor-1" });
    let seenBeforeExecution: string[] = [];

    const result = await gate.run(makeAction(), async () => {
      seenBeforeExecution = userTypes(bundle);
      return Promise.resolve({ status: "async-ok" });
    });

    expect(result).toEqual({ status: "async-ok" });
    expect(seenBeforeExecution).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
    ]);
    expect(userTypes(bundle)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
      "ai.action.executed",
    ]);
  });

  it("records ai.action.error when the gated action throws", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({ bundle, actorId: "actor-1" });

    await expect(
      gate.run(makeAction({ actionType: "delete_records" }), async () => {
        throw new Error("sink refused");
      })
    ).rejects.toThrow("sink refused");

    expect(userTypes(bundle)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
      "ai.action.error",
    ]);

    const events = bundle.records
      .filter((record) => record.event.type !== "atb.bundle.manifest")
      .map((record) => record.event);
    const precommit = events[0].data as Record<string, unknown>;
    const error = events[2].data as Record<string, unknown>;
    expect(error.action_id).toBe(precommit.action_id);
    expect(error.error_class).toBe("exception");
    expect(error.error_detail_digest).toMatch(/^sha256:/);
  });
});
