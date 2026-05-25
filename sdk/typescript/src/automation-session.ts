/**
 * AutomationSession — workflow chaining harness for multi-hop AI capture.
 *
 * Option A (session wrapper): composes WorkflowContext, ActionGate, and
 * PolicyDecisionRecorder so callers open one bundle connection, emit events
 * across successive model/tool hops, and flush without per-call boilerplate.
 *
 * Scope: RAG answer paths, privileged tool actions (with commit), and policy
 * decisions. Non-goals: new event types, orchestration runtime, host capture.
 */
import { existsSync } from "node:fs";
import {
  ActionGate,
  type ActionGateInput,
  type ActionGateOptions,
} from "./action-gate.js";
import { Bundle } from "./bundle.js";
import { isCaptureEnvironment } from "./capture.js";
import {
  AI_ACTION_COMMITTED_EVENT_TYPE,
  AI_MODEL_INVOKED_EVENT_TYPE,
  AI_MODEL_OUTPUT_EVENT_TYPE,
  AI_RESPONSE_SENT_EVENT_TYPE,
  AI_RETRIEVAL_EXECUTED_EVENT_TYPE,
  SNAPSHOT_EVENT_TYPE,
} from "./eventTypes.js";
import {
  PolicyDecisionRecorder,
  type PolicyDecisionActionInput,
} from "./policy-decision-recorder.js";
import {
  DEFAULT_WORKFLOW_SAVE_PATH,
  WorkflowContext,
  type WorkflowContextOptions,
  type WorkflowPolicyDecision,
  canonicalDigest,
  newActionId,
  valueDigest,
} from "./workflow-common.js";

/** Options for {@link AutomationSession}. */
export interface AutomationSessionOptions extends WorkflowContextOptions {
  /** Default purpose tag for {@link AutomationSession.beginRequest}. */
  purposeTag?: string;
  /** Pre-seed the active request id for chained hops. */
  requestId?: string;
  /** Capture run id from ATB_CAPTURE_RUN_ID (stored in session metadata only). */
  captureRunId?: string;
  /** Action gate mode and policy when using {@link AutomationSession.runToolAction}. */
  toolGate?: Pick<ActionGateOptions, "mode" | "policy">;
}

/** Input for {@link AutomationSession.logModelInvocation}. */
export interface ModelInvocationInput {
  provider: string;
  model: string;
  prompt: unknown;
  parameters?: Record<string, unknown>;
  requestId?: string;
}

/** Input for {@link AutomationSession.logModelOutput}. */
export interface ModelOutputInput {
  output: unknown;
  outputFormat?: string;
  requestId?: string;
}

/** Input for {@link AutomationSession.logRetrieval}. */
export interface RetrievalInput {
  query: unknown;
  corpusId: string;
  corpusVersion: string;
  topK: number;
  resultSet: unknown;
  requestId?: string;
}

/** Input for {@link AutomationSession.logResponseSent}. */
export interface ResponseSentInput {
  output: unknown;
  outputFormat?: string;
  requestId?: string;
}

/** Options for {@link AutomationSession.close}. */
export interface CloseSessionOptions {
  snapshotName?: string;
  savePath?: string;
}

/**
 * Session-scoped harness for chained ATB event emission across AI workflow hops.
 */
export class AutomationSession {
  readonly ctx: WorkflowContext;
  readonly captureRunId?: string;

  private readonly defaultPurposeTag: string;
  private readonly savePath: string;
  private readonly toolGate: ActionGate;
  private readonly policyRecorder: PolicyDecisionRecorder;
  private activeRequestId?: string;

  constructor(options: AutomationSessionOptions = {}) {
    this.ctx = new WorkflowContext(options);
    this.captureRunId = options.captureRunId?.trim() || undefined;
    this.defaultPurposeTag = options.purposeTag?.trim() || "ai_workflow";
    this.savePath = options.savePath ?? DEFAULT_WORKFLOW_SAVE_PATH;
    this.activeRequestId = options.requestId?.trim() || undefined;

    this.toolGate = new ActionGate({
      bundle: this.ctx.bundle,
      autoSave: options.autoSave,
      savePath: this.savePath,
      actorId: options.actorId,
      orgId: options.orgId,
      workspaceId: options.workspaceId,
      mode: options.toolGate?.mode,
      policy: options.toolGate?.policy,
    });
    this.policyRecorder = new PolicyDecisionRecorder({
      bundle: this.ctx.bundle,
      autoSave: options.autoSave,
      savePath: this.savePath,
      actorId: options.actorId,
      orgId: options.orgId,
      workspaceId: options.workspaceId,
    });
  }

  /** Open a session with auto-save enabled (typical live capture default). */
  static open(options: AutomationSessionOptions = {}): AutomationSession {
    return new AutomationSession({ autoSave: true, ...options });
  }

  /**
   * Create or resume a session from atb capture run environment variables.
   * Returns null when capture env vars are absent.
   */
  static fromCaptureEnvironment(
    env: Record<string, string | undefined> = process.env
  ): AutomationSession | null {
    if (!isCaptureEnvironment(env)) {
      return null;
    }
    const savePath = env.ATB_BUNDLE_PATH!.trim();
    const bundle = existsSync(savePath) ? Bundle.load(savePath) : undefined;
    return AutomationSession.open({
      bundle,
      savePath,
      captureRunId: env.ATB_CAPTURE_RUN_ID?.trim(),
      purposeTag: env.ATB_CAPTURE_MODE?.trim() || "capture_run",
    });
  }

  get bundle(): Bundle {
    return this.ctx.bundle;
  }

  /** Begin or continue the active request chain. */
  beginRequest(purposeTag?: string, requestId?: string): string {
    const tag = purposeTag?.trim() || this.defaultPurposeTag;
    const rid = this.ctx.bootstrapRequest(tag, requestId ?? this.activeRequestId);
    this.activeRequestId = rid;
    return rid;
  }

  /** Emit ai.model.invoked for the active request. */
  logModelInvocation(input: ModelInvocationInput): void {
    const requestId = this.beginRequest(this.defaultPurposeTag, input.requestId);
    this.ctx.emit(AI_MODEL_INVOKED_EVENT_TYPE, {
      request_id: requestId,
      model_provider: input.provider,
      model_id: input.model,
      model_parameters_digest: canonicalDigest(input.parameters ?? {}),
      prompt_digest: canonicalDigest(input.prompt),
    });
  }

  /** Emit ai.model.output for the active request. */
  logModelOutput(input: ModelOutputInput): void {
    const requestId = this.requestId(input.requestId);
    const payload: Record<string, unknown> = {
      output_digest: valueDigest(input.output),
      output_format: input.outputFormat ?? "text/plain",
    };
    if (requestId !== undefined) {
      payload.request_id = requestId;
    }
    this.ctx.emit(AI_MODEL_OUTPUT_EVENT_TYPE, payload);
  }

  /** Emit ai.retrieval.executed for the active request. */
  logRetrieval(input: RetrievalInput): void {
    const requestId = this.beginRequest(this.defaultPurposeTag, input.requestId);
    this.ctx.emit(AI_RETRIEVAL_EXECUTED_EVENT_TYPE, {
      request_id: requestId,
      retrieval_query_hash: canonicalDigest(input.query),
      retrieval_corpus_id: input.corpusId,
      retrieval_corpus_version: input.corpusVersion,
      top_k: input.topK,
      result_set_digest: canonicalDigest(input.resultSet),
    });
  }

  /** Emit ai.response.sent to close the application response boundary. */
  logResponseSent(input: ResponseSentInput): void {
    const requestId = this.beginRequest(this.defaultPurposeTag, input.requestId);
    this.ctx.emit(AI_RESPONSE_SENT_EVENT_TYPE, {
      request_id: requestId,
      output_digest: valueDigest(input.output),
      output_format: input.outputFormat ?? "text/plain",
    });
  }

  /**
   * Run a privileged tool action with request bootstrap, gate events, and commit.
   */
  async runToolAction<T>(
    action: ActionGateInput,
    fn: () => T | Promise<T>
  ): Promise<T> {
    this.beginRequest("privileged_tool_action");
    const actionId = newActionId(action.actionId);
    const result = await this.toolGate.run({ ...action, actionId }, fn);
    this.ctx.emit(AI_ACTION_COMMITTED_EVENT_TYPE, {
      action_id: actionId,
      commit_outcome: "success",
      sink_receipt_digest: valueDigest(result),
    });
    return result;
  }

  /** Record a policy decision without running an action. */
  logPolicyDecision(
    action: PolicyDecisionActionInput,
    decision: WorkflowPolicyDecision
  ): string {
    return this.policyRecorder.record(action, decision);
  }

  /** Persist the bundle to disk. */
  save(savePath?: string): void {
    this.ctx.bundle.save(savePath ?? this.savePath);
  }

  /** Append a named snapshot boundary event. */
  snapshot(name: string): void {
    const trimmed = name.trim();
    if (trimmed === "") {
      throw new Error("snapshot name must be non-empty");
    }
    this.ctx.emit(SNAPSHOT_EVENT_TYPE, { name: trimmed });
  }

  /** Flush the bundle and optionally append a snapshot. */
  close(options: CloseSessionOptions = {}): void {
    if (options.snapshotName !== undefined) {
      this.snapshot(options.snapshotName);
    }
    this.save(options.savePath);
  }

  private requestId(explicit?: string): string | undefined {
    const candidate = explicit?.trim() || this.activeRequestId;
    return candidate || undefined;
  }
}
