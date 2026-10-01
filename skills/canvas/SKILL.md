---
name: canvas
description: Co-write a text with the user on the canvas of their notes repository through the `nabu` CLI, when it will go through rounds before it is final — "draft this", "scaffold a comment", "let's write this together", "polish my edit", 「下書きして」「たたき台を作って」「一緒に書こう」「添削して」「canvas に」. The user edits CANVAS.md by hand, the agent revises it through nabu, and `canvas save` turns it into a note. Never read or edit the canvas directly; every read and write goes through nabu.
---

# nabu canvas

The canvas is a single draft file, `CANVAS.md` at the notes root, that the
user edits by hand while you revise it through the CLI. Use it when the
text will go through rounds before it is final: a PR comment, a proposal,
a message to someone. 「メモして」「追記して」 is the `nabu:note` skill; the
canvas is for text the user wants to shape with you. `nabu canvas -h` is
the source of truth for the contract.

## Commands

```bash
nabu canvas open                             # start a draft in CANVAS.md (stdin); refuses when one is there
nabu canvas diff                             # what the user changed by hand since your last open/write
nabu canvas read                             # the draft as it is now
nabu canvas write                            # revise the draft (stdin)
nabu canvas save <slug> [--ticket <https-url>]...  # writing/<slug>.md; the only canvas step that commits
nabu canvas drop                             # empty the canvas
```

## How to work

1. `nabu canvas open` with the first draft on stdin. If it refuses because
   the canvas is not empty, ask the user whether to save or drop what is
   there. Tell the user the path it prints and stop; they edit it in their
   editor.
2. When the user says they are done (or asks for a review), run
   `nabu canvas diff` first. The diff is the user's edit; read it before
   the full text (`nabu canvas read`). Answer questions about the draft
   from this.
3. Revise with `nabu canvas write` (whole body on stdin). Keep the user's
   wording where the diff shows they changed it deliberately. Repeat 2-3.
4. When the user says it is final, `nabu canvas save <slug> [--ticket <url>]`.
   It lands in `writing/<slug>.md` with a `created` frontmatter. Report the
   path. Only this step commits.

## Do not

- Read or edit `CANVAS.md` with Read / Edit; that bypasses the snapshot
  `canvas diff` relies on. Go through the canvas commands.
- `canvas write` while the user is editing. Run `canvas diff` first so their
  edits are read, not overwritten. Do not `canvas save` until they say so.
- Read or edit files under the notes root with Read / Edit / Write / shell
  redirection. Only nabu touches the root.
- Run git inside the notes root. nabu commits each change itself; `nabu
  push` only when the user asks.
- Guess the root. If nabu exits 2 with "no root declared", show the user
  the message; it tells them what to fix.
