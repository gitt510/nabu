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

const add = async (fields: Record<string, unknown>) =>
  (await (await call("POST", "/api/items", fields)).json()) as Item;
const list = async () => (await (await call("GET", "/api/items")).json()) as Item[];
const shape = async () =>
  (await list()).map(({ id, parent_id, text, done }) => ({ id, parent_id, text, done }));

beforeEach(async () => {
  await env.DB.prepare("DELETE FROM items").run();
});

describe("/api/items", () => {
  it("adds a root with a due day and lines under it, each after its last sibling", async () => {
    const root = await add({ text: "TOKIUM IPO", due: "2026-11-17" });
    expect(root).toMatchObject({ parent_id: null, position: 1, done: false, due: "2026-11-17" });
    const a = await add({ text: "A証券", parent_id: root.id });
    const b = await add({ text: "B証券", parent_id: root.id });
    expect([a.position, b.position]).toEqual([1, 2]);
    expect(await shape()).toEqual([
      { id: root.id, parent_id: null, text: "TOKIUM IPO", done: false },
      { id: a.id, parent_id: root.id, text: "A証券", done: false },
      { id: b.id, parent_id: root.id, text: "B証券", done: false },
    ]);
  });

  it("refuses a line out of shape or under a parent that does not exist with 422", async () => {
    expect((await call("POST", "/api/items", { text: " " })).status).toBe(422);
    expect((await call("POST", "/api/items", { text: "a", due: "2026-13-01" })).status).toBe(422);
    expect((await call("POST", "/api/items", { text: "a", parent_id: 999 })).status).toBe(422);
  });

  it("checks a line and renames it", async () => {
    const { id } = await add({ text: "A証券" });
    const res = await call("PATCH", `/api/items/${id}`, { done: true, text: "A証券で申込済" });
    expect(res.status).toBe(200);
    expect(await res.json()).toMatchObject({ id, done: true, text: "A証券で申込済" });
  });

  it("moves a line to the end of a new parent, and refuses a move under itself", async () => {
    const nov = await add({ text: "TOKIUM IPO" });
    const dec = await add({ text: "NOTA IPO" });
    await add({ text: "B証券", parent_id: dec.id });
    const a = await add({ text: "A証券", parent_id: nov.id });

    const moved = (await (
      await call("PATCH", `/api/items/${a.id}`, { parent_id: dec.id })
    ).json()) as Item;
    expect(moved).toMatchObject({ parent_id: dec.id, position: 2 });

    expect((await call("PATCH", `/api/items/${dec.id}`, { parent_id: dec.id })).status).toBe(422);
    expect((await call("PATCH", `/api/items/${dec.id}`, { parent_id: a.id })).status).toBe(422);
    expect((await call("PATCH", "/api/items/999", { done: true })).status).toBe(404);
  });

  it("deletes a line with every line under it", async () => {
    const root = await add({ text: "TOKIUM IPO" });
    const a = await add({ text: "A証券", parent_id: root.id });
    await add({ text: "抽選結果", parent_id: a.id });
    const other = await add({ text: "NOTA IPO" });

    expect((await call("DELETE", `/api/items/${root.id}`)).status).toBe(204);
    expect((await list()).map((i) => i.id)).toEqual([other.id]);
    expect((await call("DELETE", `/api/items/${root.id}`)).status).toBe(404);
  });
});
