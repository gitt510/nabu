# Config

| Consumer | Secret | Non-secret |
|---|---|---|
| Worker | — | `CF_ACCESS_TEAM_DOMAIN`, `CF_ACCESS_AUD`, D1 `database_id`, `DEV_SKIP_ACCESS` (just dev only) |
| `nabu` CLI | cloudflared access token | `base_url` |
| `just dev` | — | `DEV_SKIP_ACCESS` |
| GitHub Actions deploy | `CLOUDFLARE_API_TOKEN` | — |
| wrangler | OAuth token | — |
| op | app authorization | — |
