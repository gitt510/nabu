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
nabu note write <path> --content "<text>"   # new note only; refuses to overwrite
```

Flags may follow the positional argument. Multi-line bodies go through stdin:

```bash
nabu note append career/mygoal_2.md --heading "## 2026-09-03" <<'EOF'
- first point
- second point
EOF
```

## How to work

1. **Look before you write.** Run `nabu note ls --json` first and reuse an
   existing directory and file when one fits. Create a new directory only
   when nothing fits, and name it in lowercase kebab-case.
2. **Prefer append.** Notes grow; `append` is the default verb. Use `write`
   only for a note that does not exist yet. Never pass `--force` unless the
   user explicitly asks to replace a note.
3. **Date the entry.** When appending, pass `--heading "## YYYY-MM-DD"` with
   today's date so the note reads as a log. Consecutive appends on the same
   day land under one heading.
4. **Write what the user said, shaped for the file.** Keep their wording and
   facts; tidy into bullets or short paragraphs. Do not add commentary.
5. **Report the path.** After a write, tell the user the relative path nabu
   printed (`--json` gives `path`, `action`, `bytes`, `committed`).

## Do not

- Read or edit files under the notes root with Read / Edit / Write / shell
  redirection. Only nabu touches the root.
- Run git inside the notes root. nabu commits each change itself; pushing
  is the user's job.
- Guess the root. If nabu exits 2 with "no root declared", show the user
  the message; it tells them where the config file goes.
