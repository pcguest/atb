/**
 * @pcguest/atb-sdk — ATB (Agent Trace Bundle) TypeScript SDK
 *
 * Tamper-evident, verifiable audit trails for AI workflows.
 *
 * @example
 * ```ts
 * import { Bundle } from "@pcguest/atb-sdk";
 *
 * const bundle = new Bundle();
 * bundle.append("dev.session", { date: "2025-01-15" });
 * bundle.save();
 * ```
 */

export { Bundle, ATBVerificationError } from "./bundle.js";
export { EncryptError, decryptBundle, decryptRaw, encryptBundle, encryptRaw } from "./encrypt.js";
export { computeHash, chainEvents, GENESIS_HASH } from "./hash.js";
export { canonicalize } from "./canonicalize.js";
export { atbMiddleware } from "./vercel-ai-middleware.js";
export { normalizeOptionalIdentity, prepareForCanonical } from "./event.js";
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
export type { ATBEvent, ATBRecord, BundleOptions } from "./types.js";
