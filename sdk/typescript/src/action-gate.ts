import { createHash, randomUUID } from "node:crypto";
import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import { normalizeOptionalIdentity } from "./event.js";

const DEFAULT_SAVE_PATH = "run.atb/bundle.atb";

/** Action gate operating mode. */
export type ActionGateMode = "log_only" | "enforce";

/** Input metadata recorded before a gated action executes. */
export interface ActionGateInput {
  actionType: string;
  targetResourceId: string;
  intendedEffect: string;
  actionParameters: Record<string, unknown>;
  subjectId?: string;
  actionId?: string;
  policyContext?: Record<string, unknown>;
}

/** Policy decision returned by an action gate policy callback. */
export interface ActionGateDecision {
  decision: "allow" | "deny";
  reasonCodes?: string[];
  policyId?: string;
  policyVersion?: string;
}

/** Constructor options for {@link ActionGate}. */
export interface ActionGateOptions {
  bundle?: Bundle;
  mode?: ActionGateMode;
  policy?: (
    action: ActionGateInput
  ) => ActionGateDecision | Promise<ActionGateDecision>;
  autoSave?: boolean;
  savePath?: string;
  actorId?: string;
  orgId?: string;
  workspaceId?: string;
}

/** Error thrown when a gated action is denied in enforce mode. */
export class ActionGateDeniedError extends Error {
  /**
   * @param message Human-readable denial message.
   * @returns A new denial error.
   */
  constructor(message: string) {
    super(message);
    this.name = "ActionGateDeniedError";
  }
}

/** Records precommit, policy, and execution events around local actions. */
export class ActionGate {
  readonly bundle: Bundle;

  private readonly mode: ActionGateMode;
  private readonly policy: (
    action: ActionGateInput
  ) => ActionGateDecision | Promise<ActionGateDecision>;
  private readonly autoSave: boolean;
  private readonly savePath: string;
  private readonly actorId?: string;
  private readonly orgId?: string;
  private readonly workspaceId?: string;

  /**
   * @param options Gate configuration.
   * @returns A new action gate.
   * @throws Error when `options.mode` is not recognised.
   */
  constructor(options: ActionGateOptions = {}) {
    this.bundle = options.bundle ?? new Bundle();
    this.mode = options.mode ?? "log_only";
    this.policy =
      options.policy ??
      (() => ({
        decision: "allow",
        policyId: "local.action_gate",
        policyVersion: "v1",
        reasonCodes: [],
      }));
    this.autoSave = options.autoSave ?? false;
    this.savePath = options.savePath ?? DEFAULT_SAVE_PATH;
    this.actorId = options.actorId;
    this.orgId = options.orgId;
    this.workspaceId = options.workspaceId;

    if (!["log_only", "enforce"].includes(this.mode)) {
      throw new Error("mode must be one of: log_only, enforce");
    }
  }

  /**
   * @param action Action metadata to record and evaluate.
   * @param fn Operation to execute when allowed.
   * @returns The value returned by `fn`.
   * @throws ActionGateDeniedError when policy denies in enforce mode.
   * @throws Re-throws any error produced by `fn`.
   */
  async run<T>(
    action: ActionGateInput,
    fn: () => T | Promise<T>
  ): Promise<T> {
    const actionId = this.actionId(action);
    this.emit("ai.action.precommit", {
      action_id: actionId,
      action_type: action.actionType,
      action_parameters_digest: canonicalDigest(action.actionParameters),
      target_resource_id: action.targetResourceId,
      intended_effect: action.intendedEffect,
    });

    const rawDecision = await this.policy(action);
    const decision = normalizeDecision(rawDecision);
    this.emit("ai.policy.decision", this.policyPayload(action, actionId, decision));
    if (decision.decision === "deny" && this.mode === "enforce") {
      throw new ActionGateDeniedError(
        `action denied by policy for action_id ${actionId}`
      );
    }

    const startedAt = Date.now();
    try {
      const result = await fn();
      this.emit(
        "ai.action.executed",
        this.executedPayload(action, actionId, startedAt, result, "success")
      );
      return result;
    } catch (error) {
      this.emit(
        "ai.action.executed",
        this.executedPayload(action, actionId, startedAt, error, "error")
      );
      throw error;
    }
  }

  private actionId(action: ActionGateInput): string {
    if (action.actionId && action.actionId.trim() !== "") {
      return action.actionId.trim();
    }
    return `act_${randomUUID().replace(/-/g, "")}`;
  }

  private emit(eventType: string, payload: Record<string, unknown>): void {
    this.bundle.append(eventType, payload, {
      actorId: this.actorId,
      orgId: this.orgId,
      workspaceId: this.workspaceId,
      timestamp: nowRFC3339(),
    });
    if (this.autoSave) {
      this.bundle.save(this.savePath);
    }
  }

  private policyPayload(
    action: ActionGateInput,
    actionId: string,
    decision: Required<ActionGateDecision>
  ): Record<string, unknown> {
    const payload: Record<string, unknown> = {
      decision_id: actionId,
      action_id: actionId,
      policy_id: decision.policyId,
      policy_version: decision.policyVersion,
      decision: decision.decision,
      decision_reason_codes: decision.reasonCodes,
    };
    const subjectIdHash = subjectHash(action.subjectId, this.actorId);
    if (subjectIdHash !== undefined) {
      payload.subject_id_hash = subjectIdHash;
    }
    return payload;
  }

  private executedPayload(
    action: ActionGateInput,
    actionId: string,
    startedAt: number,
    receipt: unknown,
    executionOutcome: "success" | "error"
  ): Record<string, unknown> {
    return {
      action_id: actionId,
      action_type: action.actionType,
      tool_receipt_digest: valueDigest(receipt),
      execution_duration_ms: Math.max(0, Date.now() - startedAt),
      execution_outcome: executionOutcome,
    };
  }
}

function normalizeDecision(
  decision: ActionGateDecision
): Required<ActionGateDecision> {
  return {
    decision: decision.decision,
    reasonCodes: decision.reasonCodes ?? [],
    policyId: decision.policyId ?? "local.action_gate",
    policyVersion: decision.policyVersion ?? "v1",
  };
}

function subjectHash(
  subjectId: string | undefined,
  actorId: string | undefined
): string | undefined {
  const candidate =
    normalizeOptionalIdentity(subjectId) ?? normalizeOptionalIdentity(actorId);
  if (candidate === undefined) {
    return undefined;
  }
  return sha256(Buffer.from(candidate, "utf8"));
}

function canonicalDigest(value: unknown): string {
  return sha256(Buffer.from(canonicalize(value), "utf8"));
}

function valueDigest(value: unknown): string {
  try {
    return sha256(Buffer.from(canonicalize(value), "utf8"));
  } catch {
    return sha256(Buffer.from(String(value), "utf8"));
  }
}

function sha256(value: Uint8Array): string {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

function nowRFC3339(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}
