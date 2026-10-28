import { env } from "cloudflare:workers";
import { beforeEach, describe, expect, it } from "vitest";
import { AREAS, STATUSES } from "../schema";

// The D1 CHECK constraints are a copy of rules schema.ts owns, kept as the last guard against a write that
// skips the API. These tests fail when the two drift apart, so a change to STATUSES or AREAS comes with a
// migration that rebuilds the table.

const tableSQL = async () =>
  (await env.DB.prepare("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'tasks'").first<{ sql: string }>())!.sql;

// allowed reads the values the CHECK on column lets in, from the table as the migrations left it.
const allowed = (sql: string, column: string) => {
  const list = sql.match(new RegExp(`CHECK \\(${column} IN \\(([^)]*)\\)\\)`))?.[1];
  return list?.split(",").map((v) => v.trim().replace(/^'|'$/g, ""));
};

const insert = (fields: Record<string, string>) => {
  const row = { slug: "a", title: "A", status: "inbox", area: "work", created: "now", updated: "now", ...fields };
  return env.DB.prepare(`INSERT INTO tasks (${Object.keys(row).join(", ")}) VALUES (${Object.keys(row).map(() => "?").join(", ")})`)
    .bind(...Object.values(row))
    .run();
};

beforeEach(async () => {
  await env.DB.prepare("DELETE FROM tasks").run();
});

describe("the tasks table", () => {
  it("lets in exactly the statuses schema.ts gives", async () => {
    expect(new Set(allowed(await tableSQL(), "status"))).toEqual(new Set(STATUSES));
  });

  it("lets in exactly the areas schema.ts gives", async () => {
    expect(new Set(allowed(await tableSQL(), "area"))).toEqual(new Set(AREAS));
  });

  it("enforces its CHECK constraints on a write that skips the API", async () => {
    await expect(insert({ status: "wip" })).rejects.toThrow(/CHECK constraint failed/);
    await expect(insert({ area: "hobby" })).rejects.toThrow(/CHECK constraint failed/);
    await expect(insert({ status: "dropped" })).rejects.toThrow(/CHECK constraint failed/);
    await expect(insert({ status: "dropped", why_dropped: '["no use for it now"]' })).resolves.toBeDefined();
  });
});
