import { env, exports } from "cloudflare:workers";
import { beforeEach, describe, expect, it } from "vitest";
import type { Item } from "../schema";
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

const add = async (task_id: number, text: string) =>
  (await (await call("POST", "/api/items", { task_id, text })).json()) as Item;
const list = async (task_id: number) =>
  (await (await call("GET", `/api/items?task_id=${task_id}`)).json()) as Item[];
const newTask = async (slug: string) =>
  (
    (await (await call("POST", "/api/tasks", { slug, title: slug, area: "personal" })).json()) as {
      id: number;
    }
  ).id;

beforeEach(async () => {
  await env.DB.batch([
    env.DB.prepare("DELETE FROM items"),
    env.DB.prepare("DELETE FROM tasks"),
    env.DB.prepare("DELETE FROM events"),
  ]);
});

describe("/api/items", () => {
  it("adds lines last in their task's checklist, not done, and lists one task's only", async () => {
    const tokium = await newTask("tokium-ipo");
    const other = await newTask("other");
    const a = await add(tokium, "SBI証券");
    const b = await add(tokium, "楽天証券");
    await add(other, "elsewhere");
    expect(a).toMatchObject({ task_id: tokium, position: 1, done: false });
    expect(b.position).toBe(2);
    expect((await list(tokium)).map((i) => i.text)).toEqual(["SBI証券", "楽天証券"]);
  });

  it("refuses a line out of shape or under a task that does not exist with 422", async () => {
    const id = await newTask("t");
    expect((await call("POST", "/api/items", { task_id: id, text: " " })).status).toBe(422);
    expect((await call("POST", "/api/items", { task_id: 999, text: "a" })).status).toBe(422);
    expect((await call("GET", "/api/items")).status).toBe(422);
  });

  it("checks, renames, and moves a line within the checklist", async () => {
    const id = await newTask("t");
    const a = await add(id, "SBI証券");
    await add(id, "楽天証券");
    const res = await call("PATCH", `/api/items/${a.id}`, { done: true, text: "SBI", position: 3 });
    expect(res.status).toBe(200);
    expect((await list(id)).map((i) => [i.text, i.done])).toEqual([
      ["楽天証券", false],
      ["SBI", true],
    ]);
    expect((await call("PATCH", `/api/items/${a.id}`, { task_id: 2 })).status).toBe(422);
    expect((await call("PATCH", "/api/items/999", { done: true })).status).toBe(404);
  });

  it("deletes a line, and a task with its checklist", async () => {
    const id = await newTask("t");
    const a = await add(id, "SBI証券");
    await add(id, "楽天証券");
    expect((await call("DELETE", `/api/items/${a.id}`)).status).toBe(204);
    expect(await list(id)).toHaveLength(1);
    expect((await call("DELETE", `/api/tasks/${id}`)).status).toBe(204);
    expect(await env.DB.prepare("SELECT COUNT(*) AS n FROM items").first("n")).toBe(0);
  });
});
