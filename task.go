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

const taskUsage = `usage: nabu task <new|mv> [flags] [args]

  new <slug>           create tasks/inbox/<slug>.md
  mv  <slug> <status>  move a task to tasks/<status>/ (inbox, doing, done)

See "nabu task <command> -h".
`

// tickets collects repeated --ticket flags.
type tickets []string

func (t *tickets) String() string     { return strings.Join(*t, ",") }
func (t *tickets) Set(v string) error { *t = append(*t, v); return nil }

func runTask(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, taskUsage)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(stdout, taskUsage)
		return exitOK
	case "new":
		return runTaskNew(args[1:], stdin, stdout, stderr)
	case "mv":
		return runTaskMv(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown task command: %s\n\n%s", args[0], taskUsage)
	return exitUsage
}

func runTaskNew(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var c common
	var content, scheduled string
	var tk tickets
	var noCommit bool
	fs := newFlagSet("task new", "nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]... [--content <text>] [--no-commit] [--json]\n\ncreates tasks/inbox/<slug>.md; the body is read from stdin unless --content is given.\na slug already present in any status folder is refused.\nflags become the note's frontmatter, so the body must not start with one.\n--scheduled is the time the work is planned to happen, not a deadline", stderr)
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
	if scheduled != "" {
		if _, err := time.Parse(time.RFC3339, scheduled); err != nil {
			return fail(stderr, fmt.Errorf("--scheduled must be RFC3339 with an offset, e.g. 2026-09-15T18:00:00+09:00: %s", scheduled), exitUsage)
		}
	}
	fm, err := frontmatter(field{"scheduled", scheduled}, tk)
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
	if have := s.FindTask(slug); have != "" {
		return fail(stderr, fmt.Errorf("%s: %w", have, store.ErrExists), exitFail)
	}
	doc := append([]byte(fm), body...)
	rel, err := s.Write(store.TaskPath("inbox", slug), doc, false)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "task", len(doc), noCommit, c.json, stdout, stderr)
}

func runTaskMv(args []string, stdout, stderr io.Writer) int {
	var c common
	var noCommit bool
	fs := newFlagSet("task mv", "nabu task mv <slug> <"+strings.Join(store.TaskStatuses, "|")+"> [--no-commit] [--json]\n\nmoves tasks/<current>/<slug>.md to tasks/<status>/<slug>.md; the folder is the task's only status", stderr)
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
	slug, status := fs.Arg(0), fs.Arg(1)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	if !store.TaskStatus(status) {
		return fail(stderr, fmt.Errorf("status must be one of %s: %s", strings.Join(store.TaskStatuses, ", "), status), exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	from := s.FindTask(slug)
	if from == "" {
		return fail(stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
	}
	to := store.TaskPath(status, slug)
	if from == to {
		return fail(stderr, fmt.Errorf("%s is already %s", slug, status), exitFail)
	}
	if _, _, err := s.Move(from, to); err != nil {
		return fail(stderr, err, exitFail)
	}
	committed := false
	if !noCommit {
		committed, err = s.Commit(fmt.Sprintf("nabu: task mv %s -> %s", from, to), from, to)
		if err != nil {
			return fail(stderr, err, exitFail)
		}
	}
	if c.json {
		return emit(stdout, mvResult{From: from, To: to, Action: "task mv", Committed: committed})
	}
	state := "committed"
	if !committed {
		state = "not committed"
	}
	fmt.Fprintf(stdout, "task mv %s -> %s (%s)\n", from, to, state)
	return exitOK
}

// field is one scalar frontmatter entry; an empty value is left out.
type field struct{ key, value string }

// frontmatter renders the YAML block for the given field and tickets, or ""
// when both are empty so a plain note stays a plain note.
func frontmatter(f field, tk tickets) (string, error) {
	if f.value == "" && len(tk) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("---\n")
	if f.value != "" {
		fmt.Fprintf(&b, "%s: %q\n", f.key, f.value)
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
