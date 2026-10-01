---
name: task
description: File and move the user's tasks as markdown files under tasks/ in their nabu repository through the `nabu` CLI. Use when the user wants a task recorded, started, or finished — "add a task", "file this as a task", "I started X", "X is done", "what am I working on" — including Japanese phrasings such as 「タスクにして」「タスク登録」「着手した」「完了にして」「今やってるタスク」. Never edit the repository directly; every read and write goes through nabu.
---

# nabu task

A task is a markdown file under `tasks/` in the user's nabu repository, and its
status is the folder it sits in: `tasks/inbox/` (new), `tasks/doing/`
(started), `tasks/done/` (finished). `nabu task -h` is the source of truth
for the contract; this file only says how to use it well.

## Commands

```bash
nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]...  # tasks/inbox/<slug>.md, body on stdin
nabu task replace <slug>                     # new body on stdin; frontmatter is kept
nabu task set <slug> --waiting "<who/what>"  # the ball is with someone else
nabu task set <slug> --clear-waiting         # the ball is back
nabu task set <slug> --scheduled <RFC3339> --ticket <https-url>   # other metadata; --clear-* drops
nabu task mv <slug> <inbox|doing|done>       # move a task to its status folder
nabu task rename <slug> <new-slug>           # new slug, same status and content
nabu task ls [status] [--json]               # what exists; waiting shows who holds the ball
nabu task read <slug>                        # a task's content, any status
nabu task grep <query> [--json]              # where something is written in tasks
nabu task view                               # one HTML page of every task, opened in the browser
nabu task validate [slug]                    # which tasks are out of shape; nothing written
```

## How to work

1. **Check what exists first.** `nabu task ls --json` before filing so a
   task is not filed twice. Every task read and write is a `task` command.
2. **File with `task new`.** The slug is the filename (lowercase
   kebab-case); the body (stdin) is markdown in the task shape below, and
   nabu refuses any other. Metadata goes through flags, never hand-written
   in the body: `--scheduled` is the time the work is planned to happen
   (RFC3339 with offset, not a deadline), `--ticket` is a full `https`
   issue URL, repeatable.

   ```markdown
   # <title>

   ## For Human

   ### Now

   - <one fact about the current state, at most 30 characters>

   ### Next

   - [ ] <one action, at most 30 characters, ends in a verb>

   ---

   ## AI memo

   <everything else: config facts, commands, candidates, rejected ideas>
   ```

   `For Human` is what the user reads to know where the task stands and
   what comes next; every line in it is at most 30 characters (counted
   in runes, prefix included). A fact that does not fit goes to `AI
   memo`, with a shorter line here. Checkboxes live in `Next` only, never
   in `AI memo`, so a task has one place that says what is done. `AI
   memo` is free markdown for the agent: as dense as the work needs.
3. **Move with `task mv`.** Run `nabu task mv <slug> doing` when work
   starts and `... done` when it ends. Never write a status into the body or
   frontmatter; the folder is the status.
4. **Revise with `task replace` and `task set`.** `task
   replace` takes the new body on stdin, in the same shape `task new`
   requires, and keeps the frontmatter; `task set` changes the frontmatter
   and keeps the body. Never write a frontmatter block by hand: a body
   that starts with `---` is refused. Rewrite `Now` as state ("password
   auth is off"), not as a log of what was done.
5. **Mark a wait with `task set --waiting`.** When the user has done their
   part and waits on someone else (a reply, a review), keep the task in
   `doing/` and run `nabu task set <slug> --waiting "<who or what>"`;
   `--clear-waiting` when the ball is back. Do not write the wait into the
   body as well.
6. **Report the path.** After a write, tell the user the relative path nabu
   printed (`task new`, `replace`, and `set` print `path`; `task mv` and `task
   rename` print `from` and `to`).
7. **Repair with `task mv`.** A task that `task ls` shows as `stray`
   (a `tasks/<slug>.md` outside every status folder) moves with
   `nabu task mv <slug> inbox`. A
   frontmatter refused by `task set` or `task replace` was edited by hand;
   tell the user and let them fix the file.

## Do not

- Read or edit files under the root with Read / Edit / Write / shell
  redirection. Only nabu touches the root.
- Run git inside the root. nabu commits each change itself; `nabu
  push` only when the user asks.
- Guess the root. If nabu exits 2 with "no root declared", show the user
  the message; it tells them what to fix.
