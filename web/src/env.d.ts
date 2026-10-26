// The Access vars reach the Worker with --var on every upload, not from wrangler.jsonc, so wrangler types cannot see them.
interface AccessVars {
  CF_ACCESS_TEAM_DOMAIN: string;
  CF_ACCESS_AUD: string;
}
interface Env extends AccessVars {}
declare namespace Cloudflare {
  interface Env extends AccessVars {}
}
