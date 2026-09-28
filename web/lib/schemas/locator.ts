import { z } from "zod";

// Result of resolving an ATB semantic evidence locator against the loaded bundle.
// `ok` reports identity only: resolving a locator is not an integrity check,
// does not establish truth, and does not grant governance approval.
export const locateResponseSchema = z.object({
  ok: z.boolean(),
  canonical: z.string().optional(),
  seq: z.number().int().nonnegative(),
  record_hash_matched: z.boolean(),
  error_code: z.string().optional(),
  message: z.string().optional(),
});
