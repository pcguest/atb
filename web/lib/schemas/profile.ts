import { z } from "zod";

export const failureDTOSchema = z.object({
  kind: z.string(),
  detail: z.string(),
});

export const profileReportSummarySchema = z.object({
  profile_id: z.string(),
  pass: z.boolean(),
  chain_valid: z.boolean().optional(),
  anchor_status: z.string().optional(),
  cas_score: z.number().default(0),
  cas_grade: z.string().optional().default(""),
  effective_score: z.number().optional(),
  corroboration_bonus: z.number().optional(),
  sub_scores: z.record(z.string(), z.number()).optional().default({}),
  critical_failures: z.array(failureDTOSchema),
  warnings: z.array(z.string()),
});

export type FailureDTO = z.infer<typeof failureDTOSchema>;
export type ProfileReportSummary = z.infer<typeof profileReportSummarySchema>;
