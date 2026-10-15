---
name: task
description: File and move the user's tasks as markdown files under tasks/ in their nabu repository through the `nabu` CLI. Use when the user wants a task recorded, started, or finished — "add a task", "file this as a task", "I started X", "X is done", "drop X", "what am I working on" — including Japanese phrasings such as 「タスクにして」「タスク登録」「着手した」「完了にして」「やらないことにした」「今やってるタスク」, and when they want the viewer updated now — "publish", 「publish して」「viewer に反映して」. Never edit the repository directly; every read and write goes through nabu.
---

# nabu task

A task is a markdown file under `tasks/` in the user's nabu repository, and its
status is the folder it sits in: `tasks/inbox/` (new), `tasks/doing/`
(started), `tasks/done/` (finished), `tasks/dropped/` (decided against).
`nabu task -h` is the source of truth for the contract,
[`frontmatter.go`](../../frontmatter.go) for every frontmatter key and the
values it accepts, and [`shape.go`](../../shape.go) for the body shape; this
file only says how to use them well.

## Commands

```bash
nabu task new <slug> --area <area> [<frontmatter flags>]  # tasks/inbox/<slug>.md, body on stdin
nabu task replace <slug>                     # new body on stdin; frontmatter is kept
nabu task set <slug> --waiting "<who/what>"  # the ball is with someone else
nabu task set <slug> --clear-waiting         # the ball is back
nabu task set <slug> <frontmatter flags>    # other metadata; --clear-* drops
nabu task mv <slug> <inbox|doing|done|dropped>  # move a task to its status folder
nabu task rename <slug> <new-slug>           # new slug, same status and content
nabu task rm <slug>                          # delete a task, whatever its status
nabu task ls [status] [--json]               # status, title, created; --json adds slug, waiting, and the rest
nabu task read <slug>                        # a task's content, any status
nabu task grep <query> [--json]              # where something is written in tasks
nabu task validate [slug]                    # which tasks are out of shape; nothing written
nabu publish                                 # push, then update the viewer now instead of tonight
```

## How to work

1. **Check what exists first.** `nabu task ls --json` before filing so a
   task is not filed twice. Every task read and write is a `task` command.
2. **File with `task new`.** The slug is the filename; the body (stdin)
   is markdown in the task shape, written as below. Metadata goes through
   the flags `frontmatter.go` defines, never hand-written in the body.
   Do not invent a `--scheduled` time the user did not give. Leave
   `--project` out when the task spans several or the repo does not
   exist yet, and reuse a project already in `task ls --json` rather
   than coining a near-duplicate.

   ```markdown
   # <title>

   ## For Human

   ### Now

   - <one fact about the current state>

   ### Next

   - [ ] <one action, ends in a verb>

   ---

   ## AI memo

   <everything else: config facts, commands, candidates, rejected ideas>
   ```

   `For Human` is what the user reads to know where the task stands and
   what comes next. A fact that does not fit its line limit goes to `AI
   memo`, with a shorter line here. Checkboxes live in `Next` only, never
   in `AI memo`, so a task has one place that says what is done. `AI
   memo` is free markdown for the agent: as dense as the work needs.
3. **Move with `task mv`.** Run `nabu task mv <slug> doing` when work
   starts and `... done` when it ends. Never write a status into the body or
   frontmatter; the folder is the status.
   A task the user decides not to do ("やらない", "見送り", "won't do")
   goes to `dropped`, not `done`. First `task replace` the body into the
   shape `dropped/` requires, with the reason under For Human (the long
   version goes to `AI memo`), then `nabu task mv <slug> dropped`.
   A task that should not have been filed at all (a duplicate, a mistake,
   「消して」「削除して」) is deleted with `nabu task rm <slug>`, whatever its
   status. Delete only when the user asks; a decision not to do the work
   is `dropped`.
4. **Revise with `task replace` and `task set`.** `task
   replace` takes the new body on stdin, in the task shape, and keeps
   the frontmatter; `task set` changes the frontmatter
   and keeps the body. Never write a frontmatter block by hand: a body
   that starts with `---` is refused. Rewrite `Now` as state ("password
   auth is off"), not as a log of what was done.
5. **Mark a wait with `task set --waiting`.** When the user has done their
   part and waits on someone else (a reply, a review), keep the task in
   `doing/` and run `nabu task set <slug> --waiting "<who or what>"`;
   `--clear-waiting` when the ball is back. Do not write the wait into the
   body as well.
6. **Report the path and the frontmatter.** After a write, tell the user the
   relative path nabu printed (`task new`, `replace`, and `set` print `path`;
   `task mv` and `task rename` print `from` and `to`). After `task new` and
   `task set`, also show the `frontmatter` nabu printed, as a YAML block in
   the order nabu writes it, so the user can correct an `area` or `project`
   you chose on the spot.
7. **Repair with `task mv`.** A task that `task ls` shows as `stray`
   (a `tasks/<slug>.md` outside every status folder) moves with
   `nabu task mv <slug> inbox`. A
   frontmatter refused by `task set` or `task replace` was edited by hand;
   tell the user and let them fix the file.

8. **Publish on request.** The viewer updates once a night. When the user
   asks to publish or to see a change in the viewer now, run `nabu
   publish` and tell them the `run` URL it prints: the viewer updates when
   that run succeeds. When it fails, show the error as is; it says whether
   the push or the run failed.

## Do not

- Read or edit files in the notebook with Read / Edit / Write / shell
  redirection. Only nabu touches the notebook.
- Run git inside the notebook. nabu commits each change itself; `nabu
  push` only when the user asks. When a result carries `"push":
  "failed: ..."`, tell the user; the change is committed but not pushed.
- Guess the notebook. If nabu exits 2 with "no notebook declared", show the
  user the message; it tells them what to fix.
