import {
  WorkflowContext,
  type WorkflowContextOptions,
  type WorkflowPolicyDecision,
  canonicalDigest,
  newActionId,
} from "./workflow-common.js";

/** Pending action metadata bound to a policy decision. */
export interface PolicyDecisionActionInput {
  actionType: string;
  actionParameters: Record<string, unknown>;
  actionId?: string;
  subjectId?: string;
  requestId?: string;
}

export type PolicyDecisionRecorderOptions = WorkflowContextOptions;

/** Records policy-decision profile events without executing an action. */
export class PolicyDecisionRecorder {
  readonly ctx: WorkflowContext;

  constructor(options: PolicyDecisionRecorderOptions = {}) {
    this.ctx = new WorkflowContext(options);
  }

  get bundle() {
    return this.ctx.bundle;
  }

  record(
    action: PolicyDecisionActionInput,
    decision: WorkflowPolicyDecision
  ): string {
    this.ctx.bootstrapRequest("policy_decision", action.requestId);
    const actionId = newActionId(action.actionId);

    this.ctx.emit("ai.action.precommit", {
      action_id: actionId,
      action_type: action.actionType,
      action_parameters_digest: canonicalDigest(action.actionParameters),
    });
    this.ctx.emit(
      "ai.policy.decision",
      this.ctx.policyPayload(actionId, decision, action.subjectId)
    );
    return actionId;
  }
}
