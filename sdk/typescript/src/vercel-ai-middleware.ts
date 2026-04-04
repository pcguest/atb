import { createHash, randomBytes, randomUUID } from "node:crypto";
import { Bundle } from "./bundle.js";

export type PrivacyMode = "off" | "hash" | "redact";

type Phase = "start" | "delta" | "end" | "error";

interface SpanState {
  traceId: string;
  spanId: string;
  parentSpanId?: string;
  runId: string;
  eventType: "ai.llm.call" | "ai.tool.exec" | "ai.chain.run";
  startedAt: Date;
  context: Record<string, unknown>;
  chunks: string[];
}

interface TokenUsage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

interface PrivacyText {
  text: string;
  sha256: string;
}

export interface ATBMiddlewareOptions {
  bundle?: Bundle;
  enabled?: boolean;
  autoSave?: boolean;
  savePath?: string;
  privacyMode?: PrivacyMode;
  actorId?: string;
  orgId?: string;
  workspaceId?: string;
  framework?: string;
  frameworkVersion?: string;
}

export interface ChainStartInput {
  runId?: string;
  parentRunId?: string;
  chainName?: string;
  inputs?: Record<string, unknown>;
}

export interface ChainEndInput {
  runId?: string;
  outputs?: Record<string, unknown>;
}

export interface LLMStartInput {
  runId?: string;
  parentRunId?: string;
  provider?: string;
  model?: string;
  prompt: string;
}

export interface StepFinishInput {
  runId?: string;
  text?: string;
  usage?: {
    promptTokens?: number;
    completionTokens?: number;
    totalTokens?: number;
  };
  finishReason?: string;
  toolCalls?: Array<{
    toolName: string;
    input?: unknown;
    output?: unknown;
  }>;
}

export interface ToolStartInput {
  runId?: string;
  parentRunId?: string;
  toolName: string;
  input: unknown;
}

export interface ToolEndInput {
  runId?: string;
  output: unknown;
}

export interface ATBMiddleware {
  readonly bundle: Bundle;
  onChainStart(input: ChainStartInput): void;
  onChainEnd(input: ChainEndInput): void;
  onChainError(error: unknown, runId?: string): void;
  onLLMStart(input: LLMStartInput): void;
  onTokenGenerated(token: string, runId?: string): void;
  onStepFinish(input: StepFinishInput): void;
  onLLMEnd(input: StepFinishInput): void;
  onLLMError(error: unknown, runId?: string): void;
  onToolStart(input: ToolStartInput): void;
  onToolEnd(input: ToolEndInput): void;
  onToolError(error: unknown, runId?: string): void;
}

const DEFAULT_SAVE_PATH = "run.atb/bundle.atb";

export function atbMiddleware(options: ATBMiddlewareOptions = {}): ATBMiddleware {
  return new ATBMiddlewareImpl(options);
}

class ATBMiddlewareImpl implements ATBMiddleware {
  readonly bundle: Bundle;

  private readonly enabled: boolean;
  private readonly autoSave: boolean;
  private readonly savePath: string;
  private readonly privacyMode: PrivacyMode;
  private readonly actorId?: string;
  private readonly orgId?: string;
  private readonly workspaceId?: string;
  private readonly framework: string;
  private readonly frameworkVersion: string;
  private readonly runs: Map<string, SpanState>;

  constructor(options: ATBMiddlewareOptions) {
    this.bundle = options.bundle ?? new Bundle();
    this.enabled = options.enabled ?? true;
    this.autoSave = options.autoSave ?? false;
    this.savePath = options.savePath ?? DEFAULT_SAVE_PATH;
    this.privacyMode = options.privacyMode ?? "off";
    this.actorId = options.actorId;
    this.orgId = options.orgId;
    this.workspaceId = options.workspaceId;
    this.framework = options.framework ?? "vercel-ai";
    this.frameworkVersion = options.frameworkVersion ?? "unknown";
    this.runs = new Map<string, SpanState>();

    if (!["off", "hash", "redact"].includes(this.privacyMode)) {
      throw new Error("privacyMode must be one of: off, hash, redact");
    }
  }

  onChainStart(input: ChainStartInput): void {
    if (!this.enabled) return;
    const state = this.startSpan(input.runId, input.parentRunId, "ai.chain.run");
    state.context = {
      chain_name: input.chainName ?? "unknown",
      input_keys: Object.keys(input.inputs ?? {}).sort(),
    };

    this.emit("ai.chain.run", "start", state, state.context, true, null, undefined);
  }

  onChainEnd(input: ChainEndInput): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(input.runId, "ai.chain.run");
    const context = {
      ...state.context,
      output_keys: Object.keys(input.outputs ?? {}).sort(),
    };
    this.emit("ai.chain.run", "end", state, context, true, null, new Date());
    this.finishSpan(state.runId);
  }

  onChainError(error: unknown, runId?: string): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(runId, "ai.chain.run");
    this.emit("ai.chain.run", "error", state, state.context, false, normalizeError(error), new Date());
    this.finishSpan(state.runId);
  }

  onLLMStart(input: LLMStartInput): void {
    if (!this.enabled) return;
    const state = this.startSpan(input.runId, input.parentRunId, "ai.llm.call");
    state.context = {
      provider: input.provider ?? "unknown",
      model: input.model ?? "unknown",
      prompt: this.privacyText(input.prompt),
    };

    this.emit("ai.llm.call", "start", state, state.context, true, null, undefined);
  }

  onTokenGenerated(token: string, runId?: string): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(runId, "ai.llm.call");
    state.chunks.push(token);
    const context = {
      ...state.context,
      stream: {
        chunk_index: state.chunks.length - 1,
        chunk: this.privacyText(token),
      },
    };
    this.emit("ai.llm.call", "delta", state, context, true, null, undefined);
  }

  onStepFinish(input: StepFinishInput): void {
    if (!this.enabled) return;

    if (Array.isArray(input.toolCalls)) {
      for (let i = 0; i < input.toolCalls.length; i++) {
        const call = input.toolCalls[i];
        const toolRunId = `${this.runKey(input.runId)}:tool:${i}`;
        this.onToolStart({
          runId: toolRunId,
          parentRunId: input.runId,
          toolName: call.toolName,
          input: call.input,
        });
        this.onToolEnd({ runId: toolRunId, output: call.output });
      }
    }

    this.onLLMEnd(input);
  }

  onLLMEnd(input: StepFinishInput): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(input.runId, "ai.llm.call");

    const completion = state.chunks.length > 0 ? state.chunks.join("") : input.text ?? "";
    const usage = normalizeTokenUsage(input.usage);

    const context = {
      ...state.context,
      completion: this.privacyText(completion),
      token_usage: usage,
      finish_reason: input.finishReason ?? null,
    };

    this.emit("ai.llm.call", "end", state, context, true, null, new Date());
    this.finishSpan(state.runId);
  }

  onLLMError(error: unknown, runId?: string): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(runId, "ai.llm.call");
    this.emit("ai.llm.call", "error", state, state.context, false, normalizeError(error), new Date());
    this.finishSpan(state.runId);
  }

  onToolStart(input: ToolStartInput): void {
    if (!this.enabled) return;
    const state = this.startSpan(input.runId, input.parentRunId, "ai.tool.exec");
    state.context = {
      tool_name: input.toolName,
      input: this.privacyText(toStableString(input.input)),
    };
    this.emit("ai.tool.exec", "start", state, state.context, true, null, undefined);
  }

  onToolEnd(input: ToolEndInput): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(input.runId, "ai.tool.exec");
    const context = {
      ...state.context,
      output: this.privacyText(toStableString(input.output)),
    };
    this.emit("ai.tool.exec", "end", state, context, true, null, new Date());
    this.finishSpan(state.runId);
  }

  onToolError(error: unknown, runId?: string): void {
    if (!this.enabled) return;
    const state = this.getOrCreateSpan(runId, "ai.tool.exec");
    this.emit("ai.tool.exec", "error", state, state.context, false, normalizeError(error), new Date());
    this.finishSpan(state.runId);
  }

  private emit(
    eventType: "ai.llm.call" | "ai.tool.exec" | "ai.chain.run",
    phase: Phase,
    state: SpanState,
    context: Record<string, unknown>,
    ok: boolean,
    error: Record<string, string> | null,
    endedAt: Date | undefined
  ): void {
    const latencyMs = endedAt ? Math.max(0, endedAt.getTime() - state.startedAt.getTime()) : null;
    const emittedAt = new Date();

    const payload = {
      trace_id: state.traceId,
      span_id: state.spanId,
      parent_span_id: state.parentSpanId ?? null,
      framework: this.framework,
      framework_version: this.frameworkVersion,
      phase,
      run_id: state.runId,
      context,
      privacy: {
        mode: this.privacyMode,
        redaction_enabled: this.privacyMode !== "off",
        pii_ruleset: "phase4-gdpr-v1",
      },
      timing: {
        started_at: toIso(state.startedAt),
        ended_at: endedAt ? toIso(endedAt) : null,
        latency_ms: latencyMs,
      },
      status: {
        ok,
        error,
      },
      emitted_at: toIso(emittedAt),
    };

    this.bundle.append(eventType, payload, {
      actorId: this.actorId,
      orgId: this.orgId,
      workspaceId: this.workspaceId,
      timestamp: toIso(emittedAt),
      traceId: state.traceId,
      spanId: state.spanId,
      parentSpanId: state.parentSpanId,
    });

    if (this.autoSave) {
      this.bundle.save(this.savePath);
    }
  }

  private startSpan(
    runId: string | undefined,
    parentRunId: string | undefined,
    eventType: "ai.llm.call" | "ai.tool.exec" | "ai.chain.run"
  ): SpanState {
    const runKey = this.runKey(runId);
    const parent = parentRunId ? this.runs.get(this.runKey(parentRunId)) : undefined;

    const state: SpanState = {
      traceId: parent?.traceId ?? newId("trc"),
      spanId: newId("spn"),
      parentSpanId: parent?.spanId,
      runId: runKey,
      eventType,
      startedAt: new Date(),
      context: {},
      chunks: [],
    };
    this.runs.set(runKey, state);
    return state;
  }

  private getOrCreateSpan(
    runId: string | undefined,
    eventType: "ai.llm.call" | "ai.tool.exec" | "ai.chain.run"
  ): SpanState {
    const runKey = this.runKey(runId);
    const existing = this.runs.get(runKey);
    if (existing) {
      return existing;
    }
    return this.startSpan(runKey, undefined, eventType);
  }

  private finishSpan(runId: string): void {
    this.runs.delete(runId);
  }

  private runKey(runId: string | undefined): string {
    return runId && runId.trim() !== "" ? runId : newId("run");
  }

  private privacyText(text: string): PrivacyText {
    let emitted = text;
    if (this.privacyMode === "hash") {
      emitted = `sha256:${sha256(text)}`;
    } else if (this.privacyMode === "redact") {
      emitted = text === "" ? "" : "[REDACTED]";
    }

    return {
      text: emitted,
      sha256: `sha256:${sha256(emitted)}`,
    };
  }
}

function normalizeError(error: unknown): Record<string, string> {
  if (error instanceof Error) {
    return {
      type: error.name,
      message: error.message,
    };
  }
  return {
    type: "Error",
    message: String(error),
  };
}

function normalizeTokenUsage(usage: StepFinishInput["usage"]): TokenUsage {
  const prompt = usage?.promptTokens ?? 0;
  const completion = usage?.completionTokens ?? 0;
  const total = usage?.totalTokens ?? prompt + completion;
  return {
    prompt_tokens: prompt,
    completion_tokens: completion,
    total_tokens: total,
  };
}

function toIso(value: Date): string {
  return value.toISOString().replace(/\.\d{3}Z$/, "Z");
}

function sha256(value: string): string {
  return createHash("sha256").update(value, "utf8").digest("hex");
}

function toStableString(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  return JSON.stringify(value, objectSortReplacer, 0) ?? "";
}

function objectSortReplacer(_key: string, value: unknown): unknown {
  if (Array.isArray(value) || value === null || typeof value !== "object") {
    return value;
  }
  const entries = Object.entries(value as Record<string, unknown>).sort(([a], [b]) =>
    a.localeCompare(b)
  );
  const out: Record<string, unknown> = {};
  for (const [k, v] of entries) {
    out[k] = v;
  }
  return out;
}

function newId(prefix: string): string {
  if (typeof randomUUID === "function") {
    return `${prefix}_${randomUUID().replace(/-/g, "")}`;
  }
  return `${prefix}_${randomBytes(16).toString("hex")}`;
}
