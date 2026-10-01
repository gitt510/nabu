# nabu

nabu is an agent-integrated task manager

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
echo '{"root": "~/ghq/github.com/you/notes"}' > ~/.config/nabu/config.json
nabu init
```

- `git` on PATH
- Go 1.26+ to build (`mise install` provides it)
- The config file is `$XDG_CONFIG_HOME/nabu/config.json`, falling back to `~/.config/nabu/config.json`
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
- The folder is the status: `inbox`, `doing`, `done`
- The body has two sections: `## For Human` for the user, `## AI memo` for the agent

### Frontmatter

| key | meaning | format |
| --- | --- | --- |
| `scheduled` | when the work is planned to happen | RFC3339 with offset |
| `waiting` | who or what the next action waits on | free text |
| `tickets` | related issues or PRs | full `https` URLs |

- Frontmatter is set only through flags (`--scheduled`, `--waiting`, `--ticket`, `--clear-*`)
- A body that carries its own `---` block is refused

### Validation

- A slug is lowercase kebab-case and unique across every status folder
- The body is `# <title>`, `## For Human`, a `---` line, `## AI memo`, in that order
- Every line in `For Human` section is at most 30 characters

## Canvas

- `CANVAS.md` at the root holds one draft at a time and is never committed
- `canvas open` refuses a non-empty canvas; `canvas read`, `write`, `diff`, and `save` refuse an empty one
- `canvas diff` prints exactly the user's hand edits since the agent last wrote; a canvas started by hand diffs in full
- `canvas save <slug>` writes `writing/<slug>.md` with a `created` timestamp and any `--ticket` URLs as frontmatter, commits it, and empties the canvas
- `canvas save` and `canvas drop` leave an empty `CANVAS.md` in place rather than deleting it

## Behavior

- Every command that changes the root commits that change; there is no way to leave one uncommitted
- A commit message names the command and the path, as `nabu: task mv tasks/inbox/a.md -> tasks/doing/a.md`
- Nothing pushes on its own; `nabu push` pushes the current branch to its upstream and never pulls or rebases
- `nabu push` fails when the branch has no upstream

- Commands that change the root print JSON
- `task read`, `canvas read`, and `canvas diff` print raw text
- `task ls`, `task grep`, and `task validate` print text unless `--json`
- `--content` may be omitted; the body is then read from stdin
- Flags may come before or after the positional arguments

| exit | meaning |
| --- | --- |
| 0 | success (help included) |
| 1 | the operation failed (missing task, existing task, git error) |
| 2 | wrong invocation or no root declared |

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

- User-facing strings are English; Japanese is allowed in comments and tests (enforced by `gosmopolitan`)
