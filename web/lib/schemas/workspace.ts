import { z } from "zod";

export const workspaceBundleSummarySchema = z.object({
  id: z.string(),
  session_id: z.string(),
  bundle_path: z.string(),
  profile_id: z.string().optional(),
  head_hash: z.string().optional(),
  event_count: z.number().int().nonnegative(),
  opened_at: z.string(),
  closed_at: z.string(),
});

export const workspaceBundlesResponseSchema = z.object({
  bundles: z.array(workspaceBundleSummarySchema),
});

export type WorkspaceBundleSummary = z.infer<typeof workspaceBundleSummarySchema>;
export type WorkspaceBundlesResponse = z.infer<typeof workspaceBundlesResponseSchema>;
