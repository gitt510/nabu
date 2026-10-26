# Config

| Consumer | Secret | Non-secret |
|---|---|---|
| Worker | — | `CF_ACCESS_TEAM_DOMAIN`, `CF_ACCESS_AUD`, D1 `database_id` |
| `nabu` CLI | Access token | `url` |
| `just dev` | Access token | `url` |
| wrangler | OAuth token | — |
| op | app authorization | — |

- `CF_ACCESS_TEAM_DOMAIN` and `CF_ACCESS_AUD` live in 1Password as text fields
- `.env.refs` maps each name to an `op://` reference and stays on the machine
- `.env.refs.example` shows the names in `.env.refs`
- `CF_ACCESS_TEAM_DOMAIN` and `CF_ACCESS_AUD` reach the Worker with `--var` on every upload by `just deploy` and `just preview`
- The tests pass stand-ins for `CF_ACCESS_TEAM_DOMAIN` and `CF_ACCESS_AUD` in `web/vitest.config.ts`
- D1 `database_id` lives in `web/wrangler.jsonc`, and the D1 binding needs no credential
- `url` lives in `~/.config/nabu/config.json` and names the Worker the CLI and `just dev` call
- The Access token is issued by `cloudflared access login <url>` and kept by cloudflared
- The wrangler OAuth token is issued by `wrangler login` and kept by wrangler
- The op app authorization is granted by the 1Password app after Touch ID and kept by the app
