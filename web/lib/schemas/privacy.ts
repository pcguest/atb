import { z } from "zod";

export const privacyRevealRequestSchema = z.object({
  seq: z.number().int().positive(),
  field_path: z.string().min(1),
  reason: z.string().optional(),
});

export const privacyRevealResponseSchema = z.object({
  seq: z.number().int().positive(),
  field_path: z.string(),
  value: z.unknown(),
});

export type PrivacyRevealRequest = z.infer<typeof privacyRevealRequestSchema>;
export type PrivacyRevealResponse = z.infer<typeof privacyRevealResponseSchema>;
