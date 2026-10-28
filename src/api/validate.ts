// The answer every route gives a request that breaks the contract: 422 with every problem as text, one per line.
import { z } from "@hono/zod-openapi";
import type { Context } from "hono";

export const invalid = (c: Context, error: string) => c.json({ error }, 422);

type Issues = { issues: { path: PropertyKey[]; message: string }[] };

// failure describes, in the OpenAPI document, an answer that refuses with a reason.
export const failure = (description: string) => json(z.object({ error: z.string() }), description);

// json describes a JSON answer in the OpenAPI document.
export const json = <T extends z.ZodType>(schema: T, description: string) => ({
  content: { "application/json": { schema } },
  description,
});

// body describes the JSON body a route requires; without required, a request with no body skips validation.
export const body = <T extends z.ZodType>(schema: T, description: string) => ({ ...json(schema, description), required: true });

// hook is the validator hook that turns a failed parse into invalid.
export const hook = (result: { success: boolean; error?: Issues }, c: Context) => {
  if (!result.success && result.error) {
    return invalid(c, result.error.issues.map((i) => `${i.path.map(String).join(".") || "body"}: ${i.message}`).join("\n"));
  }
};
