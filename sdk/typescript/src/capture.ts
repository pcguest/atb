// SPDX-License-Identifier: MIT
/**
 * Capture helpers mirroring the Go `internal/capture` surface.
 */
export const FORMAT_GENERIC_JSONL = "generic-jsonl";
export const FORMAT_OPENAI_JSONL = "openai-jsonl";

export type CaptureFormat = typeof FORMAT_GENERIC_JSONL | typeof FORMAT_OPENAI_JSONL;

/** Normalised chatlog formats supported for import in Capture v1. */
export function normaliseCaptureFormat(format: string): CaptureFormat {
  const fmt = format.trim().toLowerCase();
  if (fmt === FORMAT_OPENAI_JSONL) {
    return FORMAT_GENERIC_JSONL;
  }
  if (fmt === FORMAT_GENERIC_JSONL) {
    return FORMAT_GENERIC_JSONL;
  }
  throw new Error(`unsupported provider ${format}`);
}

/** Returns true when live capture environment variables are present. */
export function isCaptureEnvironment(env: Record<string, string | undefined> = process.env): boolean {
  return Boolean(env.ATB_BUNDLE_PATH?.trim());
}
