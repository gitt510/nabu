// nabu files and moves tasks through the API of the nabu Worker, which keeps
// them in D1. An agent (or a human) never edits a task by hand: every read and
// write is a nabu command, signed in through Cloudflare Access.
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

	"github.com/gitt510/nabu/internal/api"
	"github.com/gitt510/nabu/internal/config"
)

const rootLong = `nabu files and moves tasks through the API of the nabu Worker.

Commands that change a task report the result as JSON; task read prints raw
text; task ls and task grep print a table on a terminal, plain text when piped,
and JSON with --json.

Every request is signed in through Cloudflare Access with the token
cloudflared holds: run cloudflared access login <base_url> once, and again when
the session expires.

The Worker comes from base_url in %s`

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
		Short:         "file and move tasks through the nabu API",
		Long:          fmt.Sprintf(rootLong, config.Path()),
		SilenceErrors: true,
		SilenceUsage:  true,
	})
	root.AddCommand(taskCmd(e))
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

// errNoURL is a setup problem, not an operation failure, so it exits 2.
var errNoURL = errors.New("no base_url declared")

func setupMessage() string {
	return fmt.Sprintf("no base_url declared.\n\ncreate %s like:\n\n%s\nthen sign in with cloudflared access login <base_url>\n", config.Path(), config.Example)
}

// client is the API client for the Worker the config file declares.
var client = func() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.BaseURL == "" {
		return nil, errNoURL
	}
	return api.New(cfg.BaseURL), nil
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

// fail reports err and returns its exit code: 2 for a setup problem or a
// request the API refused as malformed, otherwise code.
func fail(stderr io.Writer, err error, code int) int {
	var signedOut *api.SignedOutError
	var refused *api.Error
	switch {
	case errors.Is(err, errNoURL):
		fmt.Fprint(stderr, setupMessage())
		return exitUsage
	case errors.As(err, &signedOut):
		code = exitUsage
	case errors.As(err, &refused) && (refused.Status == 400 || refused.Status == 422):
		code = exitUsage
	}
	fmt.Fprintln(stderr, err)
	return code
}
