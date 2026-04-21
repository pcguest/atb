import { z } from "zod";

export const failureDTOSchema = z.object({
  kind: z.string(),
  detail: z.string(),
});

const subScoresSchema = z.object({
  ac: z.number(),
  ec: z.number(),
  fc: z.number(),
  gc: z.number(),
  rc: z.number(),
  sc: z.number(),
  tc: z.number(),
  xc: z.number(),
});

export const profileReportSummarySchema = z.object({
  profile_id: z.string(),
  passed: z.boolean(),
  cas_score: z.number(),
  cas_grade: z.enum(["A", "B", "C", "D", "F"]),
  effective_score: z.number().optional(),
  corroboration_bonus: z.number().optional(),
  sub_scores: subScoresSchema,
  critical_failures: z.array(failureDTOSchema),
  warnings: z.array(failureDTOSchema),
});

export type FailureDTO = z.infer<typeof failureDTOSchema>;
export type ProfileReportSummary = z.infer<typeof profileReportSummarySchema>;
