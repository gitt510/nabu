---
name: task
description: File, finish, and cancel the user's tasks through the `nabu` CLI, which keeps them behind the nabu API. Use when the user wants a task recorded, done, or canceled — "add a task", "file this as a task", "X is done", "cancel X", "drop X", "reopen X", "what is left to do" — including Japanese phrasings such as 「タスクにして」「タスク登録」「完了にして」「やらないことにした」「戻して」「残りのタスク」. Every read and write goes through nabu.
---

# nabu task

A task lives behind the nabu API, and nabu is the only way to it. Its
status is one of `todo`, `done` (the work is finished), and `canceled` (decided
against, with `why_canceled` saying why). `--scheduled` says when the work
happens; there is no "started" status. `nabu task -h` is the source of truth for the
commands and their flags, and `nabu task schema` for every rule a write
must meet: the keys, the values they accept, and the line limit. This file
only says how to use them well.

## Commands

```bash
nabu task new --area <area> [<frontmatter flags>]  # file a todo task, JSON body on stdin; prints its id
nabu task replace <id>                     # the fields to change, JSON on stdin; frontmatter is kept
nabu task set <id> --waiting "<who/what>"  # the ball is with someone else
nabu task set <id> --clear-waiting         # the ball is back
nabu task set <id> <frontmatter flags>     # other metadata; --clear-* drops
nabu task done <id>                        # the work is finished
nabu task cancel <id> "<why>"...           # decided against; one line of why_canceled per argument
nabu task reopen <id>                      # back to todo; a canceled task loses its why_canceled
nabu task rm <id>                          # delete a task, whatever its status
nabu task item add <id> "<line>"...        # add lines last in a task's checklist, one per argument
nabu task item ls <id>                     # a task's checklist: number, [x] or [ ], text
nabu task item check <id> <n>...           # mark lines done, each by its number in item ls
nabu task item uncheck <id> <n>...         # mark lines not done again
nabu task ls [status] [--json]             # status, title, created; --json adds id, the frontmatter, and the body
nabu task read <id>                        # a task's content as markdown, any status
nabu task schema                             # the rules a write must meet, as JSON Schema
nabu task grep <query> [--json]              # where something is written in tasks
```

A task is named by its id: the number `task new` prints and the viewer
shows after a title as `#6`. Pass it as `6`, or quote `'#6'`, since a
shell reads an unquoted `#` as a comment. Find a task's id with
`nabu task ls --json` when the user names it by its title.

## How to work

1. **Check what exists first.** `nabu task ls --json` before filing so a
   task is not filed twice, and `nabu task schema` before writing so the
   values and lines meet the rules.
2. **File with `task new`.** The body (stdin)
   is a JSON object of its title alone, `{"title": "<title>"}`. Metadata
   goes through the flags `nabu task -h` lists, never hand-written in the body.
   Do not invent a `--scheduled` time the user did not give. Leave
   `--project` out when the task spans several or the repo does not
   exist yet, and reuse a project already in `task ls --json` rather
   than coining a near-duplicate. Put a ticket, PR, or page the user
   mentions in `--ticket`, `--pr`, or `--link`.
3. **Finish with `task done`, or give up with `task cancel`.** Run `nabu task done <id>`
   when the work is finished. Never write a status into the body or the
   frontmatter.
   A task the user decides not to do ("やらない", "見送り", "won't do")
   is canceled, not done: `nabu task cancel <id> "<reason>"`, with the
   reason in the user's words. Ask for it when they gave none; never
   invent one. `nabu task reopen <id>` makes a done or canceled task todo again.
   A task that should not have been filed at all (a duplicate, a mistake,
   「消して」「削除して」) is deleted with `nabu task rm <id>`, whatever
   its status. Delete only when the user asks; a decision not to do the work
   is a cancel.
4. **Revise with `task replace` and `task set`.** `task replace` takes a
   new title on stdin and keeps the frontmatter; `task set` changes the
   frontmatter and keeps the title.
5. **Break the work down with `task item add`.** When the user gives the
   steps of a task, add them to its checklist, one argument per step, in
   order. When a step is finished, read the numbers with `task item ls`
   and run `nabu task item check <id> <n>`; `task item uncheck` undoes it.
   Editing a line happens in the viewer.
6. **Mark a wait with `task set --waiting`.** When the user has done their
   part and waits on someone else (a reply, a review), keep the task todo
   and run `nabu task set <id> --waiting "<who or what>"`;
   `--clear-waiting` when the ball is back. Do not write the wait into the
   body as well.
7. **Report the id and the frontmatter.** After a write, tell the user
   the title, the id as `#<id>`, and the status nabu printed. After
   `task new` and `task set`, also show the `frontmatter` nabu printed, as a YAML block in the order nabu prints
   it, so the user can correct an `area` or `project` you chose on the
   spot.
8. **Fix what the API refuses.** nabu exits 2 with the API's reason when a
   body or a value breaks the rules (a reason line too long, a canceled task
   without a reason); rewrite and run the command again.

## Do not

- Reach the API, D1, or the viewer any other way than nabu.
- Guess the setup. If nabu exits 2 with "no base_url declared" or "not signed
  in", show the user the message as is; it says what to run. Suggest they
  type `! nabu login` to sign in from the prompt.
