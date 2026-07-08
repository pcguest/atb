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

  it("records the acting principal on precommit when provided", async () => {
    const bundle = new Bundle();
    const gate = new HumanOverrideGate({ bundle, actorId: "actor-1" });

    await gate.run(
      { ...action, principal: { type: "human", idHash: "sha256:op-1" } },
      approval,
      async () => "ok",
    );

    const precommit = bundle.records
      .map((record) => record.event)
      .find((event) => event.type === "ai.action.precommit")!.data as Record<string, unknown>;
    expect(precommit.principal).toEqual({ type: "human", id_hash: "sha256:op-1" });
  });

  it("records digest-only reviewer identity evidence on approval", async () => {
    const bundle = new Bundle();
    const gate = new HumanOverrideGate({ bundle, actorId: "actor-1" });

    await gate.run(
      action,
      {
        ...approval,
        identityEvidence: {
          identityProvider: "https://idp.example",
          subject: "reviewer-1",
          assertionType: "jwt",
          assertionDigest: "sha256:assertion",
        },
      },
      async () => "ok",
    );

    const event = bundle.records
      .map((record) => record.event)
      .find((candidate) => candidate.type === "ai.human.approval")!;
    expect((event.data as Record<string, unknown>).identity_evidence).toEqual({
      identity_provider: "https://idp.example",
      subject: "reviewer-1",
      assertion_type: "jwt",
      assertion_digest: "sha256:assertion",
    });
  });
});
