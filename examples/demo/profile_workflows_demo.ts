/**
 * Profile workflow SDK demo — data export ops on a fresh bundle.
 *
 * Run from repository root:
 *   npx tsx examples/demo/profile_workflows_demo.ts
 *
 * Requires `atb` on PATH (or set ATB_BIN).
 */
import { spawnSync } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { Bundle } from "../../sdk/typescript/src/bundle.js";
import { BackgroundJobTracker } from "../../sdk/typescript/src/background-job-tracker.js";
import { DataExportGate } from "../../sdk/typescript/src/data-export-gate.js";
import { HumanOverrideGate } from "../../sdk/typescript/src/human-override-gate.js";
import { PolicyDecisionRecorder } from "../../sdk/typescript/src/policy-decision-recorder.js";

function atbBin(): string {
  return process.env.ATB_BIN ?? "atb";
}

function verifyProfile(
  bundlePath: string,
  profileId: string,
  required = true,
): Record<string, unknown> {
  const proc = spawnSync(
    atbBin(),
    ["verify", "--bundle", bundlePath, "--profile", profileId, "--format", "json"],
    { encoding: "utf8" },
  );
  if (proc.status !== 0 && required) {
    throw new Error(proc.stderr || proc.stdout || "atb verify failed");
  }
  if (!proc.stdout.trim()) {
    throw new Error(proc.stderr || "atb verify returned no output");
  }
  return JSON.parse(proc.stdout) as Record<string, unknown>;
}

async function main(): Promise<number> {
  const work = mkdtempSync(join(tmpdir(), "atb-profile-demo-"));
  const bundlePath = join(work, "bundle.atb");
  const exportCheckpoint = join(work, "export-phase.atb");

  const bundle = new Bundle();
  const jobTracker = new BackgroundJobTracker({ bundle, actorId: "scheduler-ops" });
  const exportGate = new DataExportGate({ bundle, actorId: "export-runner", recordApproval: true });

  await jobTracker.runJob(
    {
      jobType: "weekly_analytics_export",
      triggerSource: "cron",
      scheduledByIdHash: "sha256:scheduler-nightly",
      jobId: "job-export-demo",
    },
    "sha256:worker-export-01",
    async () =>
      exportGate.run(
        {
          actionType: "export_analytics",
          targetResourceId: "warehouse.analytics",
          intendedEffect: "export weekly rollup",
          actionParameters: { format: "parquet", rows: 1200 },
          actionId: "act-scheduled-export",
        },
        () => ({ rows: 1200, bytes: 450_000 }),
      ),
  );

  bundle.save(exportCheckpoint);
  const exportReport = verifyProfile(exportCheckpoint, "atb.profile.data_export");

  const policyRecorder = new PolicyDecisionRecorder({ bundle, actorId: "policy-engine" });
  policyRecorder.record(
    {
      actionType: "approve_retention_exception",
      actionParameters: { ticket: "LEGAL-88", dataset: "warehouse.pii_requests" },
      actionId: "act-retention-policy",
    },
    { decision: "allow", reasonCodes: ["legal_hold_active"] },
  );

  const overrideGate = new HumanOverrideGate({ bundle, actorId: "ops-lead" });
  await overrideGate.run(
    {
      actionType: "export_pii_subset",
      targetResourceId: "warehouse.pii_requests",
      intendedEffect: "legal hold export",
      actionParameters: { caseId: "LEGAL-88" },
      actionId: "act-override-export",
    },
    {
      approverIdHash: "sha256:legal-lead",
      approvalOutcome: "approved",
      justificationDigest: "sha256:legal-hold-justification",
    },
    () => ({ rows: 14 }),
  );

  bundle.save(bundlePath);
  const policyReport = verifyProfile(bundlePath, "atb.profile.policy_decision", false);
  const overrideReport = verifyProfile(bundlePath, "atb.profile.human_override", false);

  console.log(
    JSON.stringify(
      {
        data_export: {
          pass: exportReport.pass,
          cas_grade: exportReport.cas_grade,
          profile_id: exportReport.profile_id,
        },
        policy_decision: { pass: policyReport.pass, cas_grade: policyReport.cas_grade },
        human_override: { pass: overrideReport.pass, cas_grade: overrideReport.cas_grade },
      },
      null,
      2,
    ),
  );

  if (exportReport.pass !== true) {
    console.error("data_export verify failed");
    return 1;
  }

  bundle.verify();
  console.log(`bundle: ${bundlePath}`);
  return 0;
}

main()
  .then((code) => {
    if (code !== 0) process.exit(code);
  })
  .catch((err: unknown) => {
    console.error(err);
    process.exit(1);
  });
