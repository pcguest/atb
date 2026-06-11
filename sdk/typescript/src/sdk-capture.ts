// Opt-in capture adapters for the direct OpenAI and Anthropic SDK clients.
//
// Unlike LangChain or the Vercel AI SDK, the first-party OpenAI and Anthropic
// clients expose no callback/middleware hook. The thin adapter therefore wraps
// the client's bound `create` method: it records the request, awaits the real
// call, records the response (or error), and returns the untouched value. All
// event emission is delegated to the existing `atbMiddleware` recorder, so this
// module only maps each SDK's request/response shape onto the shared callbacks —
// it adds no second emit path.
//
// Blind spots (documented, by design):
//   - Streaming responses (`stream: true`) are NOT supported by the thin
//     adapter — it cannot aggregate a streamed result without consuming the
//     caller's stream. Such calls throw; use `atb intercept` (proxy capture) or
//     `atbMiddleware` for token-level streaming capture.
//   - Only the chat/messages create call is instrumented. Embeddings, files,
//     and other endpoints are pass-through and unrecorded.

import { randomUUID } from "node:crypto";
import { atbMiddleware, type ATBMiddleware, type ATBMiddlewareOptions } from "./vercel-ai-middleware.js";

/** Options for the SDK capture adapters. Mirrors `ATBMiddlewareOptions`. */
export interface SDKCaptureOptions extends ATBMiddlewareOptions {
  /** Reuse an existing recorder instead of constructing one. */
  middleware?: ATBMiddleware;
}

// ---------------------------------------------------------------------------
// OpenAI chat completions
// ---------------------------------------------------------------------------

/** Minimal structural shape of an OpenAI chat-completions request. */
export interface OpenAIChatParams {
  model?: string;
  messages?: Array<{ role?: string; content?: unknown }>;
  stream?: boolean;
  [key: string]: unknown;
}

/** Minimal structural shape of an OpenAI chat-completions response. */
export interface OpenAIChatResponse {
  model?: string;
  choices?: Array<{
    message?: {
      content?: string | null;
      tool_calls?: Array<{ function?: { name?: string; arguments?: string } }>;
    };
    finish_reason?: string | null;
  }>;
  usage?: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number };
  [key: string]: unknown;
}

type Create<P, R> = (params: P) => Promise<R>;

/**
 * Wrap an OpenAI `chat.completions.create` method with ATB capture.
 *
 * @example
 *   const openai = new OpenAI();
 *   const create = wrapOpenAI(
 *     openai.chat.completions.create.bind(openai.chat.completions),
 *     { bundle, privacyMode: "hash" },
 *   );
 *   const res = await create({ model: "gpt-4o", messages });
 *
 * @param create Bound `chat.completions.create` callable.
 * @param options Capture options (opt-in, privacy-moded, profile-bound).
 * @returns A drop-in replacement for `create` that records each call.
 * @throws Error when a streaming request (`stream: true`) is passed.
 */
export function wrapOpenAI<P extends OpenAIChatParams, R extends OpenAIChatResponse>(
  create: Create<P, R>,
  options: SDKCaptureOptions = {},
): Create<P, R> {
  const mw = options.middleware ?? atbMiddleware(options);
  mw.recordCaptureScope({
    targets: ["openai"],
    outOfScope:
      "Only the wrapped OpenAI chat.completions.create call is recorded; " +
      "streaming requests, other endpoints, and calls made outside this " +
      "wrapper are not captured.",
  });
  return async (params: P): Promise<R> => {
    if (params?.stream) {
      throw new Error(
        "wrapOpenAI does not support streaming (stream: true); use atb intercept or atbMiddleware",
      );
    }
    const runId = newRunId();
    mw.onLLMStart({
      runId,
      provider: "openai",
      model: params?.model ?? "unknown",
      prompt: serializeMessages(params?.messages),
    });
    try {
      const response = await create(params);
      const choice = response?.choices?.[0];
      mw.onStepFinish({
        runId,
        text: choice?.message?.content ?? "",
        usage: {
          promptTokens: response?.usage?.prompt_tokens,
          completionTokens: response?.usage?.completion_tokens,
          totalTokens: response?.usage?.total_tokens,
        },
        finishReason: choice?.finish_reason ?? undefined,
        toolCalls: (choice?.message?.tool_calls ?? []).map((call) => ({
          toolName: call.function?.name ?? "unknown",
          input: call.function?.arguments,
        })),
      });
      return response;
    } catch (err) {
      mw.onLLMError(err, runId);
      throw err;
    }
  };
}

// ---------------------------------------------------------------------------
// Anthropic messages
// ---------------------------------------------------------------------------

/** Minimal structural shape of an Anthropic messages request. */
export interface AnthropicMessagesParams {
  model?: string;
  system?: string;
  messages?: Array<{ role?: string; content?: unknown }>;
  stream?: boolean;
  [key: string]: unknown;
}

/** Minimal structural shape of an Anthropic messages response. */
export interface AnthropicMessagesResponse {
  model?: string;
  content?: Array<{ type?: string; text?: string; name?: string; input?: unknown }>;
  usage?: { input_tokens?: number; output_tokens?: number };
  stop_reason?: string | null;
  [key: string]: unknown;
}

/**
 * Wrap an Anthropic `messages.create` method with ATB capture.
 *
 * @example
 *   const anthropic = new Anthropic();
 *   const create = wrapAnthropic(
 *     anthropic.messages.create.bind(anthropic.messages),
 *     { bundle },
 *   );
 *   const res = await create({ model: "claude-sonnet-4-6", max_tokens: 1024, messages });
 *
 * @param create Bound `messages.create` callable.
 * @param options Capture options (opt-in, privacy-moded, profile-bound).
 * @returns A drop-in replacement for `create` that records each call.
 * @throws Error when a streaming request (`stream: true`) is passed.
 */
export function wrapAnthropic<P extends AnthropicMessagesParams, R extends AnthropicMessagesResponse>(
  create: Create<P, R>,
  options: SDKCaptureOptions = {},
): Create<P, R> {
  const mw = options.middleware ?? atbMiddleware(options);
  mw.recordCaptureScope({
    targets: ["anthropic"],
    outOfScope:
      "Only the wrapped Anthropic messages.create call is recorded; " +
      "streaming requests, other endpoints, and calls made outside this " +
      "wrapper are not captured.",
  });
  return async (params: P): Promise<R> => {
    if (params?.stream) {
      throw new Error(
        "wrapAnthropic does not support streaming (stream: true); use atb intercept or atbMiddleware",
      );
    }
    const runId = newRunId();
    const prompt = serializeMessages(params?.messages, params?.system);
    mw.onLLMStart({ runId, provider: "anthropic", model: params?.model ?? "unknown", prompt });
    try {
      const response = await create(params);
      const blocks = response?.content ?? [];
      const text = blocks
        .filter((b) => b.type === "text" && typeof b.text === "string")
        .map((b) => b.text)
        .join("");
      const toolCalls = blocks
        .filter((b) => b.type === "tool_use")
        .map((b) => ({ toolName: b.name ?? "unknown", input: b.input }));
      mw.onStepFinish({
        runId,
        text,
        usage: {
          promptTokens: response?.usage?.input_tokens,
          completionTokens: response?.usage?.output_tokens,
          totalTokens:
            (response?.usage?.input_tokens ?? 0) + (response?.usage?.output_tokens ?? 0),
        },
        finishReason: response?.stop_reason ?? undefined,
        toolCalls,
      });
      return response;
    } catch (err) {
      mw.onLLMError(err, runId);
      throw err;
    }
  };
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

function serializeMessages(
  messages: Array<{ role?: string; content?: unknown }> | undefined,
  system?: string,
): string {
  const lines: string[] = [];
  if (typeof system === "string" && system !== "") {
    lines.push(`system: ${system}`);
  }
  for (const message of messages ?? []) {
    const role = message?.role ?? "user";
    const content =
      typeof message?.content === "string" ? message.content : stableString(message?.content);
    lines.push(`${role}: ${content}`);
  }
  return lines.join("\n");
}

function stableString(value: unknown): string {
  if (value === undefined || value === null) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  return JSON.stringify(value) ?? "";
}

function newRunId(): string {
  return `run_${randomUUID().replace(/-/g, "")}`;
}
