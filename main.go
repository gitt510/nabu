// nabu keeps one markdown repository for tasks and drafts. Every task lives
// under tasks/ and every write is a git commit, so an agent (or a human)
// never edits the root by hand. One file, CANVAS.md, is the exception: a
// draft the user edits by hand and the agent revises through nabu.
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

const rootLong = `nabu keeps one markdown repository for tasks and drafts.

The root is a git repository; every change is committed as it is made.
Commands that change the root report the result as JSON; task read, canvas
read, and canvas diff print raw text; task ls, task grep, and task validate
print a table on a terminal, plain text when piped, and JSON with --json.
CANVAS.md is ignored by git.

The root comes from --root <dir>, or else from root in %s`

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
		Short:         "keep tasks and drafts in one markdown repository",
		Long:          fmt.Sprintf(rootLong, config.Path()),
		SilenceErrors: true,
		SilenceUsage:  true,
	})
	root.PersistentFlags().StringVar(&e.root, "root", "", "repository root (overrides the config file)")
	root.AddCommand(taskCmd(e), canvasCmd(e), pushCmd(e), initCmd(e))
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

// env is what every command runs against: the --root flag and the process
// streams.
type env struct {
	root           string
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

// errNoRoot is a setup problem, not an operation failure, so it exits 2.
var errNoRoot = errors.New("no root declared")

func setupMessage() string {
	return fmt.Sprintf("no root declared.\n\ncreate %s like:\n\n%s\nor pass --root <dir>.\n", config.Path(), config.Example)
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

func initCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "create the root declared in the config as a git repository",
		Long:  "creates the directory, runs git init -b main, and seeds README.md and .gitignore with a first commit; every step is skipped when already done",
		Args:  cobra.NoArgs,
		RunE: do(func(args []string) int {
			dir, err := resolveRoot(e.root)
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
		Short: "git push the root to its upstream",
		Long:  "runs git push for the root's current branch; the branch must already have an upstream",
		Args:  cobra.NoArgs,
		RunE: do(func(args []string) int {
			s, err := openStore(e.root)
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
