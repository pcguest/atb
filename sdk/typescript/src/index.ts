/**
 * @pcguest/atb-sdk — ATB (Agent Trace Bundle) TypeScript SDK
 *
 * Tamper-evident, replayable audit trails for AI agent workflows.
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
export type { ATBEvent, ATBRecord, BundleOptions } from "./types.js";
