# nabu

nabu is an agent-integrated task manager.

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
echo '{"notebook": "~/ghq/github.com/you/notes"}' > ~/.config/nabu/config.json
nabu init
```

- Go 1.26+ to build (`mise install` provides it)
- `nabu -h` and `nabu <command> -h` print the full contract of every command

## Tasks

### Use case

```
user:  この PR のレビュー待ち、task にしといて
agent: nabu task new review-pr-68 --ticket https://github.com/o/r/pull/68
→ tasks/inbox/review-pr-68.md
```

### Overview

- A task is one file, `tasks/<status>/<slug>.md`
- The folder is the status: `inbox`, `doing`, `done`, `dropped`
- `done` is finished work; `dropped` is work decided against, and its `For Human` says why
- `task rm` deletes a task that should not have been filed (a duplicate, a mistake), whatever its status; git history keeps the content
- The body has two sections: `## For Human` for the user, `## AI memo` for the agent
- Every change is a git commit; nothing is pushed until you ask for `nabu push`

### Frontmatter

| key | meaning | format |
| --- | --- | --- |
| `created` | when `task new` filed the task; stamped by nabu, never changed | RFC3339 with offset |
| `area` | which side of life the task belongs to; required | `work` or `personal` |
| `project` | what the task ships into: a service at work, a repo at home; absent when none | lowercase kebab-case |
| `scheduled` | when the work is planned to happen | RFC3339 with offset, or `YYYY-MM-DD` for the whole day |
| `waiting` | who or what the next action waits on | free text |
| `tickets` | work items: an issue, a Wrike task, a Zendesk request | full `https` URLs |
| `prs` | pull requests | full `https` URLs |
| `links` | everything else: a repo, an article, a post | full `https` URLs |

- A URL goes by what it points at, whoever opened it and whatever it is to this task
- Frontmatter is set only through flags (`--area`, `--project`, `--scheduled`, `--waiting`, `--ticket`, `--pr`, `--link`, `--clear-*`)
- A body that carries its own `---` block is refused

### Validation

- A slug is lowercase kebab-case and unique across every status folder
- The body is `# <title>`, `## For Human`, a `---` line, `## AI memo`, in that order
- Every line in `For Human` section is at most 30 characters
- Every task carries an `area`; `task new` is refused without `--area`, and `task validate` reports a task without one
- A task in `dropped/` carries `### Why dropped` under `For Human` with at least one line; `task mv ... dropped` is refused without it

## Canvas

### Use case

```
user:  PR #68 のコメント、一緒に書きたい
agent: nabu canvas open
→ CANVAS.md を editor で開いてください

(user edits CANVAS.md by hand)

user:  直した、添削して
agent: nabu canvas diff        # reads your edits first
       nabu canvas write

user:  いいね。保存しておいて
agent: nabu canvas save pr68-comment --ticket https://github.com/o/r/pull/68
→ gallery/pr68-comment.md
```

### Overview

- `CANVAS.md` at the top of the notebook holds one draft at a time and is never committed
- You edit `CANVAS.md` in your editor; the agent revises it through the CLI
- Every hand edit is yours: the agent reads it before revising and never reverts it
- When the draft is final, ask the agent to save it
  - it hangs in `gallery/<slug>.md` and the canvas is emptied

## Viewer

- `just view` renders `tasks/` and the viewer into `web/dist/` and serves them on `localhost:8000`
- `just publish` uploads the tasks as `tasks.json` to the R2 bucket `nabu`; the `nabu` Worker serves it behind Cloudflare Access
- `just deploy` builds the viewer and deploys it as the Worker's static assets (`web/wrangler.jsonc`)
- A push to `main` runs `go test` and deploys the same way (`.github/workflows/deploy.yml`)
- A publish started while another is running exits with `skip`

## Push on every commit

```json
{"notebook": "~/ghq/github.com/you/notes", "auto_push": true}
```

- `auto_push` runs `git push` right after each commit nabu makes
- The remote and its credentials are git's: run `git push -u origin main` in the notebook once, with SSH or a credential helper set up
- The JSON result of each write carries `"push": "ok"`, or `"push": "failed: <reason>"` with the commit kept
- `--notebook` bypasses the config file, so nothing is pushed

## Agent skill

```
/plugin marketplace add gitt510/nabu
/plugin install nabu@nabu
```

| skill | covers |
| --- | --- |
| `nabu:task` | filing, moving, and revising tasks |
| `nabu:canvas` | the co-writing loop on `CANVAS.md` |

## Development

```bash
just test     # go test ./...
just check    # gofmt, go vet, golangci-lint
just run ...  # go run . <args>
```
