# nabu

- nabu is an agent-integrated task manager

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
echo '{"notebook": "~/ghq/github.com/you/notes"}' > ~/.config/nabu/config.json
nabu init
```

- nabu needs Go 1.26+ to build
- `nabu -h` and `nabu <command> -h` print the full contract of every command

## Configuration

| key | default | meaning |
| --- | --- | --- |
| `notebook` | (required) | the git repository nabu reads and writes |
| `auto_push` | `false` | `git push` after each commit nabu makes |

- The file is `~/.config/nabu/config.json` (`$XDG_CONFIG_HOME/nabu/config.json`)

### notebook

- Every nabu command that writes makes one git commit of the files it touched
- `CANVAS.md` is the exception and is never committed
- nabu never commits a change it did not make

### auto_push

- `auto_push` pushes the current branch to its upstream after each commit
- The notebook needs an upstream branch and git credentials before the first push
- A failed push keeps the commit and returns `"push": "failed: <reason>"` in the JSON result

### publish

```yaml
# .github/workflows/publish.yml in the notebook repository
on:
  schedule:
    - cron: "17 18 * * *" # 03:17 JST
  workflow_dispatch:
jobs:
  publish:
    uses: gitt510/nabu/.github/workflows/publish.yml@main
    with:
      r2-bucket: nabu
    secrets:
      R2_EDIT: ${{ secrets.R2_EDIT }}
```

- The publish workflow replaces `<r2-bucket>/tasks.json` with the tasks of the branch the run starts on: the default branch on schedule
- The schedule runs it once a night; a push alone publishes nothing
- `nabu publish` pushes the notebook and starts the workflow on the current branch at once, through `gh`
- `nabu publish` prints the URL of the run it started, and reports a push that landed without a run
- `nabu publish` needs `gh` signed in, a single remote `origin`, and the caller workflow on the branch it starts
- `R2_EDIT` is a Cloudflare API token with R2 edit permission in the account that holds `r2-bucket`
- A newer run cancels an older one still running

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
agent: nabu task new review-pr-68 --area work --ticket https://github.com/o/r/pull/68
→ tasks/inbox/review-pr-68.md
```

### Overview

- A task is one file, `tasks/<status>/<slug>.md`
- The folder is the status: `inbox`, `doing`, `done`, `dropped`
- The body has two sections: `## For Human` for the user, `## AI memo` for the agent
- [`frontmatter.go`](frontmatter.go) defines every frontmatter key, its flag, and the values it accepts
- [`shape.go`](shape.go) defines the body shape that `task validate` checks

## Canvas

### Example

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

- `CANVAS.md` at the top of the notebook holds one draft at a time
- `canvas save <slug>` moves the draft to `gallery/<slug>.md` and commits it
- The `nabu:canvas` skill reads the hand edits with `canvas diff` before every `canvas write`

## Development

- `just` lists the recipes, such as `just test` and `just check`
