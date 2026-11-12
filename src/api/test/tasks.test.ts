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

const json = async (res: Response) => ({ status: res.status, body: (await res.json()) as any });

const task = {
  slug: "move-runtime",
  title: "runtime を新しい version へ移行",
  area: "work",
};

const events = async () =>
  (
    await env.DB.prepare("SELECT task_id, action FROM events ORDER BY id").all<{
      task_id: number;
      action: string;
    }>()
  ).results;

// Storage outlives a test within the file; start each one empty, ids from 1.
beforeEach(async () => {
  await env.DB.batch([
    env.DB.prepare("DELETE FROM tasks"),
    env.DB.prepare("DELETE FROM events"),
    env.DB.prepare("DELETE FROM sqlite_sequence WHERE name = 'tasks'"),
  ]);
});

describe("/api/tasks", () => {
  it("creates a todo task and reads it back", async () => {
    const created = await json(await call("POST", "/api/tasks", task));
    expect(created.status).toBe(201);
    expect(created.body).toMatchObject({
      ...task,
      id: 1,
      status: "todo",
      project: null,
      tickets: [],
      why_dropped: [],
      memo: [],
    });
    expect(created.body.created).toBe(created.body.updated);

    expect(await json(await call("GET", "/api/tasks/1"))).toEqual({
      status: 200,
      body: created.body,
    });
    expect((await json(await call("GET", "/api/tasks"))).body).toEqual([created.body]);
    expect(await events()).toEqual([{ task_id: 1, action: "create" }]);
  });

  it("keeps the memo a patch writes, line for line", async () => {
    await call("POST", "/api/tasks", task);
    const memo = ["schema に memo を足した", "45 文字は少し短いかも"];
    expect((await json(await call("PATCH", "/api/tasks/1", { memo }))).body.memo).toEqual(memo);
    expect((await json(await call("GET", "/api/tasks/1"))).body.memo).toEqual(memo);
  });

  it("refuses a task out of shape with 422", async () => {
    for (const bad of [
      { ...task, slug: "Bad Slug" },
      { ...task, area: "hobby" },
      { ...task, scheduled: "tomorrow" },
      { ...task, tickets: ["http://example.com/1"] },
      { ...task, links: ["https://example.com/1", "https://example.com/1"] },
      { ...task, status: "done" },
      { ...task, why_dropped: ["今は使い道がない"] },
      { ...task, memo: "free notes" },
    ]) {
      const res = await json(await call("POST", "/api/tasks", bad));
      expect(res.status, JSON.stringify(bad)).toBe(422);
      expect(res.body.error).toBeTypeOf("string");
    }
    expect((await json(await call("GET", "/api/tasks"))).body).toEqual([]);
  });

  it("names every problem of a task, one per line", async () => {
    const res = await json(
      await call("POST", "/api/tasks", { ...task, slug: "Bad Slug", area: "hobby" }),
    );
    expect(res.status).toBe(422);
    expect(res.body.error.split("\n")).toEqual([
      expect.stringMatching(/^slug: /),
      expect.stringMatching(/^area: /),
    ]);
  });

  it("refuses a memo line out of shape with 422, and accepts one of exactly the limit", async () => {
    await call("POST", "/api/tasks", task);
    for (const line of ["あ".repeat(46), "- 箇条書きの記号", "## AI memo", " "])
      expect((await call("PATCH", "/api/tasks/1", { memo: [line] })).status, line).toBe(422);
    expect((await call("PATCH", "/api/tasks/1", { memo: ["あ".repeat(45)] })).status).toBe(200);
  });

  it("refuses a slug that is taken with 409", async () => {
    await call("POST", "/api/tasks", task);
    expect((await call("POST", "/api/tasks", task)).status).toBe(409);
    await call("POST", "/api/tasks", { ...task, slug: "other" });
    expect((await call("PATCH", "/api/tasks/2", { slug: "move-runtime" })).status).toBe(409);
  });

  it("changes fields, drops optional ones with null, and records the row before and after", async () => {
    await call("POST", "/api/tasks", { ...task, project: "nabu", waiting: "review" });
    const res = await json(
      await call("PATCH", "/api/tasks/1", {
        status: "done",
        project: null,
        waiting: null,
        memo: ["検証 → 本番の順で切り替わっている"],
      }),
    );
    expect(res.status).toBe(200);
    expect(res.body).toMatchObject({
      status: "done",
      project: null,
      waiting: null,
      memo: ["検証 → 本番の順で切り替わっている"],
    });

    const row = await env.DB.prepare(
      "SELECT before, after FROM events WHERE action = 'update'",
    ).first<{ before: string; after: string }>();
    expect(JSON.parse(row!.before)).toMatchObject({ status: "todo", project: "nabu" });
    expect(JSON.parse(row!.after)).toMatchObject({ status: "done", project: null });
  });

  it("renames a task and keeps its id and its history", async () => {
    await call("POST", "/api/tasks", task);
    expect((await call("PATCH", "/api/tasks/1", { slug: "upgrade-runtime" })).status).toBe(200);
    expect((await json(await call("GET", "/api/tasks/1"))).body.slug).toBe("upgrade-runtime");
    expect(await events()).toEqual([
      { task_id: 1, action: "create" },
      { task_id: 1, action: "update" },
    ]);
  });

  it("refuses to drop a task that does not say why", async () => {
    await call("POST", "/api/tasks", task);
    const res = await json(await call("PATCH", "/api/tasks/1", { status: "dropped" }));
    expect(res).toEqual({ status: 422, body: { error: "a dropped task needs why_dropped" } });

    const ok = await call("PATCH", "/api/tasks/1", {
      status: "dropped",
      why_dropped: ["今は使い道がない"],
    });
    expect(ok.status).toBe(200);

    // leaving dropped takes the reason with it
    const kept = await json(await call("PATCH", "/api/tasks/1", { status: "todo" }));
    expect(kept).toEqual({ status: 422, body: { error: "a todo task has no why_dropped" } });
    expect((await call("PATCH", "/api/tasks/1", { status: "todo", why_dropped: [] })).status).toBe(
      200,
    );
  });

  it("lists every task in workflow order, then by slug", async () => {
    for (const slug of ["d", "c", "b", "a"]) await call("POST", "/api/tasks", { ...task, slug });
    await call("PATCH", "/api/tasks/4", { status: "dropped", why_dropped: ["不要"] });
    await call("PATCH", "/api/tasks/3", { status: "done" });
    const { body } = await json(await call("GET", "/api/tasks"));
    expect(body.map((t: any) => [t.slug, t.status])).toEqual([
      ["c", "todo"],
      ["d", "todo"],
      ["b", "done"],
      ["a", "dropped"],
    ]);
  });

  it("narrows the list to the task with a slug", async () => {
    for (const slug of ["a", "b"]) await call("POST", "/api/tasks", { ...task, slug });
    expect(
      (await json(await call("GET", "/api/tasks?slug=b"))).body.map((t: any) => [t.id, t.slug]),
    ).toEqual([[2, "b"]]);
    expect((await json(await call("GET", "/api/tasks?slug=z"))).body).toEqual([]);
  });

  it("deletes a task and keeps its events", async () => {
    await call("POST", "/api/tasks", task);
    expect((await call("DELETE", "/api/tasks/1")).status).toBe(204);
    expect((await call("DELETE", "/api/tasks/1")).status).toBe(404);
    expect((await call("PATCH", "/api/tasks/1", { title: "x" })).status).toBe(404);
    expect(await events()).toEqual([
      { task_id: 1, action: "create" },
      { task_id: 1, action: "delete" },
    ]);
  });

  it("never gives the id of a deleted task to a new one", async () => {
    await call("POST", "/api/tasks", task);
    await call("DELETE", "/api/tasks/1");
    expect((await json(await call("POST", "/api/tasks", task))).body.id).toBe(2);
  });
});

describe("/api/schema", () => {
  it("returns the task contract as JSON Schema, the line limit and the areas in it", async () => {
    const res = await json(await call("GET", "/api/schema"));
    expect(res.status).toBe(200);
    expect(res.body.patch.properties.memo.items.maxLength).toBe(45);
    for (const schema of [res.body.create, res.body.patch]) {
      expect(schema.properties.area).toEqual({ $ref: "#/$defs/Area" });
      expect(schema.$defs.Area.enum).toEqual(["work", "personal"]);
    }
  });
});
