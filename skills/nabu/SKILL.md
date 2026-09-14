---
name: nabu
description: Save, append to, read, list, and search the user's personal markdown notes through the `nabu` CLI. Use whenever the user wants something remembered or written down — "memo this", "note it", "save this to my notes", "add this to <file>", "what did I write about X" — including Japanese phrasings such as 「メモして」「ノートに書いて」「〜に追記して」「Notes に残して」「〜について書いてたっけ」. Never edit the notes repository directly; every read and write goes through nabu.
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
nabu doctor --notes                          # warn on non-kebab filenames / missing titles
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
   only for a note that does not exist yet. `write --force` is deprecated;
   do not use it.
3. **Date the entry.** When appending, pass `--heading "## YYYY-MM-DD"` with
   today's date so the note reads as a log. Consecutive appends on the same
   day land under one heading.
4. **Write what the user said, shaped for the file.** Keep their wording and
   facts; tidy into bullets or short paragraphs. Do not add commentary.
5. **Report the path.** After a write, tell the user the relative path nabu
   printed (`--json` gives `path`, `action`, `bytes`, `committed`).
6. **Repair conventions with `mv`.** When `doctor --notes` or the README
   flags a filename, rename with `nabu note mv` and grep for references
   first; fix a missing title with `replace`.

## Do not

- Read or edit files under the notes root with Read / Edit / Write / shell
  redirection. Only nabu touches the root.
- Run git inside the notes root. nabu commits each change itself; pushing
  is the user's job.
- Guess the root. If nabu exits 2 with "no root declared", run `nabu doctor`
  and show the user its output; it tells them what to fix.
