// /api/items: the checklist of a task. A line is added last, checked, renamed, moved within the checklist,
// and deleted; deleting the task deletes its checklist.
import { createRoute, OpenAPIHono, z } from "@hono/zod-openapi";
import { Item, ItemCreate, ItemPatch } from "./schema";
import { body, failure, hook, json } from "./validate";

type Row = Omit<Item, "done"> & { done: number };
const toItem = (r: Row): Item => ({ ...r, done: !!r.done });

const SELECT = "SELECT id, task_id, position, text, done, created, updated FROM items";
const find = (db: D1Database, id: number) =>
  db.prepare(`${SELECT} WHERE id = ?`).bind(id).first<Row>();

const params = z.strictObject({ id: z.coerce.number().int() });
const noSuchItem = failure("no such item");
const now = () => new Date().toISOString();

export const items = new OpenAPIHono<{ Bindings: Env }>({ defaultHook: hook })
  .openapi(
    createRoute({
      method: "get",
      path: "/",
      request: { query: z.strictObject({ task_id: z.coerce.number().int() }) },
      responses: {
        200: json(z.array(Item), "the checklist of the task, in position order"),
        422: failure("task_id is missing"),
      },
    }),
    async (c) => {
      const { results } = await c.env.DB.prepare(`${SELECT} WHERE task_id = ? ORDER BY position`)
        .bind(c.req.valid("query").task_id)
        .all<Row>();
      return c.json(results.map(toItem), 200);
    },
  )

  .openapi(
    createRoute({
      method: "post",
      path: "/",
      request: { body: body(ItemCreate, "the new line") },
      responses: {
        201: json(Item, "the line, not done, last in the checklist"),
        422: failure("the line is out of shape, or its task does not exist"),
      },
    }),
    async (c) => {
      const { task_id, text } = c.req.valid("json");
      if (!(await c.env.DB.prepare("SELECT 1 FROM tasks WHERE id = ?").bind(task_id).first()))
        return c.json({ error: "task_id: no such task" }, 422);
      const at = now();
      const row = await c.env.DB.prepare(
        `INSERT INTO items (task_id, position, text, done, created, updated)
         VALUES (?, (SELECT COALESCE(MAX(position), 0) + 1 FROM items WHERE task_id = ?), ?, 0, ?, ?)
         RETURNING *`,
      )
        .bind(task_id, task_id, text, at, at)
        .first<Row>();
      return c.json(toItem(row!), 201);
    },
  )

  .openapi(
    createRoute({
      method: "patch",
      path: "/{id}",
      request: { params, body: body(ItemPatch, "the fields to change") },
      responses: {
        200: json(Item, "the line as it is now"),
        404: noSuchItem,
        422: failure("a field is out of shape"),
      },
    }),
    async (c) => {
      const { id } = c.req.valid("param");
      const before = await find(c.env.DB, id);
      if (!before) return c.json({ error: "no such item" }, 404);
      const next = { ...toItem(before), ...c.req.valid("json") };
      const row = await c.env.DB.prepare(
        "UPDATE items SET position = ?, text = ?, done = ?, updated = ? WHERE id = ? RETURNING *",
      )
        .bind(next.position, next.text, next.done ? 1 : 0, now(), id)
        .first<Row>();
      return c.json(toItem(row!), 200);
    },
  )

  .openapi(
    createRoute({
      method: "delete",
      path: "/{id}",
      request: { params },
      responses: { 204: { description: "deleted" }, 404: noSuchItem },
    }),
    async (c) => {
      const { id } = c.req.valid("param");
      if (!(await find(c.env.DB, id))) return c.json({ error: "no such item" }, 404);
      await c.env.DB.prepare("DELETE FROM items WHERE id = ?").bind(id).run();
      return c.body(null, 204);
    },
  );
