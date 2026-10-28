import { exports } from "cloudflare:workers";
import { expect, it } from "vitest";
import { token } from "./access";

// openapi.json at the repository root is the document /api/openapi.json serves; the CLI generates its types
// from it. pnpm test -u writes it again after a route or a schema changes.
it("serves the OpenAPI document committed as openapi.json", async () => {
  const res = await exports.default.fetch(
    new Request("https://nabu.test/api/openapi.json", { headers: { "cf-access-jwt-assertion": await token() } }),
  );
  expect(res.status).toBe(200);
  await expect(`${JSON.stringify(await res.json(), null, 2)}\n`).toMatchFileSnapshot("../../../openapi.json");
});
