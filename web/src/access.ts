// access refuses any request that did not come through this Worker's Access application. Access already
// stands in front of the hostname; this check keeps the API closed if a route ever reaches the Worker
// another way (a new hostname, a changed policy, preview URLs).
import { createMiddleware } from "hono/factory";
import { verifyWithJwks } from "hono/jwt";

export const access = createMiddleware<{ Bindings: Env }>(async (c, next) => {
  // Access signs every request it lets through, whether the user came from a browser or from cloudflared.
  const token = c.req.header("cf-access-jwt-assertion");
  if (!token) return c.json({ error: "not signed in through Access" }, 401);
  const team = c.env.CF_ACCESS_TEAM_DOMAIN;
  try {
    await verifyWithJwks(token, {
      jwks_uri: `${team}/cdn-cgi/access/certs`,
      allowedAlgorithms: ["RS256"],
      // aud names this application, so a token Access issued for another one is refused too.
      verification: { iss: team, aud: c.env.CF_ACCESS_AUD },
    });
  } catch {
    return c.json({ error: "not signed in through Access" }, 401);
  }
  await next();
});
