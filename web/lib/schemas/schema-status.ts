import { z } from "zod";

// Mirrors pkg/api/v1.EventTypeStatusDTO as serialised by GET /api/v1/schema/status.
export const eventTypeStatusSchema = z.object({
  type: z.string(),
  criticality: z.string(),
  declared: z.boolean(),
  observed: z.number().int().nonnegative(),
  required_fields: z
    .array(z.string())
    .nullable()
    .transform((value) => value ?? []),
  incomplete: z.number().int().nonnegative(),
  missing_fields: z
    .array(z.string())
    .nullable()
    .optional()
    .transform((value) => value ?? []),
});

// Mirrors pkg/api/v1.SchemaStatusResponse. Go slices marshal to null when empty,
// so the array fields are coerced to [].
export const schemaStatusResponseSchema = z.object({
  schema_source: z.string(),
  declared_types: z.number().int().nonnegative(),
  observed_types: z.number().int().nonnegative(),
  total_events: z.number().int().nonnegative(),
  incomplete_events: z.number().int().nonnegative(),
  undeclared_types: z
    .array(z.string())
    .nullable()
    .transform((value) => value ?? []),
  types: z
    .array(eventTypeStatusSchema)
    .nullable()
    .transform((value) => value ?? []),
});

export type EventTypeStatus = z.infer<typeof eventTypeStatusSchema>;
export type SchemaStatusResponse = z.infer<typeof schemaStatusResponseSchema>;
