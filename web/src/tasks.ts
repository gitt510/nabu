// /api/tasks: list, read, create, change, and delete tasks in D1. Every write appends an event in the
// same batch, so the row and its history never disagree.
import { zValidator } from "@hono/zod-validator";
import { Hono, type Context } from "hono";
import { LISTS, SECTIONS, type Task, TaskCreate, TaskPatch, whole } from "./schema";

type Row = Omit<Task, (typeof LISTS)[number] | (typeof SECTIONS)[number]> & { id: number } & Record<
    (typeof LISTS)[number] | (typeof SECTIONS)[number],
    string
  >;

const JSON_COLUMNS = [...LISTS, ...SECTIONS];
const COLUMNS = ["slug", "title", "status", "area", "project", "scheduled", "waiting", ...JSON_COLUMNS, "created", "updated"] as const;

const toTask = ({ id: _, ...row }: Row): Task => {
  const t = { ...row } as Record<string, unknown>;
  for (const k of JSON_COLUMNS) t[k] = JSON.parse(row[k]);
  return t as Task;
};

const fromTask = (t: Task): unknown[] =>
  COLUMNS.map((k) => ((JSON_COLUMNS as readonly string[]).includes(k) ? JSON.stringify(t[k]) : t[k]));

// invalid answers a request that breaks the contract: 422 with the first problem as text.
const invalid = (c: Context, error: string) => c.json({ error }, 422);

type Issues = { issues: { path: PropertyKey[]; message: string }[] };

const hook = (result: { success: boolean; error?: Issues }, c: Context) => {
  if (!result.success && result.error) {
    const i = result.error.issues[0];
    return invalid(c, `${i.path.map(String).join(".") || "body"}: ${i.message}`);
  }
};

const now = () => new Date().toISOString();

const find = (db: D1Database, slug: string) => db.prepare("SELECT * FROM tasks WHERE slug = ?").bind(slug).first<Row>();

// event records a write against the task that holds slug when the statement runs, so a batch places it
// after an insert or rename and before a delete.
const event = (db: D1Database, action: string, slug: string, before: Task | null, after: Task | null) =>
  db
    .prepare(
      "INSERT INTO events (task_id, at, action, before, after) VALUES ((SELECT id FROM tasks WHERE slug = ?), ?, ?, ?, ?)",
    )
    .bind(slug, now(), action, before && JSON.stringify(before), after && JSON.stringify(after));

export const tasks = new Hono<{ Bindings: Env }>()
  // Every task, ordered by slug: callers narrow them themselves while the list stays small.
  .get("/", async (c) => {
    const { results } = await c.env.DB.prepare("SELECT * FROM tasks ORDER BY slug").all<Row>();
    return c.json(results.map(toTask));
  })

  .get("/:slug", async (c) => {
    const row = await find(c.env.DB, c.req.param("slug"));
    return row ? c.json(toTask(row)) : c.json({ error: "no such task" }, 404);
  })

  .post("/", zValidator("json", TaskCreate, hook), async (c) => {
    const at = now();
    const task: Task = {
      ...c.req.valid("json"),
      status: "inbox",
      project: c.req.valid("json").project ?? null,
      scheduled: c.req.valid("json").scheduled ?? null,
      waiting: c.req.valid("json").waiting ?? null,
      created: at,
      updated: at,
    };
    if (await find(c.env.DB, task.slug)) return c.json({ error: `${task.slug} already exists` }, 409);
    const insert = c.env.DB.prepare(
      `INSERT INTO tasks (${COLUMNS.join(", ")}) VALUES (${COLUMNS.map(() => "?").join(", ")})`,
    ).bind(...fromTask(task));
    await c.env.DB.batch([insert, event(c.env.DB, "create", task.slug, null, task)]);
    return c.json(task, 201);
  })

  .patch("/:slug", zValidator("json", TaskPatch, hook), async (c) => {
    const row = await find(c.env.DB, c.req.param("slug"));
    if (!row) return c.json({ error: "no such task" }, 404);
    const before = toTask(row);
    const after: Task = { ...before, ...c.req.valid("json"), updated: now() };
    const problem = whole(after);
    if (problem) return invalid(c, problem);
    if (after.slug !== before.slug && (await find(c.env.DB, after.slug))) {
      return c.json({ error: `${after.slug} already exists` }, 409);
    }
    const update = c.env.DB.prepare(`UPDATE tasks SET ${COLUMNS.map((k) => `${k} = ?`).join(", ")} WHERE id = ?`).bind(
      ...fromTask(after),
      row.id,
    );
    await c.env.DB.batch([update, event(c.env.DB, "update", after.slug, before, after)]);
    return c.json(after);
  })

  .delete("/:slug", async (c) => {
    const row = await find(c.env.DB, c.req.param("slug"));
    if (!row) return c.json({ error: "no such task" }, 404);
    await c.env.DB.batch([
      event(c.env.DB, "delete", row.slug, toTask(row), null),
      c.env.DB.prepare("DELETE FROM tasks WHERE id = ?").bind(row.id),
    ]);
    return c.body(null, 204);
  });
