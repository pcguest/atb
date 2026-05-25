import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { BackgroundJobTracker } from "./background-job-tracker.js";
import { DataExportGate, DataExportDeniedError } from "./data-export-gate.js";
import { HumanOverrideGate, HumanOverrideDeniedError } from "./human-override-gate.js";
import { PolicyDecisionRecorder } from "./policy-decision-recorder.js";

function userTypes(bundle: Bundle): string[] {
  return bundle.records
    .filter((record) => record.event.type !== "atb.bundle.manifest")
    .map((record) => record.event.type);
}

describe("DataExportGate", () => {
  it("records data export profile event sequence", async () => {
    const bundle = new Bundle();
    const gate = new DataExportGate({ bundle, actorId: "actor-export" });

    await gate.run(
      {
        actionType: "export_data",
        targetResourceId: "dataset-1",
        intendedEffect: "export approved dataset",
        actionParameters: { format: "csv" },
        actionId: "act-export-1",
      },
      () => ({ rows: 42 })
    );

    expect(userTypes(bundle)).toEqual([
      "ai.request.received",
      "data.export.precommit",
      "ai.policy.decision",
      "data.export.executed",
      "ai.human.approval",
    ]);

    const precommit = bundle.records.find(
      (r) => r.event.type === "data.export.precommit"
    )?.event.data as Record<string, unknown>;
    expect(precommit.action_id).toBe("act-export-1");
  });

  it("denies export in enforce mode", async () => {
    const gate = new DataExportGate({
      mode: "enforce",
      policy: async () => ({ decision: "deny", reasonCodes: ["blocked"] }),
    });

    await expect(
      gate.run(
        {
          actionType: "export_data",
          targetResourceId: "dataset-1",
          intendedEffect: "export",
          actionParameters: {},
        },
        () => "ok"
      )
    ).rejects.toBeInstanceOf(DataExportDeniedError);
  });
});

describe("PolicyDecisionRecorder", () => {
  it("records request, precommit, and policy decision", () => {
    const bundle = new Bundle();
    const recorder = new PolicyDecisionRecorder({ bundle, actorId: "actor-pol" });

    const actionId = recorder.record(
      {
        actionType: "approve_change",
        actionParameters: { ticket: "INC-1" },
        actionId: "act-pol-1",
      },
      { decision: "allow", reasonCodes: ["approved"] }
    );

    expect(actionId).toBe("act-pol-1");
    expect(userTypes(bundle)).toEqual([
      "ai.request.received",
      "ai.action.precommit",
      "ai.policy.decision",
    ]);
  });
});

describe("HumanOverrideGate", () => {
  it("records approval before precommit and execution", async () => {
    const bundle = new Bundle();
    const gate = new HumanOverrideGate({ bundle, actorId: "actor-override" });

    await gate.run(
      {
        actionType: "override_action",
        targetResourceId: "svc-1",
        intendedEffect: "run override",
        actionParameters: { mode: "manual" },
        actionId: "act-override-1",
      },
      {
        approverIdHash: "sha256:approver-1",
        approvalOutcome: "approved",
      },
      () => "done"
    );

    const types = userTypes(bundle);
    expect(types.indexOf("ai.human.approval")).toBeLessThan(
      types.indexOf("ai.action.precommit")
    );
    expect(types).toContain("ai.action.executed");
  });

  it("blocks denied overrides in enforce mode", async () => {
    const gate = new HumanOverrideGate({ mode: "enforce" });

    await expect(
      gate.run(
        {
          actionType: "override_action",
          targetResourceId: "svc-1",
          intendedEffect: "run override",
          actionParameters: {},
        },
        {
          approverIdHash: "sha256:approver-1",
          approvalOutcome: "denied",
        },
        () => "done"
      )
    ).rejects.toBeInstanceOf(HumanOverrideDeniedError);
  });
});

describe("BackgroundJobTracker", () => {
  it("records scheduled, started, and completed events", async () => {
    const bundle = new Bundle();
    const tracker = new BackgroundJobTracker({ bundle });

    await tracker.runJob(
      {
        jobType: "nightly_sync",
        triggerSource: "cron",
        scheduledByIdHash: "sha256:scheduler",
        jobId: "job-1",
      },
      "sha256:worker",
      async () => "ok"
    );

    expect(userTypes(bundle)).toEqual([
      "ai.job.scheduled",
      "ai.job.started",
      "ai.job.completed",
    ]);
  });
});
