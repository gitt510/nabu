# nabu

nabu is the scribe for one markdown notes repository: it reads, writes, replaces,
appends, renames, lists, and greps notes under a declared root and commits every
change, so an agent never edits the notes by hand. One file, `CANVAS.md`, is the
exception: a draft the user edits by hand and the agent revises through nabu,
outside git, until `canvas save` turns it into a note.

## Requirements

- `git` on PATH
- The root is an existing git repository
- Go 1.26+ to build (`mise install` provides it)

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
echo '{"root": "~/ghq/github.com/gitt510/notes"}' > ~/.config/nabu/config.json
nabu init
nabu doctor
```

- The config file is `$XDG_CONFIG_HOME/nabu/config.json`, falling back to `~/.config/nabu/config.json`
- `root` is the only key; `~` expands to the home directory
- `nabu init` creates the declared root, runs `git init -b main`, and seeds `README.md` and a `.gitignore` for the canvas files with a first commit; rerunning it changes nothing
- `nabu doctor` checks git on PATH, the config file, the root, the git identity, and the working tree; exit 1 when any check fails
- `nabu doctor --notes` also warns (never fails) about notes whose filename is not lowercase kebab-case or that have no `# ` title; `README.md` is exempt

## Usage

```bash
nabu note ls [dir]                       # list notes, sorted by path
nabu note grep <query>                   # lines containing query, case-insensitive
nabu note read <path>                    # print a note
nabu note write <path> --content <text>   # create a note (must not exist)
nabu note replace <path> --content <text> # replace the body of an existing note
nabu note append <path> --content <text>  # append to a note
nabu note mv <from> <to>                  # rename a note (destination must not exist)
nabu task new <slug> [--scheduled <RFC3339>] [--ticket <url>]...  # create tasks/inbox/<slug>.md
nabu task mv <slug> <inbox|doing|done>    # move a task to its status folder
nabu canvas open --content <text>         # start a draft in CANVAS.md (must be empty)
nabu canvas read                          # the draft as it is now
nabu canvas diff                          # the user's edits since the agent last wrote
nabu canvas write --content <text>        # replace the draft with a revision
nabu canvas save <slug> [--ticket <url>]... # move the draft to writing/<slug>.md and commit
nabu canvas drop                          # empty the canvas
nabu -h / nabu note <command> -h         # usage
```

- `--content` may be omitted; the body is then read from stdin
- Flags may come before or after the positional argument
- Commands that change the root print their result as JSON. `read`, `canvas read`, and `canvas diff` print raw text. `ls`, `grep`, and `doctor` print text unless `--json`
- `--root <dir>` overrides the config file for one invocation

## Behavior

- Paths are relative to the root, must stay inside it, and must end in `.md`
- `write` refuses an existing note; `replace` refuses a missing one, so the two verbs cannot be confused
- `mv` refuses an existing destination and commits the removal and the addition together
- `append` creates the note when absent and separates entries with a blank line
- `append --heading "## 2026-09-03"` writes the heading once; later appends under the same heading join the section
- A task's status is its folder: `tasks/inbox/`, `tasks/doing/`, `tasks/done/`. No frontmatter field records it
- `task new` writes `tasks/inbox/<slug>.md` (slug: lowercase kebab-case, no `/` or `.md`) and refuses a slug already present in any status folder. The body comes from `--content` or stdin and is not constrained. `--scheduled` (RFC3339 with offset, the planned work time) and repeatable `--ticket` (full `https` URL) become a YAML frontmatter block; with neither flag no block is written. A body that itself starts with `---` is rejected when flags are given
- `task mv` finds the task by slug in any status folder and moves it to `tasks/<status>/`; it fails on an unknown slug or when the task is already in that status
- The canvas is `CANVAS.md` at the root, one draft at a time. `open` refuses a non-empty canvas; `write`, `read`, `diff`, and `save` refuse an empty one
- `canvas open` and `canvas write` record the body they wrote in `.CANVAS.agent.md`; `canvas diff` is `git diff --no-index` from that snapshot to the file, so its output is exactly what the user changed by hand. A canvas started by hand diffs against an empty file
- `canvas save <slug>` writes the draft to `writing/<slug>.md` (created when absent, replaced when present) behind a frontmatter block with `created` (RFC3339, local offset) and any `--ticket` URLs, commits it as `nabu: canvas save writing/<slug>.md`, and empties the canvas. A draft that itself starts with `---` is refused
- `canvas save` and `canvas drop` empty `CANVAS.md` rather than deleting it, so an editor with the file open sees a reload, and remove the snapshot
- `CANVAS.md` and `.CANVAS.agent.md` are in the `.gitignore` that `init` seeds; a root initialized before the canvas existed needs the two lines added by hand
- `ls`, `grep`, and `doctor --notes` skip the canvas files; `note write`, `replace`, `append`, `mv`, and `read` refuse them
- `write`, `replace`, `append`, and `task new` commit the changed file as `nabu: <command> <path>` (`task new` as `nabu: task <path>`); `mv` commits as `nabu: mv <from> -> <to>`, `task mv` as `nabu: task mv <from> -> <to>`. There is no way to leave a change uncommitted
- `doctor --notes` also warns about a note under `tasks/` that is not directly inside a status folder
- `write`, `replace`, `append`, `task new`, and `canvas save` print `path`, `action`, `bytes`, `committed`; `mv` and `task mv` print `from`, `to`, `action`, `committed`; `canvas open`, `write`, and `drop` print `path` (`CANVAS.md`), `file` (absolute), `action`, `bytes`
- `ls` skips `.git` and non-`.md` files and prints path and title; `--json` adds `modified` and `bytes`
- `grep --json` carries `path`, `line`, `text`

| exit | meaning |
| --- | --- |
| 0 | success (help included) |
| 1 | the operation failed (missing note, existing note, git error) |
| 2 | wrong invocation or no root declared |

## Agent skill

```
/plugin marketplace add gitt510/nabu
/plugin install nabu@nabu
```

- The skill in `skills/nabu/SKILL.md` directs the agent to use nabu for every read and write under the root

## Development

```bash
just test     # go test ./...
just check    # gofmt, go vet, golangci-lint
just run ...  # go run . <args>
```

- User-facing strings are English; Japanese is allowed in comments and tests (enforced by `gosmopolitan`)
