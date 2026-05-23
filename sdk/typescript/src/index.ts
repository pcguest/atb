/**
 * @pcguest/atb-sdk — ATB (Agent Trace Bundle) TypeScript SDK
 *
 * Tamper-evident, verifiable audit trails for AI workflows.
 *
 * @example
 * ```ts
 * import {
 *   AI_MODEL_INVOKED_EVENT_TYPE,
 *   AI_MODEL_OUTPUT_EVENT_TYPE,
 *   AI_REQUEST_RECEIVED_EVENT_TYPE,
 *   Bundle,
 * } from "@pcguest/atb-sdk";
 *
 * const bundle = new Bundle();
 * bundle.append(AI_REQUEST_RECEIVED_EVENT_TYPE, {
 *   request_id: "req-001",
 *   actor_id_hash: "sha256-actor-abc",
 *   purpose_tag: "rag_answer",
 * });
 * bundle.append(AI_MODEL_INVOKED_EVENT_TYPE, {
 *   model_provider: "openai",
 *   model_id: "gpt-4o",
 *   model_parameters_digest: "sha256-params-def",
 *   prompt_digest: "sha256-prompt-ghi",
 * });
 * bundle.append(AI_MODEL_OUTPUT_EVENT_TYPE, {
 *   output_digest: "sha256-output-jkl",
 *   output_format: "text/plain",
 * });
 * bundle.save();
 * ```
 */

/** Current SDK package version. */
export const SDK_VERSION = "1.11.0";

/**
 * @returns SDK version and hash-chain algorithm metadata.
 */
export function version(): { version: string; algorithm: string } {
  return { version: SDK_VERSION, algorithm: "SHA-256+RFC8785" };
}

export { Bundle, ATBVerificationError } from "./bundle.js";
export type { SignatureEvidence, VerifyResult } from "./bundle.js";
export * from "./eventTypes.js";
export { ActionGate, ActionGateDeniedError } from "./action-gate.js";
export { EncryptError, decryptBundle, decryptRaw, encryptBundle, encryptRaw } from "./encrypt.js";
export { computeHash, chainEvents, GENESIS_HASH } from "./hash.js";
export { canonicalize } from "./canonicalize.js";
export { atbMiddleware } from "./vercel-ai-middleware.js";
export { gateVercelTool } from "./vercel-gate.js";
export { normalizeOptionalIdentity, prepareForCanonical } from "./event.js";
export type {
  ActionGateDecision,
  ActionGateInput,
  ActionGateMode,
  ActionGateOptions,
} from "./action-gate.js";
export type { AppendIdentityOptions, Event } from "./event.js";
export type {
  ATBMiddleware,
  ATBMiddlewareOptions,
  ChainEndInput,
  ChainStartInput,
  LLMStartInput,
  PrivacyMode,
  StepFinishInput,
  ToolEndInput,
  ToolStartInput,
} from "./vercel-ai-middleware.js";
export type { VercelAITool } from "./vercel-gate.js";
export type { ATBEvent, ATBRecord, BundleOptions, BundleSignature } from "./types.js";
