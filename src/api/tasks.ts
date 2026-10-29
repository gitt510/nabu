// /api/tasks: list, read, create, change, and delete tasks in D1. Every write appends an event in the
// same batch, so the row and its history never disagree.
import { createRoute, OpenAPIHono, z } from "@hono/zod-openapi";
import { LISTS, SECTIONS, STATUSES, Task, TaskCreate, TaskPatch, whole } from "./schema";
import { body, failure, hook, invalid, json } from "./validate";

type Row = Omit<Task, (typeof LISTS)[number] | (typeof SECTIONS)[number]> & { id: number } & Record<
    (typeof LISTS)[number] | (typeof SECTIONS)[number],
    string
  >;

const JSON_COLUMNS = [...LISTS, ...SECTIONS];
const COLUMNS = ["slug", "title", "status", "area", "project", "scheduled", "waiting", ...JSON_COLUMNS, "created", "updated"] as const;

const toTask = (row: Row): Task => {
  const t = { ...row } as Record<string, unknown>;
  for (const k of JSON_COLUMNS) t[k] = JSON.parse(row[k]);
  return t as Task;
};

const fromTask = (t: Omit<Task, "id">): unknown[] =>
  COLUMNS.map((k) => ((JSON_COLUMNS as readonly string[]).includes(k) ? JSON.stringify(t[k]) : t[k]));

// WORKFLOW orders rows by status in the order STATUSES gives them.
const WORKFLOW = `CASE status ${STATUSES.map((s, i) => `WHEN '${s}' THEN ${i}`).join(" ")} END`;

const now = () => new Date().toISOString();

const find = (db: D1Database, slug: string) => db.prepare("SELECT * FROM tasks WHERE slug = ?").bind(slug).first<Row>();

// event records a write against the task that holds slug when the statement runs, so a batch places it
// after an insert or rename and before a delete.
const event = (db: D1Database, action: string, slug: string, before: Omit<Task, "id"> | null, after: Omit<Task, "id"> | null) =>
  db
    .prepare(
      "INSERT INTO events (task_id, at, action, before, after) VALUES ((SELECT id FROM tasks WHERE slug = ?), ?, ?, ?, ?)",
    )
    .bind(slug, now(), action, before && JSON.stringify(before), after && JSON.stringify(after));

const slug = z.strictObject({ slug: z.string() });

export const tasks = new OpenAPIHono<{ Bindings: Env }>({ defaultHook: hook })
  // Every task in workflow order, then by slug: callers narrow them themselves while the list stays small.
  .openapi(createRoute({ method: "get", path: "/", responses: { 200: json(z.array(Task), "every task in workflow order, then by slug") } }), async (c) => {
    const { results } = await c.env.DB.prepare(`SELECT * FROM tasks ORDER BY ${WORKFLOW}, slug`).all<Row>();
    return c.json(results.map(toTask), 200);
  })

  .openapi(
    createRoute({
      method: "get",
      path: "/{slug}",
      request: { params: slug },
      responses: { 200: json(Task, "the task"), 404: failure("no such task") },
    }),
    async (c) => {
      const row = await find(c.env.DB, c.req.valid("param").slug);
      return row ? c.json(toTask(row), 200) : c.json({ error: "no such task" }, 404);
    },
  )

  .openapi(
    createRoute({
      method: "post",
      path: "/",
      request: { body: body(TaskCreate, "the new task, filed in inbox") },
      responses: { 201: json(Task, "the task as stored"), 409: failure("the slug is taken"), 422: failure("the task breaks the contract") },
    }),
    async (c) => {
      const at = now();
      const fields = c.req.valid("json");
      const task: Omit<Task, "id"> = {
        ...fields,
        status: "inbox",
        project: fields.project ?? null,
        scheduled: fields.scheduled ?? null,
        waiting: fields.waiting ?? null,
        created: at,
        updated: at,
      };
      if (await find(c.env.DB, task.slug)) return c.json({ error: `${task.slug} already exists` }, 409);
      const insert = c.env.DB.prepare(
        `INSERT INTO tasks (${COLUMNS.join(", ")}) VALUES (${COLUMNS.map(() => "?").join(", ")}) RETURNING id`,
      ).bind(...fromTask(task));
      const [{ results }] = await c.env.DB.batch<{ id: number }>([insert, event(c.env.DB, "create", task.slug, null, task)]);
      return c.json({ id: results[0].id, ...task }, 201);
    },
  )

  .openapi(
    createRoute({
      method: "patch",
      path: "/{slug}",
      request: { params: slug, body: body(TaskPatch, "the fields to change") },
      responses: {
        200: json(Task, "the task as stored"),
        404: failure("no such task"),
        409: failure("the new slug is taken"),
        422: failure("the task breaks the contract"),
      },
    }),
    async (c) => {
      const row = await find(c.env.DB, c.req.valid("param").slug);
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
      return c.json(after, 200);
    },
  )

  .openapi(
    createRoute({
      method: "delete",
      path: "/{slug}",
      request: { params: slug },
      responses: { 204: { description: "deleted; its events stay" }, 404: failure("no such task") },
    }),
    async (c) => {
      const row = await find(c.env.DB, c.req.valid("param").slug);
      if (!row) return c.json({ error: "no such task" }, 404);
      await c.env.DB.batch([
        event(c.env.DB, "delete", row.slug, toTask(row), null),
        c.env.DB.prepare("DELETE FROM tasks WHERE id = ?").bind(row.id),
      ]);
      return c.body(null, 204);
    },
  );
