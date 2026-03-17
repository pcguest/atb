import { z } from "zod";

import { hashStringSchema } from "@/lib/schemas/common";

const isoTimestampSchema = z.string().min(1);

export const bundleMetaResponseSchema = z.object({
  bundle_path: z.string(),
  event_count: z.number().int().nonnegative(),
  type_counts: z.record(z.string(), z.number().int().nonnegative()),
  first_timestamp: isoTimestampSchema.optional(),
  last_timestamp: isoTimestampSchema.optional(),
  genesis_hash: hashStringSchema.optional(),
  verified_at: isoTimestampSchema.optional(),
  verified: z.boolean(),
  verification_message: z.string(),
});

export const eventRecordSchema = z.object({
  seq: z.number().int().positive(),
  type: z.string(),
  hash: hashStringSchema,
  prev_hash: hashStringSchema,
  timestamp: isoTimestampSchema.optional(),
  trace_id: z.string().optional(),
  span_id: z.string().optional(),
  parent_span_id: z.string().optional(),
  data: z.record(z.string(), z.unknown()),
});

export const bundleEventsResponseSchema = z.object({
  offset: z.number().int().nonnegative(),
  limit: z.number().int().positive(),
  total: z.number().int().nonnegative(),
  events: z.array(eventRecordSchema),
});

export type BundleMetaResponse = z.infer<typeof bundleMetaResponseSchema>;
export type EventRecord = z.infer<typeof eventRecordSchema>;
export type BundleEventsResponse = z.infer<typeof bundleEventsResponseSchema>;
