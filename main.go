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
  init                     create the root declared in the config as a git repository
  doctor                   check the config file, the root, and git readiness
                           (--notes also lints filenames and titles)
  help, -h             print this usage

Paths are relative to the root, must stay inside it, and end in .md.
The root is a git repository; write, replace, append, and mv commit
their change unless --no-commit is given. Every command accepts --json.

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

// common holds the flags every command shares.
type common struct {
	root string
	json bool
}

func (c *common) bind(fs *flag.FlagSet) {
	fs.StringVar(&c.root, "root", "", "notes root (overrides the config file)")
	fs.BoolVar(&c.json, "json", false, "print the result as JSON")
}

// resolveRoot applies flag > config.
func (c *common) resolveRoot() (string, error) {
	if c.root != "" {
		return config.ExpandHome(c.root), nil
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

func (c *common) open() (*store.Store, error) {
	root, err := c.resolveRoot()
	if err != nil {
		return nil, err
	}
	return store.Open(root)
}

// newFlagSet returns a FlagSet whose -h prints usage to stdout and exits 0,
// and whose parse errors go to stderr with exit 2. Both are reported through
// the returned help/err by the caller's parse.
func newFlagSet(name, synopsis string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // parse reports errors and help itself
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: %s\n\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// parse runs fs.Parse and maps its outcome to an exit code. ok is true when
// the caller should continue.
func parse(fs *flag.FlagSet, args []string, stdout, stderr io.Writer) (ok bool, code int) {
	err := parseInterspersed(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		fs.SetOutput(stdout)
		fs.Usage()
		return false, exitOK
	}
	if err != nil {
		fs.SetOutput(stderr)
		fmt.Fprintln(stderr, err)
		fs.Usage()
		return false, exitUsage
	}
	return true, exitOK
}

// parseInterspersed lets flags follow positional args ("note write a.md
// --json"), which the flag package does not do on its own. Positionals are
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
	var c common
	fs := newFlagSet("init", "nabu init [--root <dir>] [--json]\n\ncreates the directory, runs git init -b main, and seeds README.md with a first commit; every step is skipped when already done", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	root, err := c.resolveRoot()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	r, err := store.Init(root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		return emit(stdout, r)
	}
	var did []string
	if r.CreatedDir {
		did = append(did, "created directory")
	}
	if r.InitedGit {
		did = append(did, "git init")
	}
	if r.Committed {
		did = append(did, "seeded README.md")
	}
	if len(did) == 0 {
		fmt.Fprintf(stdout, "%s already initialized\n", r.Root)
	} else {
		fmt.Fprintf(stdout, "%s: %s\n", r.Root, strings.Join(did, ", "))
	}
	return exitOK
}

// check is one doctor finding. Status is ok, warn, or fail.
type check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	var c common
	var notes bool
	fs := newFlagSet("doctor", "nabu doctor [--notes] [--root <dir>] [--json]\n\n--notes also warns about notes whose filename is not kebab-case or that have no \"# \" title", stderr)
	c.bind(fs)
	fs.BoolVar(&notes, "notes", false, "lint note filenames and titles (warn only)")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	checks := doctor(c.root, notes)
	failed := false
	for _, ch := range checks {
		if ch.Status == "fail" {
			failed = true
		}
	}
	if c.json {
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
			add("notes", "ok", "filenames are kebab-case and every note has a title")
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
	case "write":
		return runWrite(args[1:], stdin, stdout, stderr)
	case "replace":
		return runReplace(args[1:], stdin, stdout, stderr)
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

// writeResult is the JSON document of write, replace, and append.
type writeResult struct {
	Path      string `json:"path"`
	Action    string `json:"action"`
	Bytes     int    `json:"bytes"`
	Committed bool   `json:"committed"`
}

func runWrite(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var c common
	var content string
	var force, noCommit bool
	fs := newFlagSet("note write", "nabu note write <path> [--content <text>] [--no-commit] [--json]\n\ncontent is read from stdin unless --content is given; the note must not exist yet (see note replace)", stderr)
	c.bind(fs)
	fs.StringVar(&content, "content", "", "note body; stdin is read when omitted")
	fs.BoolVar(&force, "force", false, "deprecated: use note replace")
	fs.BoolVar(&noCommit, "no-commit", false, "leave the change uncommitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	if force {
		fmt.Fprintln(stderr, "warning: --force is deprecated and will be removed; use \"nabu note replace\" to revise an existing note")
	}
	body, err := readBody(content, fs.Lookup("content"), stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	rel, err := s.Write(fs.Arg(0), body, force)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "write", len(body), noCommit, c.json, stdout, stderr)
}

func runReplace(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var c common
	var content string
	var noCommit bool
	fs := newFlagSet("note replace", "nabu note replace <path> [--content <text>] [--no-commit] [--json]\n\nreplaces the whole body of an existing note; content is read from stdin unless --content is given", stderr)
	c.bind(fs)
	fs.StringVar(&content, "content", "", "new note body; stdin is read when omitted")
	fs.BoolVar(&noCommit, "no-commit", false, "leave the change uncommitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	body, err := readBody(content, fs.Lookup("content"), stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	rel, err := s.Replace(fs.Arg(0), body)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "replace", len(body), noCommit, c.json, stdout, stderr)
}

// mvResult is the JSON document of mv.
type mvResult struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Action    string `json:"action"`
	Committed bool   `json:"committed"`
}

func runMv(args []string, stdout, stderr io.Writer) int {
	var c common
	var noCommit bool
	fs := newFlagSet("note mv", "nabu note mv <from> <to> [--no-commit] [--json]\n\nrenames a note; the destination must not exist", stderr)
	c.bind(fs)
	fs.BoolVar(&noCommit, "no-commit", false, "leave the change uncommitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 2 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	from, to, err := s.Move(fs.Arg(0), fs.Arg(1))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	committed := false
	if !noCommit {
		committed, err = s.Commit(fmt.Sprintf("nabu: mv %s -> %s", from, to), from, to)
		if err != nil {
			return fail(stderr, err, exitFail)
		}
	}
	if c.json {
		return emit(stdout, mvResult{From: from, To: to, Action: "mv", Committed: committed})
	}
	state := "committed"
	if !committed {
		state = "not committed"
	}
	fmt.Fprintf(stdout, "mv %s -> %s (%s)\n", from, to, state)
	return exitOK
}

func runAppend(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var c common
	var content, heading string
	var noCommit bool
	fs := newFlagSet("note append", "nabu note append <path> [--content <text>] [--heading <line>] [--no-commit] [--json]\n\ncontent is read from stdin unless --content is given", stderr)
	c.bind(fs)
	fs.StringVar(&content, "content", "", "text to append; stdin is read when omitted")
	fs.StringVar(&heading, "heading", "", "markdown heading to append under (written once per run of appends, e.g. \"## 2026-09-03\")")
	fs.BoolVar(&noCommit, "no-commit", false, "leave the change uncommitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	if heading != "" && !strings.HasPrefix(heading, "#") {
		return fail(stderr, errors.New("--heading must start with #"), exitUsage)
	}
	body, err := readBody(content, fs.Lookup("content"), stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	rel, err := s.Append(fs.Arg(0), body, heading)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "append", len(body), noCommit, c.json, stdout, stderr)
}

func finishWrite(s *store.Store, rel, action string, n int, noCommit, asJSON bool, stdout, stderr io.Writer) int {
	committed := false
	if !noCommit {
		var err error
		committed, err = s.Commit(fmt.Sprintf("nabu: %s %s", action, rel), rel)
		if err != nil {
			return fail(stderr, err, exitFail)
		}
	}
	if asJSON {
		return emit(stdout, writeResult{Path: rel, Action: action, Bytes: n, Committed: committed})
	}
	state := "committed"
	if !committed {
		state = "not committed"
	}
	fmt.Fprintf(stdout, "%s %s (%d bytes, %s)\n", action, rel, n, state)
	return exitOK
}

func runRead(args []string, stdout, stderr io.Writer) int {
	var c common
	fs := newFlagSet("note read", "nabu note read <path> [--json]", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	b, err := s.Read(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		return emit(stdout, map[string]string{"path": fs.Arg(0), "content": string(b)})
	}
	_, _ = stdout.Write(b)
	return exitOK
}

func runLs(args []string, stdout, stderr io.Writer) int {
	var c common
	fs := newFlagSet("note ls", "nabu note ls [dir] [--json]", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() > 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	entries, err := s.List(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		if entries == nil {
			entries = []store.Entry{}
		}
		return emit(stdout, entries)
	}
	for _, e := range entries {
		fmt.Fprintln(stdout, e.Path)
	}
	return exitOK
}

func runGrep(args []string, stdout, stderr io.Writer) int {
	var c common
	fs := newFlagSet("note grep", "nabu note grep <query> [--json]", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	matches, err := s.Grep(fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
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

// readBody returns --content when the flag was given, otherwise all of stdin.
func readBody(content string, f *flag.Flag, stdin io.Reader) ([]byte, error) {
	if f != nil && content != "" {
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
