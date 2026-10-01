// nabu is the scribe for a notes repository. It reads, writes, appends,
// lists, and greps markdown notes under one declared root and records each
// write as a git commit, so an agent (or a human) never edits the root by
// hand.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gitt510/nabu/internal/config"
	"github.com/gitt510/nabu/internal/store"
)

const usage = `usage: nabu <command> [args]

  note write   <path>      create a note (refuses to overwrite)
  note replace <path>      replace the body of an existing note
  note append  <path>      append to a note, creating it when absent
  note mv      <from> <to> rename a note (refuses to overwrite)
  note read    <path>      print a note
  note ls      [dir]       list notes under dir (root when omitted), tasks/ left out
  note grep    <query>     find lines containing query (case-insensitive), tasks/ left out
  task new     <slug>      create tasks/inbox/<slug>.md with optional --scheduled / --ticket frontmatter
  task replace <slug>      replace a task's body, keeping its frontmatter
  task set     <slug>      change a task's frontmatter (--scheduled, --waiting, --ticket, --clear-*)
  task mv      <slug> <st> move a task to tasks/<st>/ (inbox, doing, done)
  task read    <slug>      print a task, whatever its status
  task ls      [st]        list tasks with status, title, and waiting (--json adds the rest)
  task grep    <query>     find lines containing query under tasks/
  task validate [slug]     check one task, or every task, against the task shape
  canvas open              start a draft in CANVAS.md for the user to edit by hand
  canvas read|write|diff   read, revise, or see the user's edits to the draft
  canvas save  <slug>      move the draft to writing/<slug>.md and commit
  canvas drop              empty the draft
  push                     git push the root to its upstream
  init                     create the root declared in the config as a git repository
  help, -h             print this usage

Paths are relative to the root, must stay inside it, and end in .md.
tasks/ belongs to the task commands: the note commands refuse a path under
it and leave it out of ls and grep.
The root is a git repository; every change is committed as it is made.
Commands that change the root report the result as JSON; read, canvas read,
and canvas diff print raw text; ls, grep, and task validate print text unless --json.
CANVAS.md is ignored by git and is not a note.

The root comes from --root <dir>, or else from root in ` + "%s" + `
See "nabu <command> -h" for the flags of each command.
`

const noteUsage = `usage: nabu note <write|replace|append|mv|read|ls|grep> [flags] [args]

See "nabu note <command> -h".
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
	case "note":
		return runNote(args[1:], stdin, stdout, stderr)
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
	return fs.String("root", "", "notes root (overrides the config file)")
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

// parseInterspersed lets flags follow positional args ("note write a.md
// --root x"), which the flag package does not do on its own. Positionals are
// collected in order and restored through fs.Args after the last pass.
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

func runNote(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, noteUsage)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(stdout, noteUsage)
		return exitOK
	case "write", "replace":
		return runWrite(args[1:], args[0], stdin, stdout, stderr)
	case "mv":
		return runMv(args[1:], stdout, stderr)
	case "append":
		return runAppend(args[1:], stdin, stdout, stderr)
	case "read":
		return runRead(args[1:], stdout, stderr)
	case "ls":
		return runLs(args[1:], stdout, stderr)
	case "grep":
		return runGrep(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown note command: %s\n\n%s", args[0], noteUsage)
	return exitUsage
}

// writeResult is the JSON document of every command that writes a note.
type writeResult struct {
	Path      string `json:"path"`
	Action    string `json:"action"`
	Bytes     int    `json:"bytes"`
	Committed bool   `json:"committed"`
}

// runWrite is note write and note replace: the same body command, differing
// only in whether the note must be absent or present.
func runWrite(args []string, action string, stdin io.Reader, stdout, stderr io.Writer) int {
	synopsis := map[string]string{
		"write":   "nabu note write <path> [--content <text>]\n\ncontent is read from stdin unless --content is given; the note must not exist yet (see note replace).\na path under tasks/ is refused: tasks are written by nabu task",
		"replace": "nabu note replace <path> [--content <text>]\n\nreplaces the whole body of an existing note; content is read from stdin unless --content is given.\na path under tasks/ is refused: tasks are written by nabu task",
	}[action]
	fs := newFlagSet("note "+action, synopsis)
	root := bindRoot(fs)
	content := fs.String("content", "", "note body; stdin is read when omitted")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	if isTaskPath(fs.Arg(0)) {
		return fail(stderr, fmt.Errorf("%s %w", fs.Arg(0), errTaskPath), exitUsage)
	}
	body, err := readBody(*content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	var rel string
	if action == "write" {
		rel, err = s.Write(fs.Arg(0), body)
	} else {
		rel, err = s.Replace(fs.Arg(0), body)
	}
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, action, len(body), stdout, stderr)
}

// mvResult is the JSON document of mv.
type mvResult struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Action    string `json:"action"`
	Committed bool   `json:"committed"`
}

func runMv(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("note mv", "nabu note mv <from> <to>\n\nrenames a note; the destination must not exist")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 2, 2, stdout, stderr); !ok {
		return code
	}
	for _, p := range fs.Args() {
		if isTaskPath(p) {
			return fail(stderr, fmt.Errorf("%s %w", p, errTaskPath), exitUsage)
		}
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	from, to, err := s.Move(fs.Arg(0), fs.Arg(1))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishMv(s, from, to, "mv", stdout, stderr)
}

func finishMv(s *store.Store, from, to, action string, stdout, stderr io.Writer) int {
	committed, err := s.Commit(fmt.Sprintf("nabu: %s %s -> %s", action, from, to), from, to)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, mvResult{From: from, To: to, Action: action, Committed: committed})
}

func runAppend(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("note append", "nabu note append <path> [--content <text>] [--heading <line>]\n\ncontent is read from stdin unless --content is given")
	root := bindRoot(fs)
	content := fs.String("content", "", "text to append; stdin is read when omitted")
	heading := fs.String("heading", "", "markdown heading to append under (written once per run of appends, e.g. \"## 2026-09-03\")")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	if *heading != "" && !strings.HasPrefix(*heading, "#") {
		return fail(stderr, errors.New("--heading must start with #"), exitUsage)
	}
	if isTaskPath(fs.Arg(0)) {
		return fail(stderr, fmt.Errorf("%s %w", fs.Arg(0), errTaskPath), exitUsage)
	}
	body, err := readBody(*content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	rel, err := s.Append(fs.Arg(0), body, *heading)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "append", len(body), stdout, stderr)
}

// finishWrite commits the written note as "nabu: <action> <path>" and
// reports it.
func finishWrite(s *store.Store, rel, action string, n int, stdout, stderr io.Writer) int {
	committed, err := s.Commit(fmt.Sprintf("nabu: %s %s", action, rel), rel)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, writeResult{Path: rel, Action: action, Bytes: n, Committed: committed})
}

func runRead(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("note read", "nabu note read <path>\n\na path under tasks/ is refused: see nabu task read")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	if isTaskPath(fs.Arg(0)) {
		return fail(stderr, fmt.Errorf("%s %w", fs.Arg(0), errTaskPath), exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	b, err := s.Read(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	_, _ = stdout.Write(b)
	return exitOK
}

func runLs(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("note ls", "nabu note ls [dir] [--json]\n\nprints path and title per note; --json adds modified and bytes.\ntasks/ is left out: see nabu task ls")
	root := bindRoot(fs)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if ok, code := parse(fs, args, 0, 1, stdout, stderr); !ok {
		return code
	}
	if isTaskPath(fs.Arg(0)) {
		return fail(stderr, fmt.Errorf("%s %w", fs.Arg(0), errTaskPath), exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	all, err := s.List(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	entries := []store.Entry{}
	for _, e := range all {
		if !isTaskPath(e.Path) {
			entries = append(entries, e)
		}
	}
	if *asJSON {
		return emit(stdout, entries)
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, e := range entries {
		fmt.Fprintf(tw, "%s\t%s\n", e.Path, e.Title)
	}
	_ = tw.Flush()
	return exitOK
}

func runGrep(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("note grep", "nabu note grep <query> [--json]\n\ntasks/ is left out: see nabu task grep")
	root := bindRoot(fs)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	matches, err := s.Grep(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return printMatches(stdout, matches, *asJSON, func(p string) bool { return !isTaskPath(p) })
}

// printMatches prints the grep hits whose path passes keep, as text or JSON.
func printMatches(stdout io.Writer, matches []store.Match, asJSON bool, keep func(string) bool) int {
	kept := []store.Match{}
	for _, m := range matches {
		if keep(m.Path) {
			kept = append(kept, m)
		}
	}
	if asJSON {
		return emit(stdout, kept)
	}
	for _, m := range kept {
		fmt.Fprintf(stdout, "%s:%d: %s\n", m.Path, m.Line, m.Text)
	}
	return exitOK
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
