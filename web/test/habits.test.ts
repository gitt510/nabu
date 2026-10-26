import { env, exports } from "cloudflare:workers";
import { beforeEach, describe, expect, it } from "vitest";
import { token } from "./access";

const call = async (method: string, path: string, body?: unknown) =>
  exports.default.fetch(
    new Request(`https://nabu.test${path}`, {
      method,
      headers: {
        "cf-access-jwt-assertion": await token(),
        ...(body === undefined ? {} : { "content-type": "application/json" }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  );

const list = async (date: string) => (await (await call("GET", `/api/habits?date=${date}`)).json()) as unknown[];

// Storage outlives a test within the file; start each one empty.
beforeEach(async () => {
  await env.DB.batch([env.DB.prepare("DELETE FROM habit_checks"), env.DB.prepare("DELETE FROM habits")]);
});

describe("/api/habits", () => {
  it("adds a habit, not yet done", async () => {
    const res = await call("POST", "/api/habits", { title: "do some exercise" });
    expect(res.status).toBe(201);
    const habit = (await res.json()) as { id: number };
    expect(habit).toEqual({ id: habit.id, title: "do some exercise", done: false });
    expect(await list("2026-10-26")).toEqual([habit]);
  });

  it("refuses a habit out of shape and a day that does not exist with 422", async () => {
    expect((await call("POST", "/api/habits", { title: " " })).status).toBe(422);
    expect((await call("POST", "/api/habits", { title: "read", minutes: 5 })).status).toBe(422);
    expect((await call("GET", "/api/habits")).status).toBe(422);
    expect((await call("GET", "/api/habits?date=today")).status).toBe(422);
  });

  it("checks a habit on one day only, and checking twice or unchecking twice changes nothing", async () => {
    const { id } = (await (await call("POST", "/api/habits", { title: "read" })).json()) as { id: number };
    for (let i = 0; i < 2; i++) expect((await call("PUT", `/api/habits/${id}/checks/2026-10-26`)).status).toBe(204);
    expect(await list("2026-10-26")).toEqual([{ id, title: "read", done: true }]);
    expect(await list("2026-10-27")).toEqual([{ id, title: "read", done: false }]);

    for (let i = 0; i < 2; i++) expect((await call("DELETE", `/api/habits/${id}/checks/2026-10-26`)).status).toBe(204);
    expect(await list("2026-10-26")).toEqual([{ id, title: "read", done: false }]);
  });

  it("renames a habit and keeps its checks", async () => {
    const { id } = (await (await call("POST", "/api/habits", { title: "monst(main) do event quest" })).json()) as { id: number };
    await call("PUT", `/api/habits/${id}/checks/2026-10-26`);
    expect((await call("PATCH", `/api/habits/${id}`, { title: "Monst main: clear event quests" })).status).toBe(204);
    expect(await list("2026-10-26")).toEqual([{ id, title: "Monst main: clear event quests", done: true }]);
    expect((await call("PATCH", `/api/habits/${id}`, { title: " " })).status).toBe(422);
    expect((await call("PATCH", "/api/habits/999", { title: "read" })).status).toBe(404);
  });

  it("deletes a habit with its checks", async () => {
    const { id } = (await (await call("POST", "/api/habits", { title: "read" })).json()) as { id: number };
    await call("PUT", `/api/habits/${id}/checks/2026-10-26`);
    expect((await call("DELETE", `/api/habits/${id}`)).status).toBe(204);
    expect(await list("2026-10-26")).toEqual([]);
    expect(await env.DB.prepare("SELECT * FROM habit_checks").all().then((r) => r.results)).toEqual([]);
    expect((await call("DELETE", `/api/habits/${id}`)).status).toBe(404);
    expect((await call("PUT", `/api/habits/${id}/checks/2026-10-26`)).status).toBe(404);
  });
});
