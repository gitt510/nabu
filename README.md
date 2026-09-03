# nabu

nabu is the scribe for one markdown notes repository: it reads, writes, appends,
lists, and greps notes under a declared root and commits every write, so an agent
never edits the notes by hand.

## Requirements

- `git` on PATH
- The root is an existing git repository
- Go 1.26+ to build (`mise install` provides it)

## Setup

```bash
go install github.com/gitt510/nabu@latest
mkdir -p ~/.config/nabu
cat > ~/.config/nabu/config.toml <<'EOF'
root = "~/ghq/github.com/gitt510/notes"
EOF
nabu init
nabu doctor
```

- The config file is `$XDG_CONFIG_HOME/nabu/config.toml`, falling back to `~/.config/nabu/config.toml`
- `root` is the only key; `~` expands to the home directory
- `nabu init` creates the declared root, runs `git init -b main`, and seeds `README.md` with a first commit; rerunning it changes nothing
- `nabu doctor` checks git on PATH, the config file, the root, the git identity, and the working tree; exit 1 when any check fails

## Usage

```bash
nabu note ls [dir]                       # list notes, sorted by path
nabu note grep <query>                   # lines containing query, case-insensitive
nabu note read <path>                    # print a note
nabu note write <path> --content <text>  # create a note
nabu note append <path> --content <text> # append to a note
nabu -h / nabu note <command> -h         # usage
```

- `--content` may be omitted; the body is then read from stdin
- Flags may come before or after the positional argument
- `--json` on any command emits the result as JSON
- `--root <dir>` overrides the config file for one invocation

## Behavior

- Paths are relative to the root, must stay inside it, and must end in `.md`
- `write` refuses an existing note; `--force` overwrites it
- `append` creates the note when absent and separates entries with a blank line
- `append --heading "## 2026-09-03"` writes the heading once; later appends under the same heading join the section, and consecutive list items form one list
- `write` and `append` commit the changed file as `nabu: <command> <path>`; `--no-commit` leaves the change uncommitted
- `ls` skips `.git` and non-`.md` files; `--json` carries `path`, `title` (first `# ` heading), `modified`, `bytes`
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
