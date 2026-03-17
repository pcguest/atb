import { z } from "zod";

import { hashStringSchema } from "@/lib/schemas/common";

export const verificationStatusSchema = z.enum(["valid", "invalid"]);

export const verificationResponseSchema = z.object({
  status: verificationStatusSchema,
  message: z.string(),
  bundle_path: z.string(),
  chain_length: z.number().int().nonnegative(),
  head_hash: hashStringSchema.optional(),
});

export type VerificationResponse = z.infer<typeof verificationResponseSchema>;
