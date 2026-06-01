import { z } from "zod";

// Mirrors internal/sessionindex.SessionEntry as serialised by GET /api/v1/sessions.
export const sessionActorSchema = z.object({
  display_name: z.string(),
  email: z.string(),
});

export const sessionEntrySchema = z.object({
  session_id: z.string(),
  actor: sessionActorSchema,
  started_at: z.string(),
  closed_at: z.string().nullable().optional(),
  exchange_count: z.number().int().nonnegative(),
  inferred_profile: z.string(),
  cas_grade: z.string(),
  anomaly_flags: z.array(z.string()),
  bundle_path: z.string(),
});

// sessions is a Go slice that marshals to null when empty; coerce to [].
export const sessionsResponseSchema = z.object({
  sessions: z
    .array(sessionEntrySchema)
    .nullable()
    .transform((value) => value ?? []),
});

export type SessionEntry = z.infer<typeof sessionEntrySchema>;
export type SessionsResponse = z.infer<typeof sessionsResponseSchema>;
