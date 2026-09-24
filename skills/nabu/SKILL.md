---
name: nabu
description: Save, append to, read, list, and search the user's personal markdown notes through the `nabu` CLI. Use whenever the user wants something remembered or written down — "memo this", "note it", "save this to my notes", "add this to <file>", "what did I write about X" — including Japanese phrasings such as 「メモして」「ノートに書いて」「〜に追記して」「Notes に残して」「〜について書いてたっけ」. Also use it to co-write a text with the user on the canvas when it will go through rounds before it is final — "draft this", "scaffold a comment", "let's write this together", "polish my edit", 「下書きして」「たたき台を作って」「一緒に書こう」「添削して」「canvas に」. Never edit the notes repository directly; every read and write goes through nabu.
---

# nabu

`nabu` is the only door to the user's notes repository. The root, path rules,
and commit behaviour live in the CLI, not here. Run `nabu -h` and
`nabu note <command> -h` for the current contract; the help text is the
source of truth and this file only says how to use it well.

## Commands

```bash
nabu note ls [dir] --json          # what exists (path, title, modified)
nabu note grep <query> --json      # where something is written
nabu note read <path>              # a note's content
nabu note append <path> --heading "## YYYY-MM-DD" --content "<text>"
nabu note write <path> --content "<text>"    # new note only; refuses to overwrite
nabu note replace <path> --content "<text>"  # existing note only; whole-body revise
nabu note mv <from> <to>                     # rename; never overwrites
nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]...  # tasks/inbox/<slug>.md
nabu task mv <slug> <inbox|doing|done>       # move a task to its status folder
nabu canvas open                             # start a draft in CANVAS.md (stdin); refuses when one is there
nabu canvas diff                             # what the user changed by hand since your last open/write
nabu canvas write                            # revise the draft (stdin)
nabu canvas save <slug> [--ticket <https-url>]...  # writing/<slug>.md; the only canvas step that commits
nabu canvas drop                             # empty the canvas
nabu doctor --notes                          # warn on non-kebab filenames / missing titles / misplaced tasks
```

Flags may follow the positional argument. Multi-line bodies go through stdin:

```bash
nabu note append career/mygoal_2.md --heading "## 2026-09-03" <<'EOF'
- first point
- second point
EOF
```

## How to work

1. **Read the conventions first.** `nabu note read README.md` gives the
   layout and filename rules of this particular root; they win over any
   default here. Then `nabu note ls --json` to see what exists, and reuse an
   existing file when one fits.
2. **Pick the verb by the note's shape.** A log (journal, 1:1, goal
   entries) grows: use `append`. A structured document (proposal, spec,
   README) is revised: `read` it, rewrite the whole body, and `replace`.
   Reordering or rewriting sections the user asked for needs no extra
   permission; dropping content they did not mention does. Use `write`
   only for a note that does not exist yet.
3. **Date the entry.** When appending, pass `--heading "## YYYY-MM-DD"` with
   today's date so the note reads as a log. Consecutive appends on the same
   day land under one heading.
4. **Write what the user said, shaped for the file.** Keep their wording and
   facts; tidy into bullets or short paragraphs. Do not add commentary.
5. **File a task with `task new`.** A task is a note under `tasks/`; the
   slug is the filename, the body (stdin) is ordinary markdown starting with
   a `# ` title. Metadata goes through flags, never hand-written in the body:
   `--scheduled` is the time the work is planned to happen (RFC3339 with
   offset, not a deadline), `--ticket` is a full `https` issue URL, repeatable.
   Later edits use `note replace`, which rewrites the whole file: keep the
   frontmatter block in the new body.
6. **Move a task with `task mv`.** The folder is the task's status:
   `tasks/inbox/` (new), `tasks/doing/` (started), `tasks/done/` (finished).
   Never write a status into the body or frontmatter; run
   `nabu task mv <slug> doing` when work starts and `... done` when it ends.
   `nabu note ls tasks/doing --json` lists what is in flight.
7. **Report the path.** After a write, tell the user the relative path nabu
   printed (every write prints JSON with `path`, `action`, `bytes`, `committed`).
8. **Repair conventions with `mv`.** When `doctor --notes` or the README
   flags a filename, rename with `nabu note mv` and grep for references
   first; fix a missing title with `replace`. A task flagged as outside a
   status folder moves with `nabu note mv tasks/<slug>.md tasks/inbox/<slug>.md`.

## Co-writing on a canvas

The canvas is a single draft file, `CANVAS.md` at the notes root, that the
user edits by hand while you revise it through the CLI. Use it when the
text will go through rounds before it is final: a PR comment, a proposal,
a message to someone. 「メモして」「追記して」 is still `note`; the canvas is
for text the user wants to shape with you.

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
- Run git inside the notes root. nabu commits each change itself; pushing
  is the user's job.
- Guess the root. If nabu exits 2 with "no root declared", run `nabu doctor`
  and show the user its output; it tells them what to fix.
