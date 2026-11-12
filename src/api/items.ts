// /api/items: an outline of lines, each under a parent or none. The page reads them all at once and builds
// the tree itself; a write moves or changes one line, and deleting a line deletes the lines under it.
import { createRoute, OpenAPIHono, z } from "@hono/zod-openapi";
import { Item, ItemCreate, ItemPatch } from "./schema";
import { body, failure, hook, json } from "./validate";

type Row = Omit<Item, "done"> & { done: number };
const toItem = (r: Row): Item => ({ ...r, done: !!r.done });

const SELECT = "SELECT id, parent_id, position, text, done, due, created, updated FROM items";
const find = (db: D1Database, id: number) =>
  db.prepare(`${SELECT} WHERE id = ?`).bind(id).first<Row>();

// last is the position after the last line under parent (or among the roots).
const last = async (db: D1Database, parent: number | null) =>
  ((
    await db
      .prepare("SELECT MAX(position) AS p FROM items WHERE parent_id IS ?")
      .bind(parent)
      .first<{ p: number | null }>()
  )?.p ?? 0) + 1;

// SUBTREE is id and every line under it, for ? = id.
const SUBTREE = `WITH RECURSIVE sub(id) AS (
  SELECT ? UNION ALL SELECT items.id FROM items JOIN sub ON items.parent_id = sub.id
)`;

const params = z.strictObject({ id: z.coerce.number().int() });
const noSuchItem = failure("no such item");
const now = () => new Date().toISOString();

export const items = new OpenAPIHono<{ Bindings: Env }>({ defaultHook: hook })
  .openapi(
    createRoute({
      method: "get",
      path: "/",
      responses: { 200: json(z.array(Item), "every line, siblings in position order") },
    }),
    async (c) => {
      const { results } = await c.env.DB.prepare(
        `${SELECT} ORDER BY parent_id, position`,
      ).all<Row>();
      return c.json(results.map(toItem), 200);
    },
  )

  .openapi(
    createRoute({
      method: "post",
      path: "/",
      request: { body: body(ItemCreate, "the new line") },
      responses: {
        201: json(Item, "the line, not done, after its last sibling"),
        422: failure("the line is out of shape, or its parent does not exist"),
      },
    }),
    async (c) => {
      const { text, parent_id, due } = c.req.valid("json");
      if (parent_id !== null && !(await find(c.env.DB, parent_id)))
        return c.json({ error: "parent_id: no such item" }, 422);
      const at = now();
      const row = await c.env.DB.prepare(
        `INSERT INTO items (parent_id, position, text, done, due, created, updated)
         VALUES (?, ?, ?, 0, ?, ?, ?) RETURNING *`,
      )
        .bind(parent_id, await last(c.env.DB, parent_id), text, due, at, at)
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
        422: failure(
          "a field is out of shape, or the new parent does not exist or lies under the line",
        ),
      },
    }),
    async (c) => {
      const { id } = c.req.valid("param");
      const before = await find(c.env.DB, id);
      if (!before) return c.json({ error: "no such item" }, 404);
      const fields = c.req.valid("json");
      const next = { ...toItem(before), ...fields };
      if (fields.parent_id !== undefined && fields.parent_id !== before.parent_id) {
        if (fields.parent_id !== null) {
          if (!(await find(c.env.DB, fields.parent_id)))
            return c.json({ error: "parent_id: no such item" }, 422);
          const under = await c.env.DB.prepare(`${SUBTREE} SELECT 1 FROM sub WHERE id = ?`)
            .bind(id, fields.parent_id)
            .first();
          if (under) return c.json({ error: "parent_id: must not be the line or under it" }, 422);
        }
        if (fields.position === undefined) next.position = await last(c.env.DB, fields.parent_id);
      }
      const row = await c.env.DB.prepare(
        `UPDATE items SET parent_id = ?, position = ?, text = ?, done = ?, due = ?, updated = ?
         WHERE id = ? RETURNING *`,
      )
        .bind(next.parent_id, next.position, next.text, next.done ? 1 : 0, next.due, now(), id)
        .first<Row>();
      return c.json(toItem(row!), 200);
    },
  )

  .openapi(
    createRoute({
      method: "delete",
      path: "/{id}",
      request: { params },
      responses: { 204: { description: "deleted with every line under it" }, 404: noSuchItem },
    }),
    async (c) => {
      const { id } = c.req.valid("param");
      if (!(await find(c.env.DB, id))) return c.json({ error: "no such item" }, 404);
      await c.env.DB.prepare(`${SUBTREE} DELETE FROM items WHERE id IN (SELECT id FROM sub)`)
        .bind(id)
        .run();
      return c.body(null, 204);
    },
  );
