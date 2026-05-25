import { createHash, randomUUID } from "node:crypto";
import { Bundle } from "./bundle.js";
import { canonicalize } from "./canonicalize.js";
import { normalizeOptionalIdentity } from "./event.js";

export const DEFAULT_WORKFLOW_SAVE_PATH = "run.atb/bundle.atb";

/** Optional sink that receives workflow events instead of local bundle I/O. */
export interface WorkflowEventSink {
  append(eventType: string, payload: Record<string, unknown>): void;
}

/** Shared emitter options for profile workflow helpers. */
export interface WorkflowContextOptions {
  bundle?: Bundle;
  autoSave?: boolean;
  savePath?: string;
  actorId?: string;
  orgId?: string;
  workspaceId?: string;
  /** When set, events are sent to the local ATB Agent instead of the bundle file. */
  eventSink?: WorkflowEventSink;
}

/** Policy decision shape shared by workflow gates. */
export interface WorkflowPolicyDecision {
  decision: "allow" | "deny";
  reasonCodes?: string[];
  policyId?: string;
  policyVersion?: string;
}

/** Appends profile events with optional auto-save. */
export class WorkflowContext {
  readonly bundle: Bundle;

  private readonly autoSave: boolean;
  private readonly savePath: string;
  private readonly actorId?: string;
  private readonly orgId?: string;
  private readonly workspaceId?: string;
  private readonly eventSink?: WorkflowEventSink;
  private requestBootstrapped = false;

  constructor(options: WorkflowContextOptions = {}) {
    this.bundle = options.bundle ?? new Bundle();
    this.autoSave = options.autoSave ?? false;
    this.savePath = options.savePath ?? DEFAULT_WORKFLOW_SAVE_PATH;
    this.actorId = options.actorId;
    this.orgId = options.orgId;
    this.workspaceId = options.workspaceId;
    this.eventSink = options.eventSink;
  }

  /** Emit one event to the bundle. */
  emit(eventType: string, payload: Record<string, unknown>): void {
    if (this.eventSink) {
      this.eventSink.append(eventType, payload);
      return;
    }
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

  /** Emit `ai.request.received` once per context when absent. */
  bootstrapRequest(purposeTag: string, requestId?: string): string {
    const rid = requestId?.trim() || `req_${randomUUID().replace(/-/g, "")}`;
    if (!this.requestBootstrapped) {
      this.emit("ai.request.received", {
        request_id: rid,
        actor_id_hash: actorIdHash(this.actorId),
        purpose_tag: purposeTag,
      });
      this.requestBootstrapped = true;
    }
    return rid;
  }

  /** Build a normalized policy decision payload. */
  policyPayload(
    actionId: string,
    decision: WorkflowPolicyDecision,
    subjectId?: string
  ): Record<string, unknown> {
    const normalized = normalizeWorkflowDecision(decision);
    const payload: Record<string, unknown> = {
      decision_id: actionId,
      action_id: actionId,
      policy_id: normalized.policyId,
      policy_version: normalized.policyVersion,
      decision: normalized.decision,
      decision_reason_codes: normalized.reasonCodes,
    };
    const subjectIdHash = subjectHash(subjectId, this.actorId);
    if (subjectIdHash !== undefined) {
      payload.subject_id_hash = subjectIdHash;
    }
    return payload;
  }
}

export function newActionId(actionId?: string): string {
  if (actionId && actionId.trim() !== "") {
    return actionId.trim();
  }
  return `act_${randomUUID().replace(/-/g, "")}`;
}

export function newJobId(jobId?: string): string {
  if (jobId && jobId.trim() !== "") {
    return jobId.trim();
  }
  return `job_${randomUUID().replace(/-/g, "")}`;
}

export function newApprovalId(approvalId?: string): string {
  if (approvalId && approvalId.trim() !== "") {
    return approvalId.trim();
  }
  return `appr_${randomUUID().replace(/-/g, "")}`;
}

export function canonicalDigest(value: unknown): string {
  return sha256(Buffer.from(canonicalize(value), "utf8"));
}

export function valueDigest(value: unknown): string {
  try {
    return sha256(Buffer.from(canonicalize(value), "utf8"));
  } catch {
    return sha256(Buffer.from(String(value), "utf8"));
  }
}

export function subjectHash(
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

export function actorIdHash(actorId: string | undefined): string {
  return subjectHash(undefined, actorId) ?? sha256(Buffer.from("unknown", "utf8"));
}

export function normalizeWorkflowDecision(
  decision: WorkflowPolicyDecision
): Required<WorkflowPolicyDecision> {
  return {
    decision: decision.decision,
    reasonCodes: decision.reasonCodes ?? [],
    policyId: decision.policyId ?? "local.workflow",
    policyVersion: decision.policyVersion ?? "v1",
  };
}

function sha256(value: Uint8Array): string {
  return `sha256:${createHash("sha256").update(value).digest("hex")}`;
}

function nowRFC3339(): string {
  return new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
}
