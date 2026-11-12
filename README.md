# nabu

- nabu is an agent-integrated task manager

## Setup

```bash
go install github.com/gitt510/nabu/cmd/nabu@latest
brew install cloudflared
mkdir -p ~/.config/nabu
echo '{"base_url": "https://nabu.<you>.workers.dev"}' > ~/.config/nabu/config.json
cloudflared access login https://nabu.<you>.workers.dev
```

- nabu needs Go 1.26+ to build
- `nabu -h` and `nabu <command> -h` print the full contract of every command

## Configuration

| key | default | meaning |
| --- | --- | --- |
| `base_url` | (required) | the nabu Worker, which serves the viewer and the API |

- The file is `~/.config/nabu/config.json` (`$XDG_CONFIG_HOME/nabu/config.json`)
- The file holds no secret

## Sign-in

- Cloudflare Access guards the Worker, both the viewer and the API
- nabu sends the Access token that `cloudflared` holds with every request
- nabu fails with `not signed in: run cloudflared access login <base_url>` when `cloudflared` holds no token or an expired one

## Agent skill

```
/plugin marketplace add gitt510/nabu
/plugin install nabu@nabu
```

- `skills/` holds one directory per skill

## Tasks

### Example

```
user:  この PR のレビュー待ち、task にしといて
agent: nabu task new review-pr-68 --area work --pr https://github.com/o/r/pull/68
→ {"action": "task new", "slug": "review-pr-68", "status": "todo", ...}
```

### Overview

- A task lives in Cloudflare D1, and the Worker's `/api/tasks` is the only way in
- A task's status is one of `todo`, `done`, `dropped`, set by `nabu task done`, `nabu task drop`, and `nabu task reopen`
- The body is `why`, why the task is worth starting, and `dod`, its definition of done, each a list of lines
- A change to the body adds columns and leaves old tasks as they were: `situation`, `complication`, `question`, and `answer` stay in D1, unread
- A dropped task also carries `why_dropped`, and no other task does
- A task also carries `memo`, lines of what was done or thought, added one at a time in the viewer
- `nabu task new` and `nabu task replace` take the body as JSON, and `nabu task read` prints it as markdown
- The API refuses a task out of shape, so no stored task is out of shape
- Every write keeps the task before and after it as an event
- [`cmd/nabu/frontmatter.go`](cmd/nabu/frontmatter.go) maps every frontmatter key to its flag, and passes each value to the API unchecked
- [`src/api/schema.ts`](src/api/schema.ts) defines every rule a write must meet
- `nabu task schema` prints those rules as JSON Schema, as the API serves them at `/api/schema`
- [`openapi.json`](openapi.json) describes every route of the API as OpenAPI, as the API serves it at `/api/openapi.json`

## Development

- `just` lists the recipes, such as `just test` and `just check`
- `pnpm test -u` writes `openapi.json` again after a route or a schema changes, and `pnpm test` fails while it is stale
- `go generate ./...` writes the CLI's types in `internal/api/openapi.gen.go` again from `openapi.json`, and `just check` fails while they are stale
- `just dev` serves the viewer and the Worker from the working tree on the local D1, and sends the token `cloudflared` holds with every `/api/*` request, so the Worker checks it as in production
- `just db-pull` copies the production D1 into the local D1
- `src/api/migrations/` holds the D1 schema, applied with `wrangler d1 migrations apply nabu --remote`
- [`docs/config.md`](docs/config.md) lists every config by consumer and sensitivity
- [`docs/architecture.md`](docs/architecture.md) draws the routes that cross containers and how the task contract reaches each consumer
