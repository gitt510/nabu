// /api/habits: list the habits with whether each is done on a day, add, rename, and delete them, and check them off.
// A check is a row in habit_checks, so checking twice or unchecking twice changes nothing.
import { zValidator } from "@hono/zod-validator";
import { Hono } from "hono";
import { z } from "zod";
import { day, type Habit, HabitCreate, HabitPatch } from "./schema";
import { hook } from "./validate";

const id = z.coerce.number().int();
const params = z.strictObject({ id, date: day });

const exists = async (db: D1Database, id: string) => !!(await db.prepare("SELECT 1 FROM habits WHERE id = ?").bind(id).first());

export const habits = new Hono<{ Bindings: Env }>()
  // Every habit in the order it was added, done or not on ?date=.
  .get("/", zValidator("query", z.object({ date: day }), hook), async (c) => {
    const { results } = await c.env.DB.prepare(
      `SELECT id, title, EXISTS (SELECT 1 FROM habit_checks WHERE habit_id = habits.id AND date = ?) AS done
       FROM habits ORDER BY id`,
    )
      .bind(c.req.valid("query").date)
      .all<{ id: number; title: string; done: number }>();
    return c.json(results.map((h): Habit => ({ ...h, done: !!h.done })));
  })

  .post("/", zValidator("json", HabitCreate, hook), async (c) => {
    const { title } = c.req.valid("json");
    const row = await c.env.DB.prepare("INSERT INTO habits (title, created) VALUES (?, ?) RETURNING id")
      .bind(title, new Date().toISOString())
      .first<{ id: number }>();
    return c.json({ id: row!.id, title, done: false } satisfies Habit, 201);
  })

  .patch("/:id", zValidator("param", z.strictObject({ id }), hook), zValidator("json", HabitPatch, hook), async (c) => {
    const { id } = c.req.valid("param");
    if (!(await exists(c.env.DB, String(id)))) return c.json({ error: "no such habit" }, 404);
    await c.env.DB.prepare("UPDATE habits SET title = ? WHERE id = ?").bind(c.req.valid("json").title, id).run();
    return c.body(null, 204);
  })

  // Deleting a habit deletes its checks with it.
  .delete("/:id", async (c) => {
    const id = c.req.param("id");
    if (!(await exists(c.env.DB, id))) return c.json({ error: "no such habit" }, 404);
    await c.env.DB.batch([
      c.env.DB.prepare("DELETE FROM habit_checks WHERE habit_id = ?").bind(id),
      c.env.DB.prepare("DELETE FROM habits WHERE id = ?").bind(id),
    ]);
    return c.body(null, 204);
  })

  .put("/:id/checks/:date", zValidator("param", params, hook), async (c) => {
    const { id, date } = c.req.valid("param");
    if (!(await exists(c.env.DB, String(id)))) return c.json({ error: "no such habit" }, 404);
    await c.env.DB.prepare("INSERT OR IGNORE INTO habit_checks (habit_id, date) VALUES (?, ?)").bind(id, date).run();
    return c.body(null, 204);
  })

  .delete("/:id/checks/:date", zValidator("param", params, hook), async (c) => {
    const { id, date } = c.req.valid("param");
    if (!(await exists(c.env.DB, String(id)))) return c.json({ error: "no such habit" }, 404);
    await c.env.DB.prepare("DELETE FROM habit_checks WHERE habit_id = ? AND date = ?").bind(id, date).run();
    return c.body(null, 204);
  });
