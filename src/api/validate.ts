// The answer every route gives a request that breaks the contract: 422 with every problem as text, one per line.
import type { Context } from "hono";

export const invalid = (c: Context, error: string) => c.json({ error }, 422);

type Issues = { issues: { path: PropertyKey[]; message: string }[] };

// hook is the zValidator hook that turns a failed parse into invalid.
export const hook = (result: { success: boolean; error?: Issues }, c: Context) => {
  if (!result.success && result.error) {
    return invalid(c, result.error.issues.map((i) => `${i.path.map(String).join(".") || "body"}: ${i.message}`).join("\n"));
  }
};
