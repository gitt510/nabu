# nabu

nabu keeps one markdown repository for tasks and drafts. Tasks live under
`tasks/` with their status as the folder, every change is a git commit, and
`CANVAS.md` is the one draft the user edits by hand while an agent revises it
through nabu.

## Requirements

- `git` on PATH
- Go 1.26+ to build (`mise install` provides it)

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
echo '{"root": "~/ghq/github.com/you/notes"}' > ~/.config/nabu/config.json
nabu init
```

- The config file is `$XDG_CONFIG_HOME/nabu/config.json`, falling back to `~/.config/nabu/config.json`
- `root` is the only key; `~` expands to the home directory
- `--root <dir>` overrides the config file for one invocation
- `nabu init` creates the root as a git repository on `main` with a first commit; rerunning it changes nothing
- A root initialized before the canvas existed needs `CANVAS.md` and `.CANVAS.agent.md` added to its `.gitignore` by hand
- `nabu -h` and `nabu <command> -h` print the full contract of every command

## Behavior

### Tasks

- A task is `tasks/<status>/<slug>.md`; the folder is its only status: `inbox`, `doing`, `done`
- A slug is lowercase kebab-case and is unique across every status folder
- A task body is `# <title>`, a `## For Human` section, a `---` line, then a `## AI memo` section
- `For Human` holds only blank lines, `### ` headings, and `- ` bullets of at most 30 characters after the `- `, `- [ ] `, or `- [x] ` prefix
- `AI memo` is free markdown
- A body in any other shape is refused and nothing is written
- Frontmatter comes only from flags: `scheduled` (RFC3339 with offset, the planned work time), `waiting` (who or what the next action waits on), `tickets` (full `https` URLs)
- A body that starts with `---` is refused
- `task set` changes frontmatter and keeps the body; `task replace` changes the body and keeps the frontmatter
- A task whose frontmatter was edited by hand is refused by `task set` and `task replace` until fixed by hand
- `task ls` shows a `tasks/<slug>.md` outside every status folder as `stray`; `task mv <slug> inbox` puts it away
- `task validate` writes nothing and exits 1 when any task is out of shape

### Canvas

- `CANVAS.md` at the root holds one draft at a time and is never committed
- `canvas open` refuses a non-empty canvas; `canvas read`, `write`, `diff`, and `save` refuse an empty one
- `canvas diff` prints exactly the user's hand edits since the agent last wrote; a canvas started by hand diffs in full
- `canvas save <slug>` writes `writing/<slug>.md` with a `created` timestamp and any `--ticket` URLs as frontmatter, commits it, and empties the canvas
- `canvas save` and `canvas drop` leave an empty `CANVAS.md` in place rather than deleting it

### Git

- Every command that changes the root commits that change; there is no way to leave one uncommitted
- A commit message names the command and the path, as `nabu: task mv tasks/inbox/a.md -> tasks/doing/a.md`
- Nothing pushes on its own; `nabu push` pushes the current branch to its upstream and never pulls or rebases
- `nabu push` fails when the branch has no upstream

### Output

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
