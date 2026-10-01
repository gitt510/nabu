// nabu keeps one markdown repository for tasks and drafts. Every task lives
// under tasks/ and every write is a git commit, so an agent (or a human)
// never edits the root by hand. One file, CANVAS.md, is the exception: a
// draft the user edits by hand and the agent revises through nabu.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gitt510/nabu/internal/config"
	"github.com/gitt510/nabu/internal/store"
)

const usage = `usage: nabu <command> [args]

  task new     <slug>      create tasks/inbox/<slug>.md with optional --scheduled / --ticket frontmatter
  task replace <slug>      replace a task's body, keeping its frontmatter
  task set     <slug>      change a task's frontmatter (--scheduled, --waiting, --ticket, --clear-*)
  task mv      <slug> <st> move a task to tasks/<st>/ (inbox, doing, done)
  task rename  <slug> <new> give a task a new slug, keeping its status and content
  task read    <slug>      print a task, whatever its status
  task ls      [st]        list tasks with status, title, and waiting (--json adds the rest)
  task grep    <query>     find lines containing query under tasks/
  task validate [slug]     check one task, or every task, against the task shape
  canvas open              start a draft in CANVAS.md for the user to edit by hand
  canvas read|write|diff   read, revise, or see the user's edits to the draft
  canvas save  <slug>      move the draft to gallery/<slug>.md and commit
  canvas drop              empty the draft
  push                     git push the root to its upstream
  init                     create the root declared in the config as a git repository
  help, -h                 print this usage

The root is a git repository; every change is committed as it is made.
Commands that change the root report the result as JSON; task read, canvas
read, and canvas diff print raw text; task ls, task grep, and task validate
print text unless --json.
CANVAS.md is ignored by git.

The root comes from --root <dir>, or else from root in ` + "%s" + `
See "nabu <command> -h" for the flags of each command.
`

// exit codes: 0 ok, 1 the operation failed, 2 the invocation was wrong
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, rootUsage())
		return exitUsage
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, rootUsage())
		return exitOK
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "push":
		return runPush(args[1:], stdout, stderr)
	case "task":
		return runTask(args[1:], stdin, stdout, stderr)
	case "canvas":
		return runCanvas(args[1:], stdin, stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown command: %s\n\n%s", args[0], rootUsage())
	return exitUsage
}

func rootUsage() string { return fmt.Sprintf(usage, config.Path()) }

// errNoRoot is a setup problem, not an operation failure, so it exits 2.
var errNoRoot = errors.New("no root declared")

func setupMessage() string {
	return fmt.Sprintf("no root declared.\n\ncreate %s like:\n\n%s\nor pass --root <dir>.\n", config.Path(), config.Example)
}

// bindRoot adds the --root flag every command takes.
func bindRoot(fs *flag.FlagSet) *string {
	return fs.String("root", "", "repository root (overrides the config file)")
}

// resolveRoot applies flag > config.
func resolveRoot(flagRoot string) (string, error) {
	if flagRoot != "" {
		return config.ExpandHome(flagRoot), nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.Root == "" {
		return "", errNoRoot
	}
	return cfg.Root, nil
}

func openStore(flagRoot string) (*store.Store, error) {
	root, err := resolveRoot(flagRoot)
	if err != nil {
		return nil, err
	}
	return store.Open(root)
}

// newFlagSet returns a FlagSet whose output parse decides: help goes to
// stdout with exit 0, a wrong invocation to stderr with exit 2.
func newFlagSet(name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s\n\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// parse runs fs.Parse, checks that between min and max positional args
// remain, and maps the outcome to an exit code. ok is true when the caller
// should continue.
func parse(fs *flag.FlagSet, args []string, minArgs, maxArgs int, stdout, stderr io.Writer) (ok bool, code int) {
	err := parseInterspersed(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		fs.SetOutput(stdout)
		fs.Usage()
		return false, exitOK
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
	}
	if err != nil || fs.NArg() < minArgs || fs.NArg() > maxArgs {
		fs.SetOutput(stderr)
		fs.Usage()
		return false, exitUsage
	}
	return true, exitOK
}

// parseInterspersed lets flags follow positional args ("task new a
// --root x"), which the flag package does not do on its own. Positionals
// are collected in order and restored through fs.Args after the last pass.
func parseInterspersed(fs *flag.FlagSet, args []string) error {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	return fs.Parse(append([]string{"--"}, positional...))
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("init", "nabu init [--root <dir>]\n\ncreates the directory, runs git init -b main, and seeds README.md and .gitignore with a first commit; every step is skipped when already done")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	dir, err := resolveRoot(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	r, err := store.Init(dir)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, r)
}

func runPush(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("push", "nabu push [--root <dir>]\n\nruns git push for the root's current branch; the branch must already have an upstream")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	upstream, err := s.Push()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, map[string]string{"action": "push", "upstream": upstream})
}

// writeResult is the JSON document of every command that writes a file.
type writeResult struct {
	Path      string `json:"path"`
	Action    string `json:"action"`
	Bytes     int    `json:"bytes"`
	Committed bool   `json:"committed"`
}

// finishWrite commits the written file as "nabu: <action> <path>" and
// reports it.
func finishWrite(s *store.Store, rel, action string, n int, stdout, stderr io.Writer) int {
	committed, err := s.Commit(fmt.Sprintf("nabu: %s %s", action, rel), rel)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, writeResult{Path: rel, Action: action, Bytes: n, Committed: committed})
}

// mvResult is the JSON document of every command that moves a file.
type mvResult struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Action    string `json:"action"`
	Committed bool   `json:"committed"`
}

// finishMv commits the move as "nabu: <action> <from> -> <to>" and
// reports it.
func finishMv(s *store.Store, from, to, action string, stdout, stderr io.Writer) int {
	committed, err := s.Commit(fmt.Sprintf("nabu: %s %s -> %s", action, from, to), from, to)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, mvResult{From: from, To: to, Action: action, Committed: committed})
}

// readBody returns --content when given, otherwise all of stdin.
func readBody(content string, stdin io.Reader) ([]byte, error) {
	if content != "" {
		return []byte(content), nil
	}
	b, err := io.ReadAll(stdin)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, errors.New("empty body: pass --content or pipe text on stdin")
	}
	return b, nil
}

// hasFrontmatter reports whether a body opens with a YAML block of its own.
func hasFrontmatter(body []byte) bool {
	return strings.HasPrefix(strings.TrimLeft(string(body), "\n"), "---")
}

func emit(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return exitFail
	}
	return exitOK
}

func fail(stderr io.Writer, err error, code int) int {
	if errors.Is(err, errNoRoot) {
		fmt.Fprint(stderr, setupMessage())
		return exitUsage
	}
	fmt.Fprintln(stderr, err)
	return code
}
