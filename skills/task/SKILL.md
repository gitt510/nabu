---
name: task
description: File, finish, and drop the user's tasks through the `nabu` CLI, which keeps them behind the nabu API. Use when the user wants a task recorded, done, or dropped — "add a task", "file this as a task", "X is done", "drop X", "reopen X", "what is left to do" — including Japanese phrasings such as 「タスクにして」「タスク登録」「完了にして」「やらないことにした」「戻して」「残りのタスク」. Every read and write goes through nabu.
---

# nabu task

A task lives behind the nabu API, and nabu is the only way to it. Its
status is one of `todo`, `done` (the work is finished), and `dropped` (decided
against, with `why_dropped` saying why). `--scheduled` says when the work
happens; there is no "started" status. `nabu task -h` is the source of truth for the
commands and their flags, and `nabu task schema` for every rule a write
must meet: the keys, the values they accept, and the line limit. This file
only says how to use them well.

## Commands

```bash
nabu task new <slug> --area <area> [<frontmatter flags>]  # file a todo task, JSON body on stdin
nabu task replace <slug>                     # the fields to change, JSON on stdin; frontmatter is kept
nabu task set <slug> --waiting "<who/what>"  # the ball is with someone else
nabu task set <slug> --clear-waiting         # the ball is back
nabu task set <slug> <frontmatter flags>    # other metadata; --clear-* drops
nabu task done <slug>                        # the work is finished
nabu task drop <slug> "<why>"...             # decided against; one line of why_dropped per argument
nabu task reopen <slug>                      # back to todo; a dropped task loses its why_dropped
nabu task rename <slug> <new-slug>           # new slug, same status and content
nabu task rm <slug>                          # delete a task, whatever its status
nabu task ls [status] [--json]               # status, title, created; --json adds slug, the frontmatter, and the body
nabu task read <slug>                        # a task's content as markdown, any status
nabu task schema                             # the rules a write must meet, as JSON Schema
nabu task grep <query> [--json]              # where something is written in tasks
```

Every command after `new` also takes a task's id in place of its slug: the
number the viewer shows after a title as `#6`. Pass it as `6`,
or quote `'#6'`, since a shell reads an unquoted `#` as a comment.

## How to work

1. **Check what exists first.** `nabu task ls --json` before filing so a
   task is not filed twice, and `nabu task schema` before writing so the
   values and lines meet the rules.
2. **File with `task new`.** The slug names the task; the body (stdin)
   is a JSON object of its title alone, `{"title": "<title>"}`. Metadata
   goes through the flags `nabu task -h` lists, never hand-written in the body.
   Do not invent a `--scheduled` time the user did not give. Leave
   `--project` out when the task spans several or the repo does not
   exist yet, and reuse a project already in `task ls --json` rather
   than coining a near-duplicate. Put a ticket, PR, or page the user
   mentions in `--ticket`, `--pr`, or `--link`.
3. **Finish with `task done`, or give up with `task drop`.** Run `nabu task done <slug>`
   when the work is finished. Never write a status into the body or the
   frontmatter.
   A task the user decides not to do ("やらない", "見送り", "won't do")
   is dropped, not done: `nabu task drop <slug> "<reason>"`, with the
   reason in the user's words. Ask for it when they gave none; never
   invent one. `nabu task reopen <slug>` makes a done or dropped task todo again.
   A task that should not have been filed at all (a duplicate, a mistake,
   「消して」「削除して」) is deleted with `nabu task rm <slug>`, whatever
   its status. Delete only when the user asks; a decision not to do the work
   is a drop.
4. **Revise with `task replace` and `task set`.** `task replace` takes a
   new title on stdin and keeps the frontmatter; `task set` changes the
   frontmatter and keeps the title.
5. **Mark a wait with `task set --waiting`.** When the user has done their
   part and waits on someone else (a reply, a review), keep the task todo
   and run `nabu task set <slug> --waiting "<who or what>"`;
   `--clear-waiting` when the ball is back. Do not write the wait into the
   body as well.
6. **Report the slug and the frontmatter.** After a write, tell the user
   the slug and status nabu printed (`task rename` also prints `from` and
   `to`). After `task new` and `task set`, also show the
   `frontmatter` nabu printed, as a YAML block in the order nabu prints
   it, so the user can correct an `area` or `project` you chose on the
   spot.
7. **Fix what the API refuses.** nabu exits 2 with the API's reason when a
   body or a value breaks the rules (a reason line too long, a dropped task
   without a reason); rewrite and run the command again.

## Do not

- Reach the API, D1, or the viewer any other way than nabu.
- Guess the setup. If nabu exits 2 with "no base_url declared" or "not signed
  in", show the user the message as is; it says what to run. Suggest they
  type `! cloudflared access login <base_url>` to sign in from the prompt.
