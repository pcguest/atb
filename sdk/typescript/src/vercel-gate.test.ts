import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { ActionGate, ActionGateDeniedError } from "./action-gate.js";
import { gateVercelTool } from "./vercel-gate.js";

function userTypes(bundle: Bundle): string[] {
  return bundle.records
    .filter((record) => record.event.type !== "atb.bundle.manifest")
    .map((record) => record.event.type);
}

describe("gateVercelTool", () => {
  it("gates vercel tool and records events in order", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({ bundle, actorId: "actor-1" });
    const tool = gateVercelTool(
      {
        name: "deploy_change",
        description: "roll out release",
        parameters: { type: "object" },
        execute: async (parameters: { version: string }) => ({ result: parameters.version }),
      },
      gate,
    );

    const result = await tool.execute({ version: "0.9.0-beta" });

    expect(result).toEqual({ result: "0.9.0-beta" });
    expect(userTypes(bundle)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
      "ai.action.executed",
    ]);
  });

  it("enforce mode blocks when policy denies", async () => {
    const bundle = new Bundle();
    const gate = new ActionGate({
      bundle,
      mode: "enforce",
      actorId: "actor-1",
      policy: () => ({ decision: "deny", reasonCodes: ["blocked"] }),
    });
    const tool = gateVercelTool(
      {
        name: "deploy_change",
        description: "roll out release",
        parameters: { type: "object" },
        execute: async (parameters: { version: string }) => ({ result: parameters.version }),
      },
      gate,
    );

    await expect(tool.execute({ version: "0.9.0-beta" })).rejects.toBeInstanceOf(ActionGateDeniedError);
    expect(userTypes(bundle)).toEqual([
      "ai.action.precommit",
      "ai.policy.decision",
    ]);
  });
});
