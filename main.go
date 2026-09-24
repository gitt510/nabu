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
	"os/exec"
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
  note ls      [dir]       list notes under dir (root when omitted)
  note grep    <query>     find lines containing query (case-insensitive)
  task new     <slug>      create tasks/inbox/<slug>.md with optional --scheduled / --ticket frontmatter
  task mv      <slug> <st> move a task to tasks/<st>/ (inbox, doing, done)
  canvas open              start a draft in CANVAS.md for the user to edit by hand
  canvas read|write|diff   read, revise, or see the user's edits to the draft
  canvas save  <slug>      move the draft to writing/<slug>.md and commit
  canvas drop              empty the draft
  init                     create the root declared in the config as a git repository
  doctor                   check the config file, the root, and git readiness
                           (--notes also lints filenames and titles)
  help, -h             print this usage

Paths are relative to the root, must stay inside it, and end in .md.
The root is a git repository; every change is committed as it is made.
Commands that change the root report the result as JSON; read, canvas read,
and canvas diff print raw text; ls, grep, and doctor print text unless --json.
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
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
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

// check is one doctor finding. Status is ok, warn, or fail.
type check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("doctor", "nabu doctor [--notes] [--root <dir>] [--json]\n\n--notes also warns about notes whose filename is not kebab-case, that have no \"# \" title, or that sit under tasks/ outside a status folder")
	root := bindRoot(fs)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	notes := fs.Bool("notes", false, "lint note filenames, titles, and task folders (warn only)")
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	checks := doctor(*root, *notes)
	failed := false
	for _, ch := range checks {
		if ch.Status == "fail" {
			failed = true
		}
	}
	if *asJSON {
		emit(stdout, map[string]any{"ok": !failed, "checks": checks})
	} else {
		for _, ch := range checks {
			fmt.Fprintf(stdout, "%-4s %-10s %s\n", ch.Status, ch.Name, ch.Detail)
		}
	}
	if failed {
		return exitFail
	}
	return exitOK
}

// doctor runs the readiness checks in dependency order and stops at the
// first failure that makes the later checks meaningless.
func doctor(rootFlag string, notes bool) []check {
	var out []check
	add := func(name, status, detail string) {
		out = append(out, check{Name: name, Status: status, Detail: detail})
	}

	if _, err := exec.LookPath("git"); err != nil {
		add("git", "fail", "git not found on PATH")
		return out
	}
	add("git", "ok", "on PATH")

	root := ""
	if rootFlag != "" {
		root = config.ExpandHome(rootFlag)
		add("config", "ok", "skipped: --root given")
	} else {
		cfg, err := config.Load()
		switch {
		case err != nil:
			add("config", "fail", err.Error())
			return out
		case cfg.Root == "":
			add("config", "fail", "no root declared in "+config.Path())
			return out
		}
		add("config", "ok", config.Path())
		root = cfg.Root
	}

	s, err := store.Open(root)
	if err != nil {
		add("root", "fail", err.Error())
		return out
	}
	add("root", "ok", s.Root)

	if out, err := exec.Command("git", "-C", s.Root, "var", "GIT_COMMITTER_IDENT").Output(); err != nil {
		add("identity", "fail", "git cannot resolve user.name / user.email for the root")
	} else {
		ident := strings.TrimSpace(string(out))
		if i := strings.LastIndex(ident, ">"); i >= 0 {
			ident = ident[:i+1]
		}
		add("identity", "ok", ident)
	}

	if out, err := exec.Command("git", "-C", s.Root, "status", "--porcelain").Output(); err != nil {
		add("worktree", "fail", err.Error())
	} else if n := len(strings.Split(strings.TrimSpace(string(out)), "\n")); len(strings.TrimSpace(string(out))) > 0 {
		add("worktree", "warn", fmt.Sprintf("%d uncommitted change(s); nabu commits only the file it writes", n))
	} else {
		add("worktree", "ok", "clean")
	}

	if notes {
		findings, err := s.Lint()
		switch {
		case err != nil:
			add("notes", "fail", err.Error())
		case len(findings) == 0:
			add("notes", "ok", "filenames are kebab-case, every note has a title, tasks sit in status folders")
		default:
			for _, f := range findings {
				add("notes", "warn", fmt.Sprintf("%s: %s (%s)", f.Path, f.Detail, f.Rule))
			}
		}
	}
	return out
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
		"write":   "nabu note write <path> [--content <text>]\n\ncontent is read from stdin unless --content is given; the note must not exist yet (see note replace)",
		"replace": "nabu note replace <path> [--content <text>]\n\nreplaces the whole body of an existing note; content is read from stdin unless --content is given",
	}[action]
	fs := newFlagSet("note "+action, synopsis)
	root := bindRoot(fs)
	content := fs.String("content", "", "note body; stdin is read when omitted")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
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
	fs := newFlagSet("note read", "nabu note read <path>")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
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
	fs := newFlagSet("note ls", "nabu note ls [dir] [--json]\n\nprints path and title per note; --json adds modified and bytes")
	root := bindRoot(fs)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if ok, code := parse(fs, args, 0, 1, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	entries, err := s.List(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if *asJSON {
		if entries == nil {
			entries = []store.Entry{}
		}
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
	fs := newFlagSet("note grep", "nabu note grep <query> [--json]")
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
	if *asJSON {
		if matches == nil {
			matches = []store.Match{}
		}
		return emit(stdout, matches)
	}
	for _, m := range matches {
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
