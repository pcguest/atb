import {
  WorkflowContext,
  type WorkflowContextOptions,
  canonicalDigest,
  newActionId,
  newApprovalId,
  valueDigest,
} from "./workflow-common.js";

export type HumanOverrideGateMode = "log_only" | "enforce";

export interface HumanOverrideActionInput {
  actionType: string;
  targetResourceId: string;
  intendedEffect: string;
  actionParameters: Record<string, unknown>;
  actionId?: string;
  requestId?: string;
}

export interface HumanOverrideApprovalInput {
  approverIdHash: string;
  approvalOutcome?: "approved" | "denied";
  justificationDigest?: string;
  approvalId?: string;
}

export interface HumanOverrideGateOptions extends WorkflowContextOptions {
  mode?: HumanOverrideGateMode;
}

export class HumanOverrideDeniedError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "HumanOverrideDeniedError";
  }
}

/** Records human-override profile events around a gated action. */
export class HumanOverrideGate {
  readonly ctx: WorkflowContext;
  private readonly mode: HumanOverrideGateMode;

  constructor(options: HumanOverrideGateOptions = {}) {
    this.ctx = new WorkflowContext(options);
    this.mode = options.mode ?? "log_only";
    if (!["log_only", "enforce"].includes(this.mode)) {
      throw new Error("mode must be one of: log_only, enforce");
    }
  }

  get bundle() {
    return this.ctx.bundle;
  }

  async run<T>(
    action: HumanOverrideActionInput,
    approval: HumanOverrideApprovalInput,
    fn: () => T | Promise<T>
  ): Promise<T> {
    this.ctx.bootstrapRequest("human_override", action.requestId);
    const actionId = newActionId(action.actionId);
    const outcome = approval.approvalOutcome ?? "approved";

    this.ctx.emit("ai.human.approval", {
      approval_id: newApprovalId(approval.approvalId),
      approver_id_hash: approval.approverIdHash,
      approval_outcome: outcome,
      justification_digest:
        approval.justificationDigest ?? canonicalDigest({ reason: "human override" }),
      action_id: actionId,
    });

    this.ctx.emit("ai.action.precommit", {
      action_id: actionId,
      action_type: action.actionType,
      action_parameters_digest: canonicalDigest(action.actionParameters),
      target_resource_id: action.targetResourceId,
      intended_effect: action.intendedEffect,
    });

    if (outcome === "denied" && this.mode === "enforce") {
      throw new HumanOverrideDeniedError(
        `human override denied for action_id ${actionId}`
      );
    }

    const startedAt = Date.now();
    try {
      const result = await fn();
      this.ctx.emit("ai.action.executed", {
        action_id: actionId,
        execution_outcome: "success",
        tool_receipt_digest: valueDigest(result),
        execution_duration_ms: Math.max(0, Date.now() - startedAt),
      });
      return result;
    } catch (error) {
      this.ctx.emit("ai.action.executed", {
        action_id: actionId,
        execution_outcome: "error",
        tool_receipt_digest: valueDigest(error),
        execution_duration_ms: Math.max(0, Date.now() - startedAt),
      });
      throw error;
    }
  }
}
