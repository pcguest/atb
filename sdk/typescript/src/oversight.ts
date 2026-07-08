import {
  DATA_EXPORT,
  HUMAN_APPROVAL,
  HUMAN_OVERRIDE,
  TOOL_CALL,
} from "./eventTypes.js";
import { identityEvidencePayload, type IdentityEvidence } from "./identity-evidence.js";

/** Compatible event sink for oversight emitters. */
export interface OversightEventSink {
  emit(type: string, payload: Record<string, unknown>): void;
  sessionId?: string;
  session_id?: string;
}

/** Options for {@link ToolCallEmitter.emit}. */
export interface ToolCallOptions {
  /** Name of the tool invoked. Required. */
  toolName: string;
  actorId?: string;
  /** SHA-256 hex of the canonicalised tool input. */
  inputDigest?: string;
  /** SHA-256 hex of the canonicalised tool output. */
  outputDigest?: string;
}

/** Options for {@link DataExportEmitter.emit}. */
export interface DataExportOptions {
  /** Identifier of the export destination. Required. */
  exportTarget: string;
  actorId?: string;
  recordCount?: number;
  classification?: string;
}

/** Options for {@link HumanOverrideEmitter.emit}. */
export interface HumanOverrideOptions {
  /** Short rationale for the override. Required. */
  overrideReason: string;
  actorId?: string;
  overriddenActionId?: string;
  identityEvidence?: IdentityEvidence;
}

/** Options for {@link HumanApprovalEmitter.emit}. */
export interface HumanApprovalOptions {
  /** Identifier of the action being approved. Required. */
  approvedActionId: string;
  actorId?: string;
  approverId?: string;
  note?: string;
  identityEvidence?: IdentityEvidence;
}

function requireNonEmpty(value: string, field: string): string {
  const trimmed = value?.trim() ?? "";
  if (!trimmed) {
    throw new Error(`atb: ${field} is required`);
  }
  return trimmed;
}

function sessionIdFromCtx(ctx: OversightEventSink): string | undefined {
  const raw = ctx.sessionId ?? ctx.session_id;
  if (raw === undefined || raw === null) {
    return undefined;
  }
  const trimmed = String(raw).trim();
  return trimmed !== "" ? trimmed : undefined;
}

function basePayload(ctx: OversightEventSink): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  const sid = sessionIdFromCtx(ctx);
  if (sid) {
    payload.session_id = sid;
  }
  return payload;
}

function optionalStr(
  payload: Record<string, unknown>,
  key: string,
  value?: string
): void {
  if (value !== undefined && value.trim() !== "") {
    payload[key] = value.trim();
  }
}

/**
 * Emits atb.tool.call events for direct tool invocations.
 *
 * Attach to a WorkflowContext (or compatible event sink) and call emit() once
 * per tool invocation to be recorded in the audit trail.
 */
export class ToolCallEmitter {
  constructor(private readonly ctx: OversightEventSink) {}

  /**
   * Emit an atb.tool.call event.
   *
   * @param opts Tool call metadata.
   * @throws {Error} If opts.toolName is empty.
   */
  emit(opts: ToolCallOptions): void {
    const payload = basePayload(this.ctx);
    payload.tool_name = requireNonEmpty(opts.toolName, "toolName");
    optionalStr(payload, "actor_id", opts.actorId);
    optionalStr(payload, "tool_input_digest", opts.inputDigest);
    optionalStr(payload, "tool_output_digest", opts.outputDigest);
    this.ctx.emit(TOOL_CALL, payload);
  }
}

/** Emits atb.data.export events for outbound data movements. */
export class DataExportEmitter {
  constructor(private readonly ctx: OversightEventSink) {}

  /**
   * Emit an atb.data.export event.
   *
   * @param opts Export metadata.
   * @throws {Error} If opts.exportTarget is empty.
   */
  emit(opts: DataExportOptions): void {
    const payload = basePayload(this.ctx);
    payload.export_target = requireNonEmpty(opts.exportTarget, "exportTarget");
    optionalStr(payload, "actor_id", opts.actorId);
    if (opts.recordCount !== undefined && opts.recordCount > 0) {
      payload.record_count = opts.recordCount;
    }
    optionalStr(payload, "classification", opts.classification);
    this.ctx.emit(DATA_EXPORT, payload);
  }
}

/**
 * Emits atb.human.override events when a human overrides an AI recommendation
 * or system guardrail.
 */
export class HumanOverrideEmitter {
  constructor(private readonly ctx: OversightEventSink) {}

  /**
   * Emit an atb.human.override event.
   *
   * @param opts Override metadata.
   * @throws {Error} If opts.overrideReason is empty.
   */
  emit(opts: HumanOverrideOptions): void {
    const payload = basePayload(this.ctx);
    payload.override_reason = requireNonEmpty(opts.overrideReason, "overrideReason");
    optionalStr(payload, "actor_id", opts.actorId);
    optionalStr(payload, "overridden_action_id", opts.overriddenActionId);
    const identityEvidence = identityEvidencePayload(opts.identityEvidence);
    if (identityEvidence) {
      payload.identity_evidence = identityEvidence;
    }
    this.ctx.emit(HUMAN_OVERRIDE, payload);
  }
}

/**
 * Emits atb.human.approval events when a human explicitly approves a pending
 * AI action.
 */
export class HumanApprovalEmitter {
  constructor(private readonly ctx: OversightEventSink) {}

  /**
   * Emit an atb.human.approval event.
   *
   * @param opts Approval metadata.
   * @throws {Error} If opts.approvedActionId is empty.
   */
  emit(opts: HumanApprovalOptions): void {
    const payload = basePayload(this.ctx);
    payload.approved_action_id = requireNonEmpty(
      opts.approvedActionId,
      "approvedActionId"
    );
    optionalStr(payload, "actor_id", opts.actorId);
    optionalStr(payload, "approver_id", opts.approverId);
    optionalStr(payload, "note", opts.note);
    const identityEvidence = identityEvidencePayload(opts.identityEvidence);
    if (identityEvidence) {
      payload.identity_evidence = identityEvidence;
    }
    this.ctx.emit(HUMAN_APPROVAL, payload);
  }
}
