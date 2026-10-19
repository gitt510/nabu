import { env, exports } from "cloudflare:workers";
import { beforeEach, describe, expect, it } from "vitest";

const call = (method: string, path: string, body?: unknown) =>
  exports.default.fetch(
    new Request(`https://nabu.test${path}`, {
      method,
      headers: body === undefined ? {} : { "content-type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  );

const json = async (res: Response) => ({ status: res.status, body: (await res.json()) as any });

const task = {
  slug: "move-runtime",
  title: "runtime を新しい version へ移行",
  area: "work",
  situation: ["- 本番と検証が旧 runtime"],
  complication: ["- 旧 runtime はサポートが切れる"],
  question: ["- 期限前に止めずに移れる？"],
  answer: ["- 未定"],
};

const events = async () =>
  (await env.DB.prepare("SELECT task_id, action FROM events ORDER BY id").all<{ task_id: number; action: string }>()).results;

// Storage outlives a test within the file; start each one empty, ids from 1.
beforeEach(async () => {
  await env.DB.batch([env.DB.prepare("DELETE FROM tasks"), env.DB.prepare("DELETE FROM events")]);
});

describe("/api/tasks", () => {
  it("creates a task in inbox and reads it back", async () => {
    const created = await json(await call("POST", "/api/tasks", task));
    expect(created.status).toBe(201);
    expect(created.body).toMatchObject({ ...task, status: "inbox", project: null, tickets: [], why_dropped: [] });
    expect(created.body.created).toBe(created.body.updated);

    expect(await json(await call("GET", "/api/tasks/move-runtime"))).toEqual({ status: 200, body: created.body });
    expect((await json(await call("GET", "/api/tasks"))).body).toEqual([created.body]);
    expect(await events()).toEqual([{ task_id: 1, action: "create" }]);
  });

  it("refuses a task out of shape with 422", async () => {
    for (const bad of [
      { ...task, slug: "Bad Slug" },
      { ...task, area: "hobby" },
      { ...task, situation: ["- " + "あ".repeat(29)] },
      { ...task, situation: ["## AI memo"] },
      { ...task, situation: [" "] },
      { ...task, scheduled: "tomorrow" },
      { ...task, tickets: ["http://example.com/1"] },
      { ...task, links: ["https://example.com/1", "https://example.com/1"] },
      { ...task, status: "done" },
      { ...task, memo: "free notes" },
    ]) {
      const res = await json(await call("POST", "/api/tasks", bad));
      expect(res.status, JSON.stringify(bad)).toBe(422);
      expect(res.body.error).toBeTypeOf("string");
    }
    expect((await json(await call("GET", "/api/tasks"))).body).toEqual([]);
  });

  it("accepts a line of exactly the limit", async () => {
    const res = await call("POST", "/api/tasks", { ...task, situation: ["- " + "あ".repeat(28)] });
    expect(res.status).toBe(201);
  });

  it("refuses a slug that is taken with 409", async () => {
    await call("POST", "/api/tasks", task);
    expect((await call("POST", "/api/tasks", task)).status).toBe(409);
    await call("POST", "/api/tasks", { ...task, slug: "other" });
    expect((await call("PATCH", "/api/tasks/other", { slug: "move-runtime" })).status).toBe(409);
  });

  it("changes fields, drops optional ones with null, and records the row before and after", async () => {
    await call("POST", "/api/tasks", { ...task, project: "nabu", waiting: "review" });
    const res = await json(
      await call("PATCH", "/api/tasks/move-runtime", { status: "doing", project: null, waiting: null, answer: ["- 検証 → 本番の順で切り替える"] }),
    );
    expect(res.status).toBe(200);
    expect(res.body).toMatchObject({ status: "doing", project: null, waiting: null, answer: ["- 検証 → 本番の順で切り替える"] });

    const row = await env.DB.prepare("SELECT before, after FROM events WHERE action = 'update'").first<{ before: string; after: string }>();
    expect(JSON.parse(row!.before)).toMatchObject({ status: "inbox", project: "nabu" });
    expect(JSON.parse(row!.after)).toMatchObject({ status: "doing", project: null });
  });

  it("renames a task and keeps its history on the same id", async () => {
    await call("POST", "/api/tasks", task);
    expect((await call("PATCH", "/api/tasks/move-runtime", { slug: "upgrade-runtime" })).status).toBe(200);
    expect((await call("GET", "/api/tasks/move-runtime")).status).toBe(404);
    expect((await call("GET", "/api/tasks/upgrade-runtime")).status).toBe(200);
    expect(await events()).toEqual([
      { task_id: 1, action: "create" },
      { task_id: 1, action: "update" },
    ]);
  });

  it("refuses to drop a task that does not say why", async () => {
    await call("POST", "/api/tasks", task);
    const res = await json(await call("PATCH", "/api/tasks/move-runtime", { status: "dropped" }));
    expect(res).toEqual({ status: 422, body: { error: "a dropped task needs why_dropped" } });

    const ok = await call("PATCH", "/api/tasks/move-runtime", { status: "dropped", why_dropped: ["- 今は使い道がない"] });
    expect(ok.status).toBe(200);
  });

  it("filters by status and finds text in any section", async () => {
    await call("POST", "/api/tasks", task);
    await call("POST", "/api/tasks", { slug: "other", title: "Other", area: "personal", situation: ["- 100%_done"] });
    await call("PATCH", "/api/tasks/other", { status: "doing" });

    const slugs = async (query: string) => (await json(await call("GET", `/api/tasks${query}`))).body.map((t: any) => t.slug);
    expect(await slugs("?status=doing")).toEqual(["other"]);
    expect(await slugs("?q=サポート")).toEqual(["move-runtime"]);
    expect(await slugs("?q=OTHER")).toEqual(["other"]);
    expect(await slugs("?q=0%25_d")).toEqual(["other"]);
    expect(await slugs("?q=0%25x")).toEqual([]);
    expect((await call("GET", "/api/tasks?status=later")).status).toBe(422);
  });

  it("deletes a task and keeps its events", async () => {
    await call("POST", "/api/tasks", task);
    expect((await call("DELETE", "/api/tasks/move-runtime")).status).toBe(204);
    expect((await call("DELETE", "/api/tasks/move-runtime")).status).toBe(404);
    expect((await call("PATCH", "/api/tasks/move-runtime", { title: "x" })).status).toBe(404);
    expect(await events()).toEqual([
      { task_id: 1, action: "create" },
      { task_id: 1, action: "delete" },
    ]);
  });
});
