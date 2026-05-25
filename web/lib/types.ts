import type { z } from "zod";

import {
  apiErrorSchema,
  bundleEventsResponseSchema,
  bundleGraphResponseSchema,
  bundleMetaResponseSchema,
  eventRecordSchema,
  failureDTOSchema,
  graphEdgeSchema,
  graphNodeSchema,
  privacyRevealRequestSchema,
  privacyRevealResponseSchema,
  profileReportSummarySchema,
  verificationResponseSchema,
  workspaceBundlesResponseSchema,
} from "@/lib/schemas";

export type APIError = z.infer<typeof apiErrorSchema>;
export type VerificationResponse = z.infer<typeof verificationResponseSchema>;
export type BundleMetaResponse = z.infer<typeof bundleMetaResponseSchema>;
export type EventRecord = z.infer<typeof eventRecordSchema>;
export type BundleEventsResponse = z.infer<typeof bundleEventsResponseSchema>;
export type GraphNode = z.infer<typeof graphNodeSchema>;
export type GraphEdge = z.infer<typeof graphEdgeSchema>;
export type BundleGraphResponse = z.infer<typeof bundleGraphResponseSchema>;
export type PrivacyRevealRequest = z.infer<typeof privacyRevealRequestSchema>;
export type PrivacyRevealResponse = z.infer<typeof privacyRevealResponseSchema>;
export type FailureDTO = z.infer<typeof failureDTOSchema>;
export type ProfileReportSummary = z.infer<typeof profileReportSummarySchema>;
export type WorkspaceBundlesResponse = z.infer<typeof workspaceBundlesResponseSchema>;
