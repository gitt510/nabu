# nabu

- nabu is an agent-integrated task manager

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
echo '{"notebook": "~/ghq/github.com/you/notes"}' > ~/.config/nabu/config.json
nabu init
```

- Building nabu needs Go 1.26+, which `mise install` provides
- `nabu -h` and `nabu <command> -h` print the full contract of every command

## Configuration

| key | default | meaning |
| --- | --- | --- |
| `notebook` | (required) | the git repository nabu reads and writes |
| `auto_push` | `false` | `git push` after each commit nabu makes |

- The file is `~/.config/nabu/config.json` (`$XDG_CONFIG_HOME/nabu/config.json`)

### notebook

- Every nabu command that writes makes one git commit of the files it touched
- `CANVAS.md` is the exception, ignored by git and never committed
- nabu never commits a change it did not make

### auto_push

- `auto_push` pushes the current branch to its upstream after each commit
- The notebook needs an upstream branch and git credentials before the first push
- A failed push keeps the commit and returns `"push": "failed: <reason>"` in the JSON result

## Agent skill

```
/plugin marketplace add gitt510/nabu
/plugin install nabu@nabu
```

| skill | covers |
| --- | --- |
| `nabu:task` | filing, moving, and revising tasks |
| `nabu:canvas` | the co-writing loop on `CANVAS.md` |

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
- `done` holds finished work
- `dropped` holds work decided against, with the reason in its `For Human`
- `task rm` deletes a task that should not have been filed (a duplicate, a mistake), whatever its status
- Git history keeps the content of a deleted task
- The body has two sections: `## For Human` for the user, `## AI memo` for the agent

### Frontmatter

| key | meaning | format |
| --- | --- | --- |
| `created` | filing time, stamped by nabu and never changed | RFC3339 with offset |
| `area` | side of life, required | `work` or `personal` |
| `project` | destination of the work: a service at work, a repo at home, or none | lowercase kebab-case |
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
- Every task carries an `area`
- `task new` refuses a task without `--area`, and `task validate` reports one
- A task in `dropped/` carries `### Why dropped` under `For Human` with at least one line
- `task mv ... dropped` refuses a task without it

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
- You edit `CANVAS.md` in your editor
- The agent revises `CANVAS.md` through the CLI
- The agent reads every hand edit before revising and never reverts one
- The agent saves the draft when you call it final
  - The saved draft hangs in `gallery/<slug>.md` and the canvas is emptied

## Development

- `just` lists the recipes, such as `just test` and `just check`
