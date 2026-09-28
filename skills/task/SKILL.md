---
name: task
description: File and move the user's tasks as markdown notes under tasks/ in their notes repository through the `nabu` CLI. Use when the user wants a task recorded, started, or finished in their notes — "add a task", "file this as a task", "I started X", "X is done", "what am I working on" — including Japanese phrasings such as 「タスクにして」「タスク登録」「着手した」「完了にして」「今やってるタスク」. Never edit the notes repository directly; every read and write goes through nabu.
---

# nabu task

A task is a note under `tasks/` in the user's notes repository, and its
status is the folder it sits in: `tasks/inbox/` (new), `tasks/doing/`
(started), `tasks/done/` (finished). `nabu task -h` is the source of truth
for the contract; this file only says how to use it well.

## Commands

```bash
nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]...  # tasks/inbox/<slug>.md, body on stdin
nabu task mv <slug> <inbox|doing|done>       # move a task to its status folder
nabu note ls tasks/doing --json              # what is in flight
nabu note read tasks/doing/<slug>.md         # a task's content
nabu note replace tasks/<st>/<slug>.md       # revise a task's body (whole file, stdin)
nabu doctor --notes                          # warn on tasks outside a status folder
```

## How to work

1. **Read the conventions first.** `nabu note read README.md` gives the
   rules of this particular root; they win over any default here. Check
   `nabu note ls tasks --json` before filing so a task is not filed twice.
2. **File with `task new`.** The slug is the filename (lowercase
   kebab-case); the body (stdin) is ordinary markdown starting with a `# `
   title. Metadata goes through flags, never hand-written in the body:
   `--scheduled` is the time the work is planned to happen (RFC3339 with
   offset, not a deadline), `--ticket` is a full `https` issue URL, repeatable.
3. **Move with `task mv`.** Run `nabu task mv <slug> doing` when work
   starts and `... done` when it ends. Never write a status into the body or
   frontmatter; the folder is the status.
4. **Revise with `note replace`.** It rewrites the whole file: read it
   first and keep the frontmatter block in the new body.
5. **Report the path.** After a write, tell the user the relative path nabu
   printed (`task new` prints `path`; `task mv` prints `from` and `to`).
6. **Repair with `note mv`.** A task flagged by `doctor --notes` as outside
   a status folder moves with
   `nabu note mv tasks/<slug>.md tasks/inbox/<slug>.md`.

Ordinary notes are the `nabu:note` skill.

## Do not

- Read or edit files under the notes root with Read / Edit / Write / shell
  redirection. Only nabu touches the root.
- Run git inside the notes root. nabu commits each change itself; `nabu
  push` only when the user asks.
- Guess the root. If nabu exits 2 with "no root declared", run `nabu doctor`
  and show the user its output; it tells them what to fix.
