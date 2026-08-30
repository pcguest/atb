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
export const SDK_VERSION = "1.15.4";

/**
 * @returns SDK version and hash-chain algorithm metadata.
 */
export function version(): { version: string; algorithm: string } {
  return { version: SDK_VERSION, algorithm: "SHA-256+RFC8785" };
}

export {
  Bundle,
  ATBVerificationError,
  BundleResourceLimitError,
  MAX_BUNDLE_RECORDS,
  MAX_BUNDLE_SIZE_BYTES,
  MAX_LINE_SIZE_BYTES,
} from "./bundle.js";
export type { BundleLoadLimits, SignatureEvidence, VerifyResult } from "./bundle.js";
export { MortiseClient, MortiseError } from "./mortise-client.js";
export type {
  MortiseClientOptions,
  MortiseJSON,
  MortiseReceipt,
} from "./mortise-client.js";
export * from "./eventTypes.js";
export { ActionGate, ActionGateDeniedError, principalPayload } from "./action-gate.js";
export {
  BackgroundJobTracker,
} from "./background-job-tracker.js";
export { DataExportGate, DataExportDeniedError } from "./data-export-gate.js";
export { HumanOverrideGate, HumanOverrideDeniedError } from "./human-override-gate.js";
export { PolicyDecisionRecorder } from "./policy-decision-recorder.js";
export { AutomationSession } from "./automation-session.js";
export {
  WorkflowContext,
  actorIdHash,
  canonicalDigest,
  newActionId,
  newJobId,
  valueDigest,
} from "./workflow-common.js";
export { EncryptError, decryptBundle, decryptRaw, encryptBundle, encryptRaw } from "./encrypt.js";
export { computeHash, chainEvents, GENESIS_HASH } from "./hash.js";
export { canonicalize } from "./canonicalize.js";
export { atbMiddleware } from "./vercel-ai-middleware.js";
export { gateVercelTool } from "./vercel-gate.js";
export { wrapOpenAI, wrapAnthropic } from "./sdk-capture.js";
export type {
  SDKCaptureOptions,
  OpenAIChatParams,
  OpenAIChatResponse,
  AnthropicMessagesParams,
  AnthropicMessagesResponse,
} from "./sdk-capture.js";
export { normalizeOptionalIdentity, prepareForCanonical } from "./event.js";
export { identityEvidencePayload } from "./identity-evidence.js";
export type { IdentityEvidence } from "./identity-evidence.js";
export type {
  ActionGateDecision,
  ActionGateInput,
  ActionGateMode,
  ActionGateOptions,
  ActionPrincipal,
} from "./action-gate.js";
export type {
  BackgroundJobScheduleInput,
  BackgroundJobTrackerOptions,
} from "./background-job-tracker.js";
export type {
  DataExportApproval,
  DataExportGateOptions,
  DataExportInput,
  DataExportGateMode,
} from "./data-export-gate.js";
export type {
  HumanOverrideActionInput,
  HumanOverrideApprovalInput,
  HumanOverrideGateOptions,
  HumanOverrideGateMode,
} from "./human-override-gate.js";
export type {
  PolicyDecisionActionInput,
  PolicyDecisionRecorderOptions,
} from "./policy-decision-recorder.js";
export type {
  AutomationSessionOptions,
  CloseSessionOptions,
  ModelInvocationInput,
  ModelOutputInput,
  ResponseSentInput,
  RetrievalInput,
} from "./automation-session.js";
export type {
  WorkflowContextOptions,
  WorkflowPolicyDecision,
} from "./workflow-common.js";
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
export {
  FORMAT_GENERIC_JSONL,
  FORMAT_OPENAI_JSONL,
  isCaptureEnvironment,
  normaliseCaptureFormat,
} from "./capture.js";
export type { CaptureFormat } from "./capture.js";
