import { z } from "zod";

export const apiErrorSchema = z.object({
  error: z.string(),
});

export const hashStringSchema = z
  .string()
  .regex(/^[0-9a-f]{64}$/i, "Expected a 64-character SHA-256 hash string");
