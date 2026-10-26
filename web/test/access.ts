// A stand-in for Access: a key pair whose public half the Worker fetches as the team's certs, and tokens
// signed with it the way Access signs a request it lets through.
import { env } from "cloudflare:workers";
import { sign } from "hono/jwt";
import { vi } from "vitest";

const team = `https://${env.CF_ACCESS_TEAM_DOMAIN}`;

const pair = async (kid: string) => {
  const keys = (await crypto.subtle.generateKey(
    { name: "RSASSA-PKCS1-v1_5", modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: "SHA-256" },
    true,
    ["sign", "verify"],
  )) as CryptoKeyPair;
  const jwk = async (k: CryptoKey) => ({ ...((await crypto.subtle.exportKey("jwk", k)) as JsonWebKey), kid, alg: "RS256" });
  return { private: await jwk(keys.privateKey), public: await jwk(keys.publicKey) };
};

const access = await pair("access");
// stranger is a key Access never published, as a forged token would use.
const stranger = await pair("access");

const realFetch = globalThis.fetch;
vi.spyOn(globalThis, "fetch").mockImplementation((input, init) =>
  String(input instanceof Request ? input.url : input) === `${team}/cdn-cgi/access/certs`
    ? Promise.resolve(Response.json({ keys: [access.public] }))
    : realFetch(input, init),
);

// token is what Access would put in cf-access-jwt-assertion; claims override the valid defaults.
export const token = (claims: Record<string, unknown> = {}, key: "access" | "stranger" = "access") => {
  const now = Math.floor(Date.now() / 1000);
  return sign(
    { iss: team, aud: [env.CF_ACCESS_AUD], email: "me@example.com", iat: now, nbf: now, exp: now + 600, ...claims },
    (key === "access" ? access : stranger).private as never,
  );
};
