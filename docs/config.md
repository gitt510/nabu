# Config

| Consumer | Secret | Non-secret |
|---|---|---|
| Worker | — | `CF_ACCESS_TEAM_DOMAIN`, `CF_ACCESS_AUD`, D1 `database_id` |
| `nabu` CLI | cloudflared access token | `url` |
| `just dev` | cloudflared access token | `url` |
| GitHub Actions deploy | `CLOUDFLARE_API_TOKEN` | — |
| wrangler | OAuth token | — |
| op | app authorization | — |
