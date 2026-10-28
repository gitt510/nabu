// /api/habits: list the habits with whether each is done on a day, add, rename, and delete them, and check them off.
// A check is a row in habit_checks, so checking twice or unchecking twice changes nothing.
import { createRoute, OpenAPIHono, z } from "@hono/zod-openapi";
import { day, Habit, HabitCreate, HabitPatch } from "./schema";
import { body, failure, hook, json } from "./validate";

const id = z.coerce.number().int();
const params = z.strictObject({ id, date: day });

const exists = async (db: D1Database, id: number) => !!(await db.prepare("SELECT 1 FROM habits WHERE id = ?").bind(id).first());

const noSuchHabit = failure("no such habit");

export const habits = new OpenAPIHono<{ Bindings: Env }>({ defaultHook: hook })
  // Every habit in the order it was added, done or not on ?date=.
  .openapi(
    createRoute({
      method: "get",
      path: "/",
      request: { query: z.object({ date: day }) },
      responses: { 200: json(z.array(Habit), "every habit, done or not on date"), 422: failure("date is missing or not a day") },
    }),
    async (c) => {
      const { results } = await c.env.DB.prepare(
        `SELECT id, title, EXISTS (SELECT 1 FROM habit_checks WHERE habit_id = habits.id AND date = ?) AS done
         FROM habits ORDER BY id`,
      )
        .bind(c.req.valid("query").date)
        .all<{ id: number; title: string; done: number }>();
      return c.json(results.map((h): Habit => ({ ...h, done: !!h.done })), 200);
    },
  )

  .openapi(
    createRoute({
      method: "post",
      path: "/",
      request: { body: body(HabitCreate, "the new habit") },
      responses: { 201: json(Habit, "the habit, not yet done"), 422: failure("the habit is out of shape") },
    }),
    async (c) => {
      const { title } = c.req.valid("json");
      const row = await c.env.DB.prepare("INSERT INTO habits (title, created) VALUES (?, ?) RETURNING id")
        .bind(title, new Date().toISOString())
        .first<{ id: number }>();
      return c.json({ id: row!.id, title, done: false } satisfies Habit, 201);
    },
  )

  .openapi(
    createRoute({
      method: "patch",
      path: "/{id}",
      request: { params: z.strictObject({ id }), body: body(HabitPatch, "the new title") },
      responses: { 204: { description: "renamed; its checks stay" }, 404: noSuchHabit, 422: failure("the title is out of shape") },
    }),
    async (c) => {
      const { id } = c.req.valid("param");
      if (!(await exists(c.env.DB, id))) return c.json({ error: "no such habit" }, 404);
      await c.env.DB.prepare("UPDATE habits SET title = ? WHERE id = ?").bind(c.req.valid("json").title, id).run();
      return c.body(null, 204);
    },
  )

  // Deleting a habit deletes its checks with it.
  .openapi(
    createRoute({
      method: "delete",
      path: "/{id}",
      request: { params: z.strictObject({ id }) },
      responses: { 204: { description: "deleted with its checks" }, 404: noSuchHabit },
    }),
    async (c) => {
      const { id } = c.req.valid("param");
      if (!(await exists(c.env.DB, id))) return c.json({ error: "no such habit" }, 404);
      await c.env.DB.batch([
        c.env.DB.prepare("DELETE FROM habit_checks WHERE habit_id = ?").bind(id),
        c.env.DB.prepare("DELETE FROM habits WHERE id = ?").bind(id),
      ]);
      return c.body(null, 204);
    },
  )

  .openapi(
    createRoute({
      method: "put",
      path: "/{id}/checks/{date}",
      request: { params },
      responses: { 204: { description: "done on date" }, 404: noSuchHabit, 422: failure("date is not a day") },
    }),
    async (c) => {
      const { id, date } = c.req.valid("param");
      if (!(await exists(c.env.DB, id))) return c.json({ error: "no such habit" }, 404);
      await c.env.DB.prepare("INSERT OR IGNORE INTO habit_checks (habit_id, date) VALUES (?, ?)").bind(id, date).run();
      return c.body(null, 204);
    },
  )

  .openapi(
    createRoute({
      method: "delete",
      path: "/{id}/checks/{date}",
      request: { params },
      responses: { 204: { description: "not done on date" }, 404: noSuchHabit, 422: failure("date is not a day") },
    }),
    async (c) => {
      const { id, date } = c.req.valid("param");
      if (!(await exists(c.env.DB, id))) return c.json({ error: "no such habit" }, 404);
      await c.env.DB.prepare("DELETE FROM habit_checks WHERE habit_id = ? AND date = ?").bind(id, date).run();
      return c.body(null, 204);
    },
  );
