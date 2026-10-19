// /tasks.json: every task in the shape the viewer reads, the one `nabu task ls --json` prints.
import { Hono } from "hono";
import { LISTS, type Task } from "./schema";
import { type Row, toTask } from "./tasks";

const SECTIONS = [
  ["## Situation", "situation"],
  ["## Complication", "complication"],
  ["## Question", "question"],
  ["## Answer", "answer"],
  ["## Why dropped", "why_dropped"],
] as const;

// markdown writes a task's body the way `nabu task read` does; an empty reason is left out.
export const markdown = (t: Task) =>
  `# ${t.title}\n` +
  SECTIONS.filter(([, k]) => k !== "why_dropped" || t.why_dropped.length > 0)
    .map(([h, k]) => `\n${h}\n\n${t[k].map((l) => `${l}\n`).join("")}`)
    .join("");

// frontmatter leaves out what the task does not have, as the CLI's JSON does.
const frontmatter = (t: Task) => {
  const out: Record<string, string | string[]> = { created: t.created, area: t.area };
  for (const k of ["project", "scheduled", "waiting"] as const) if (t[k]) out[k] = t[k];
  for (const k of LISTS) if (t[k].length > 0) out[k] = t[k];
  return out;
};

export const view = new Hono<{ Bindings: Env }>().get("/tasks.json", async (c) => {
  const { results } = await c.env.DB.prepare("SELECT * FROM tasks ORDER BY slug").all<Row>();
  const rows = results.map(toTask).map((t) => ({
    slug: t.slug,
    status: t.status,
    title: t.title,
    frontmatter: frontmatter(t),
    updated: t.updated,
    body: markdown(t),
  }));
  // no-store, so a reload shows the latest write.
  return c.json(rows, 200, { "cache-control": "no-store" });
});
