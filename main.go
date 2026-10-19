// nabu keeps one markdown repository, the notebook, for tasks.
// Every task lives under tasks/ and every write is a git commit, so an agent
// (or a human) never edits the notebook by hand.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/config"
	"github.com/gitt510/nabu/internal/store"
)

const rootLong = `nabu keeps one markdown repository, the notebook, for tasks.

The notebook is a git repository; every change is committed as it is made,
and pushed to its upstream when auto_push is set in the config file.
Commands that change the notebook report the result as JSON; task read prints
raw text; task ls, task grep, and task validate print a table on a terminal,
plain text when piped, and JSON with --json.

The notebook comes from notebook in %s`

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
	// help lists commands in the order they are added, most used first
	cobra.EnableCommandSorting = false
	e := &env{stdin: stdin, stdout: stdout, stderr: stderr}
	root := parentCmd(e, &cobra.Command{
		Use:           "nabu",
		Short:         "keep tasks in one markdown repository",
		Long:          fmt.Sprintf(rootLong, config.Path()),
		SilenceErrors: true,
		SilenceUsage:  true,
	})
	root.AddCommand(taskCmd(e), pushCmd(e), publishCmd(e), initCmd(e))
	root.SetArgs(args)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	cmd, err := root.ExecuteC()
	var code exitCode
	switch {
	case err == nil:
		return exitOK
	case errors.As(err, &code):
		return int(code)
	}
	// cobra itself refused the invocation: an unknown flag or command, or the wrong number of args
	fmt.Fprintf(stderr, "%v\n\n%s", err, cmd.UsageString())
	return exitUsage
}

// env is what every command runs against: the process streams.
type env struct {
	stdin          io.Reader
	stdout, stderr io.Writer
}

// exitCode is returned by a command that has already reported its outcome,
// so run only has to exit with it.
type exitCode int

func (c exitCode) Error() string { return "exit " + strconv.Itoa(int(c)) }

// do adapts a command body that reports to stdout and stderr itself and
// returns an exit code.
func do(body func(args []string) int) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, args []string) error {
		if code := body(args); code != exitOK {
			return exitCode(code)
		}
		return nil
	}
}

// parentCmd makes cmd a group of subcommands: run alone it prints its usage
// to stderr and exits 2, and an unknown subcommand is an invocation error.
func parentCmd(e *env, cmd *cobra.Command) *cobra.Command {
	cmd.Args = cobra.ArbitraryArgs
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
		}
		fmt.Fprint(e.stderr, cmd.UsageString())
		return exitCode(exitUsage)
	}
	return cmd
}

// arg is the i-th positional arg, or "" when there are fewer.
func arg(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// errNoNotebook is a setup problem, not an operation failure, so it exits 2.
var errNoNotebook = errors.New("no notebook declared")

func setupMessage() string {
	return fmt.Sprintf("no notebook declared.\n\ncreate %s like:\n\n%s", config.Path(), config.Example)
}

// resolveNotebook reads the notebook from the config file.
func resolveNotebook() (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if cfg.Notebook == "" {
		return "", errNoNotebook
	}
	return cfg.Notebook, nil
}

// openStore opens the notebook the config file declares, with its auto_push.
func openStore() (*store.Store, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.Notebook == "" {
		return nil, errNoNotebook
	}
	s, err := store.Open(cfg.Notebook)
	if err != nil {
		return nil, err
	}
	s.AutoPush = cfg.AutoPush
	return s, nil
}

// pushAfter is the push field of a write's JSON document: "" when nothing
// was committed or auto_push is off, "ok", or "failed: <reason>". A failed
// push leaves the commit in place, so the write itself still succeeds.
func pushAfter(s *store.Store, committed bool) string {
	if !committed || !s.AutoPush {
		return ""
	}
	if _, err := s.Push(); err != nil {
		return "failed: " + err.Error()
	}
	return "ok"
}

func initCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "create the notebook declared in the config as a git repository",
		Long:  "creates the directory, runs git init -b main, and seeds README.md and .gitignore with a first commit; every step is skipped when already done",
		Args:  cobra.NoArgs,
		RunE: do(func(args []string) int {
			dir, err := resolveNotebook()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			r, err := store.Init(dir)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, r)

		}),
	}
	return cmd
}

func pushCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "git push the notebook to its upstream",
		Long:  "runs git push for the notebook's current branch; the branch must already have an upstream",
		Args:  cobra.NoArgs,
		RunE: do(func(args []string) int {
			s, err := openStore()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			upstream, err := s.Push()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, map[string]string{"action": "push", "upstream": upstream})

		}),
	}
	return cmd
}

func publishCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "push the notebook and run its publish workflow now",
		Long:  "pushes the notebook to its upstream, then starts the notebook repository's " + store.PublishWorkflow + " on the current branch through gh, so the viewer shows the pushed tasks without waiting for the schedule.\nprints the run's URL; the viewer updates when that run succeeds. a failed push starts nothing; a push that lands but a run that cannot start is reported as such.\nthe workflow must take workflow_dispatch, and gh must be signed in",
		Args:  cobra.NoArgs,
		RunE: do(func(args []string) int {
			s, err := openStore()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			upstream, err := s.Push()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			branch, run, err := s.Publish()
			if err != nil {
				return fail(e.stderr, fmt.Errorf("pushed to %s, but the publish run did not start: %w", upstream, err), exitFail)
			}
			return emit(e.stdout, map[string]string{"action": "publish", "upstream": upstream, "workflow": store.PublishWorkflow, "ref": branch, "run": run})

		}),
	}
	return cmd
}

// writeResult is the JSON document of every command that writes a file.
// Frontmatter is the task frontmatter as written, for the commands that
// change it (task new, task set).
type writeResult struct {
	Path        string `json:"path"`
	Action      string `json:"action"`
	Bytes       int    `json:"bytes"`
	Committed   bool   `json:"committed"`
	Push        string `json:"push,omitempty"`
	Frontmatter *meta  `json:"frontmatter,omitempty"`
}

// finishWrite commits the written file as "nabu: <action> <path>" and
// reports it, with fm when the command changed a task's frontmatter.
func finishWrite(s *store.Store, rel, action string, n int, fm *meta, stdout, stderr io.Writer) int {
	committed, err := s.Commit(fmt.Sprintf("nabu: %s %s", action, rel), rel)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, writeResult{Path: rel, Action: action, Bytes: n, Committed: committed, Push: pushAfter(s, committed), Frontmatter: fm})
}

// mvResult is the JSON document of every command that moves a file.
type mvResult struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Action    string `json:"action"`
	Committed bool   `json:"committed"`
	Push      string `json:"push,omitempty"`
}

// finishMv commits the move as "nabu: <action> <from> -> <to>" and
// reports it.
func finishMv(s *store.Store, from, to, action string, stdout, stderr io.Writer) int {
	committed, err := s.Commit(fmt.Sprintf("nabu: %s %s -> %s", action, from, to), from, to)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, mvResult{From: from, To: to, Action: action, Committed: committed, Push: pushAfter(s, committed)})
}

// rmResult is the JSON document of a command that deletes a file.
type rmResult struct {
	Path      string `json:"path"`
	Action    string `json:"action"`
	Committed bool   `json:"committed"`
	Push      string `json:"push,omitempty"`
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
	if errors.Is(err, errNoNotebook) {
		fmt.Fprint(stderr, setupMessage())
		return exitUsage
	}
	fmt.Fprintln(stderr, err)
	return code
}
