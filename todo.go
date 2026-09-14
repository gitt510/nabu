package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/gitt510/nabu/internal/store"
)

const todoUsage = `usage: nabu todo new <slug> [flags]

See "nabu todo new -h".
`

// tickets collects repeated --ticket flags.
type tickets []string

func (t *tickets) String() string     { return strings.Join(*t, ",") }
func (t *tickets) Set(v string) error { *t = append(*t, v); return nil }

func runTodo(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, todoUsage)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(stdout, todoUsage)
		return exitOK
	case "new":
		return runTodoNew(args[1:], stdin, stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown todo command: %s\n\n%s", args[0], todoUsage)
	return exitUsage
}

func runTodoNew(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var c common
	var content, scheduled string
	var tk tickets
	var noCommit bool
	fs := newFlagSet("todo new", "nabu todo new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]... [--content <text>] [--no-commit] [--json]\n\ncreates tasks/<slug>.md; the body is read from stdin unless --content is given.\nflags become the note's frontmatter, so the body must not start with one.\n--scheduled is the time the work is planned to happen, not a deadline", stderr)
	c.bind(fs)
	fs.StringVar(&content, "content", "", "note body; stdin is read when omitted")
	fs.StringVar(&scheduled, "scheduled", "", "planned work time, RFC3339 with offset (2026-09-15T18:00:00+09:00)")
	fs.Var(&tk, "ticket", "related issue URL (https); repeatable")
	fs.BoolVar(&noCommit, "no-commit", false, "leave the change uncommitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	slug := fs.Arg(0)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	fm, err := frontmatter(scheduled, tk)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	body, err := readBody(content, fs.Lookup("content"), stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	if fm != "" && strings.HasPrefix(strings.TrimLeft(string(body), "\n"), "---") {
		return fail(stderr, errors.New("body already starts with frontmatter; pass metadata as flags or drop them"), exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	doc := append([]byte(fm), body...)
	rel, err := s.Write("tasks/"+slug+".md", doc, false)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "todo", len(doc), noCommit, c.json, stdout, stderr)
}

// frontmatter renders the YAML block for the given flags, or "" when none
// were given so a plain task stays a plain note.
func frontmatter(scheduled string, tk tickets) (string, error) {
	if scheduled == "" && len(tk) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("---\n")
	if scheduled != "" {
		if _, err := time.Parse(time.RFC3339, scheduled); err != nil {
			return "", fmt.Errorf("--scheduled must be RFC3339 with an offset, e.g. 2026-09-15T18:00:00+09:00: %s", scheduled)
		}
		fmt.Fprintf(&b, "scheduled: %q\n", scheduled)
	}
	if len(tk) > 0 {
		b.WriteString("tickets:\n")
		for _, t := range tk {
			u, err := url.Parse(t)
			if err != nil || u.Scheme != "https" || u.Host == "" {
				return "", fmt.Errorf("--ticket must be a full https URL: %s", t)
			}
			fmt.Fprintf(&b, "  - %q\n", t)
		}
	}
	b.WriteString("---\n")
	return b.String(), nil
}

var _ flag.Value = (*tickets)(nil)
