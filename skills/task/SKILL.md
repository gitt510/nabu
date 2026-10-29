---
name: task
description: File and move the user's tasks through the `nabu` CLI, which keeps them behind the nabu API. Use when the user wants a task recorded, started, or finished — "add a task", "file this as a task", "I started X", "X is done", "drop X", "what am I working on" — including Japanese phrasings such as 「タスクにして」「タスク登録」「着手した」「完了にして」「やらないことにした」「今やってるタスク」. Every read and write goes through nabu.
---

# nabu task

A task lives behind the nabu API, and nabu is the only way to it. Its
status is one of `inbox` (new), `doing` (started), `done` (finished), and
`dropped` (decided against). `nabu task -h` is the source of truth for the
commands and their flags, and `nabu task schema` for every rule a write
must meet: the keys, the values they accept, and the line limit. This file
only says how to use them well.

## Commands

```bash
nabu task new <slug> --area <area> [<frontmatter flags>]  # file in inbox, JSON body on stdin
nabu task replace <slug>                     # the fields to change, JSON on stdin; frontmatter is kept
nabu task set <slug> --waiting "<who/what>"  # the ball is with someone else
nabu task set <slug> --clear-waiting         # the ball is back
nabu task set <slug> <frontmatter flags>    # other metadata; --clear-* drops
nabu task mv <slug> <inbox|doing|done|dropped>  # change the status
nabu task rename <slug> <new-slug>           # new slug, same status and content
nabu task rm <slug>                          # delete a task, whatever its status
nabu task ls [status] [--json]               # status, title, created; --json adds slug, the frontmatter, and the body
nabu task read <slug>                        # a task's content as markdown, any status
nabu task schema                             # the rules a write must meet, as JSON Schema
nabu task grep <query> [--json]              # where something is written in tasks
```

Every command after `new` also takes a task's id in place of its slug: the
number the viewer shows after a title and copies as `#6`. Pass it as `6`,
or quote `'#6'`, since a shell reads an unquoted `#` as a comment.

## How to work

1. **Check what exists first.** `nabu task ls --json` before filing so a
   task is not filed twice, and `nabu task schema` before writing so the
   values and lines meet the rules.
2. **File with `task new`.** The slug names the task; the body (stdin)
   is a JSON object in the task shape, written as below. Metadata goes through
   the flags `nabu task -h` lists, never hand-written in the body.
   Do not invent a `--scheduled` time the user did not give. Leave
   `--project` out when the task spans several or the repo does not
   exist yet, and reuse a project already in `task ls --json` rather
   than coining a near-duplicate.

   ```json
   {
     "title": "<title>",
     "situation": ["<what is so now>"],
     "complication": ["<what goes wrong with it, and why now>"],
     "question": ["<what the task has to answer>"],
     "answer": ["<the answer, or 未定 until there is one>"]
   }
   ```

   The sections are SCQA, the introduction Barbara Minto lays out in *The
   Pyramid Principle*. The body is only what the user reads: no steps,
   commands, or notes for the agent, and no other key. Each string is one
item, without a leading `- `. A line
   holds at most the `maxLength` `nabu task schema` gives; a fact that does
   not fit is cut, not moved elsewhere. The agent finds the details again when it does the work.
   Load the `gitt510-skills:writing-documents` skill, when it is
   available, before writing a body, and write every bullet under it;
   where the two disagree, the task shape here wins.
3. **Move with `task mv`.** Run `nabu task mv <slug> doing` when work
   starts and `... done` when it ends. Never write a status into the body
   or the frontmatter.
   A task the user decides not to do ("やらない", "見送り", "won't do")
   goes to `dropped`, not `done`. First `task replace` with the reason in
   `why_dropped` (`{"why_dropped": ["<reason>"]}`), then
   `nabu task mv <slug> dropped`.
   A task that should not have been filed at all (a duplicate, a mistake,
   「消して」「削除して」) is deleted with `nabu task rm <slug>`, whatever its
   status. Delete only when the user asks; a decision not to do the work
   is `dropped`.
4. **Revise with `task replace` and `task set`.** `task replace` takes the
   keys to change on stdin, in the task shape, and keeps the rest and the
   frontmatter; `task set` changes the frontmatter and keeps the body.
   Write Situation as state ("password auth is
   off"), not as a log of what was done. Revise the body only when the user
   asks; do not propose updates to it.
5. **Mark a wait with `task set --waiting`.** When the user has done their
   part and waits on someone else (a reply, a review), keep the task in
   `doing` and run `nabu task set <slug> --waiting "<who or what>"`;
   `--clear-waiting` when the ball is back. Do not write the wait into the
   body as well.
6. **Report the slug and the frontmatter.** After a write, tell the user
   the slug and status nabu printed (`task mv` and `task rename` print
   `from` and `to`). After `task new` and `task set`, also show the
   `frontmatter` nabu printed, as a YAML block in the order nabu prints
   it, so the user can correct an `area` or `project` you chose on the
   spot.
7. **Fix what the API refuses.** nabu exits 2 with the API's reason when a
   body or a value breaks the rules (a line too long, a dropped task
   without a reason); rewrite and run the command again.

## Do not

- Reach the API, D1, or the viewer any other way than nabu.
- Guess the setup. If nabu exits 2 with "no base_url declared" or "not signed
  in", show the user the message as is; it says what to run. Suggest they
  type `! cloudflared access login <base_url>` to sign in from the prompt.
