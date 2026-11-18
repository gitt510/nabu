import { env, exports } from "cloudflare:workers";
import { describe, expect, it } from "vitest";
import app from "../index";
import { token } from "./access";

const get = (headers: Record<string, string> = {}) =>
  exports.default.fetch(new Request("https://nabu.test/api/tasks", { headers }));

describe("access", () => {
  it("lets a token Access signed for this application through", async () => {
    expect((await get({ "cf-access-jwt-assertion": await token() })).status).toBe(200);
  });

  it("refuses a request that did not come through Access", async () => {
    for (const headers of <Record<string, string>[]>[
      {},
      { "cf-access-jwt-assertion": "not-a-jwt" },
      { "cf-access-jwt-assertion": await token({}, "stranger") },
      { "cf-access-jwt-assertion": await token({ aud: ["another-application"] }) },
      {
        "cf-access-jwt-assertion": await token({
          iss: "https://another-team.cloudflareaccess.com",
        }),
      },
      { "cf-access-jwt-assertion": await token({ exp: Math.floor(Date.now() / 1000) - 60 }) },
    ]) {
      const res = await get(headers);
      expect(res.status, JSON.stringify(headers)).toBe(401);
      expect(await res.json()).toEqual({ error: "not signed in through Access" });
    }
  });

  it("lets just dev through on localhost alone, and nothing through without its flag", async () => {
    const dev = { ...env, DEV_SKIP_ACCESS: "1" };
    const at = (host: string) => new Request(`http://${host}/api/tasks`);
    expect((await app.fetch(at("localhost:8788"), dev)).status).toBe(200);
    expect((await app.fetch(at("nabu.test"), dev)).status).toBe(401);
    expect((await app.fetch(at("localhost:8788"), env)).status).toBe(401);
  });

  it("guards every route the Worker answers", async () => {
    expect(
      (await exports.default.fetch(new Request("https://nabu.test/api/tasks/any"))).status,
    ).toBe(401);
    expect((await exports.default.fetch(new Request("https://nabu.test/anything"))).status).toBe(
      401,
    );
    expect(env.CF_ACCESS_AUD).not.toBe("");
  });
});
