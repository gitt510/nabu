import { env, exports } from "cloudflare:workers";
import { describe, expect, it } from "vitest";
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
      { "cf-access-jwt-assertion": await token({ iss: "https://another-team.cloudflareaccess.com" }) },
      { "cf-access-jwt-assertion": await token({ exp: Math.floor(Date.now() / 1000) - 60 }) },
    ]) {
      const res = await get(headers);
      expect(res.status, JSON.stringify(headers)).toBe(401);
      expect(await res.json()).toEqual({ error: "not signed in through Access" });
    }
  });

  it("guards every route the Worker answers", async () => {
    expect((await exports.default.fetch(new Request("https://nabu.test/api/tasks/any"))).status).toBe(401);
    expect((await exports.default.fetch(new Request("https://nabu.test/anything"))).status).toBe(401);
    expect(env.ACCESS_AUD).not.toBe("");
  });
});
