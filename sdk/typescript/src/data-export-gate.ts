import {
  WorkflowContext,
  type WorkflowContextOptions,
  type WorkflowPolicyDecision,
  actorIdHash,
  canonicalDigest,
  newActionId,
  newApprovalId,
  valueDigest,
} from "./workflow-common.js";
import { identityEvidencePayload, type IdentityEvidence } from "./identity-evidence.js";

export type DataExportGateMode = "log_only" | "enforce";

/** Input metadata for a gated data export. */
export interface DataExportInput {
  actionType: string;
  targetResourceId: string;
  intendedEffect: string;
  actionParameters: Record<string, unknown>;
  subjectId?: string;
  actionId?: string;
  requestId?: string;
}

/** Human approval metadata for export workflows. */
export interface DataExportApproval {
  approverIdHash: string;
  approvalOutcome?: "approved" | "denied";
  justificationDigest?: string;
  approvalId?: string;
  /** Advanced: digest-only caller-provided reviewer identity evidence. */
  identityEvidence?: IdentityEvidence;
}

export interface DataExportGateOptions extends WorkflowContextOptions {
  mode?: DataExportGateMode;
  policy?: (
    exportAction: DataExportInput
  ) => WorkflowPolicyDecision | Promise<WorkflowPolicyDecision>;
  /** When true, emit ai.human.approval after execution (required by profile when executed). */
  recordApproval?: boolean | ((exportAction: DataExportInput) => DataExportApproval);
}

export class DataExportDeniedError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "DataExportDeniedError";
  }
}

/** Records data export profile events around an export operation. */
export class DataExportGate {
  readonly ctx: WorkflowContext;
  private readonly mode: DataExportGateMode;
  private readonly actorId?: string;
  private readonly policy: (
    exportAction: DataExportInput
  ) => WorkflowPolicyDecision | Promise<WorkflowPolicyDecision>;
  private readonly recordApproval:
    | boolean
    | ((exportAction: DataExportInput) => DataExportApproval);

  constructor(options: DataExportGateOptions = {}) {
    this.ctx = new WorkflowContext(options);
    this.actorId = options.actorId;
    this.mode = options.mode ?? "log_only";
    this.policy =
      options.policy ??
      (() => ({
        decision: "allow",
        policyId: "local.data_export",
        policyVersion: "v1",
        reasonCodes: [],
      }));
    this.recordApproval = options.recordApproval ?? true;
    if (!["log_only", "enforce"].includes(this.mode)) {
      throw new Error("mode must be one of: log_only, enforce");
    }
  }

  get bundle() {
    return this.ctx.bundle;
  }

  async run<T>(
    exportAction: DataExportInput,
    fn: () => T | Promise<T>
  ): Promise<T> {
    this.ctx.bootstrapRequest("data_export", exportAction.requestId);
    const actionId = newActionId(exportAction.actionId);

    this.ctx.emit("data.export.precommit", {
      action_id: actionId,
      action_type: exportAction.actionType,
      action_parameters_digest: canonicalDigest(exportAction.actionParameters),
      target_resource_id: exportAction.targetResourceId,
      intended_effect: exportAction.intendedEffect,
    });

    const decision = await this.policy(exportAction);
    this.ctx.emit(
      "ai.policy.decision",
      this.ctx.policyPayload(actionId, decision, exportAction.subjectId)
    );
    if (decision.decision === "deny" && this.mode === "enforce") {
      throw new DataExportDeniedError(
        `data export denied by policy for action_id ${actionId}`
      );
    }

    const startedAt = Date.now();
    try {
      const result = await fn();
      this.ctx.emit("data.export.executed", {
        action_id: actionId,
        execution_outcome: "success",
        tool_receipt_digest: valueDigest(result),
        execution_duration_ms: Math.max(0, Date.now() - startedAt),
      });
      this.maybeRecordApproval(exportAction, actionId);
      return result;
    } catch (error) {
      // A data export that threw did not complete: record the forensic
      // data.export.error event, not a success-shaped executed record.
      this.ctx.emit("data.export.error", {
        action_id: actionId,
        error_class: "exception",
        error_detail_digest: valueDigest(error),
      });
      this.maybeRecordApproval(exportAction, actionId);
      throw error;
    }
  }

  private maybeRecordApproval(
    exportAction: DataExportInput,
    actionId: string
  ): void {
    if (this.recordApproval === false) {
      return;
    }
    const approval =
      typeof this.recordApproval === "function"
        ? this.recordApproval(exportAction)
        : {
            approverIdHash: actorIdHash(this.actorId),
            approvalOutcome: "approved" as const,
            justificationDigest: canonicalDigest({ reason: "export approved" }),
          };
    const payload: Record<string, unknown> = {
      approval_id: newApprovalId(approval.approvalId),
      approver_id_hash: approval.approverIdHash,
      approval_outcome: approval.approvalOutcome ?? "approved",
      justification_digest:
        approval.justificationDigest ?? canonicalDigest({ reason: "export approved" }),
      action_id: actionId,
    };
    const identityEvidence = identityEvidencePayload(approval.identityEvidence);
    if (identityEvidence) {
      payload.identity_evidence = identityEvidence;
    }
    this.ctx.emit("ai.human.approval", payload);
  }
}
