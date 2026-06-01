import { describe, expect, it } from "vitest";
import { Bundle } from "./bundle.js";
import { DataExportGate, type DataExportInput } from "./data-export-gate.js";

function userTypes(bundle: Bundle): string[] {
  return bundle.records
    .filter((record) => record.event.type !== "atb.bundle.manifest")
    .map((record) => record.event.type);
}

const action: DataExportInput = {
  actionType: "export_dataset",
  targetResourceId: "dataset-1",
  intendedEffect: "export to s3",
  actionParameters: { rows: 100 },
};

describe("DataExportGate", () => {
  it("records data.export.error when the export throws", async () => {
    const bundle = new Bundle();
    const gate = new DataExportGate({ bundle, actorId: "actor-1", recordApproval: false });

    await expect(
      gate.run(action, async () => {
        throw new Error("sink refused");
      })
    ).rejects.toThrow("sink refused");

    const types = userTypes(bundle);
    expect(types).toContain("data.export.error");
    expect(types).not.toContain("data.export.executed");

    const error = bundle.records
      .filter((r) => r.event.type === "data.export.error")
      .map((r) => r.event.data)[0] as Record<string, unknown>;
    expect(error.error_class).toBe("exception");
    expect(String(error.error_detail_digest)).toMatch(/^sha256:/);
  });
});
