package main

import (
	"errors"
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
	var tk tickets
	fs := newFlagSet("task new", "nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]... [--content <text>]\n\ncreates tasks/inbox/<slug>.md; the body is read from stdin unless --content is given.\na slug already present in any status folder is refused.\nflags become the note's frontmatter, so the body must not start with one.\n--scheduled is the time the work is planned to happen, not a deadline")
	root := bindRoot(fs)
	content := fs.String("content", "", "note body; stdin is read when omitted")
	scheduled := fs.String("scheduled", "", "planned work time, RFC3339 with offset (2026-09-15T18:00:00+09:00)")
	fs.Var(&tk, "ticket", "related issue URL (https); repeatable")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	slug := fs.Arg(0)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	if *scheduled != "" {
		if _, err := time.Parse(time.RFC3339, *scheduled); err != nil {
			return fail(stderr, fmt.Errorf("--scheduled must be RFC3339 with an offset, e.g. 2026-09-15T18:00:00+09:00: %s", *scheduled), exitUsage)
		}
	}
	fm, err := frontmatter(field{"scheduled", *scheduled}, tk)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	body, err := readBody(*content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	if fm != "" && hasFrontmatter(body) {
		return fail(stderr, errors.New("body already starts with frontmatter; pass metadata as flags or drop them"), exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if have := s.FindTask(slug); have != "" {
		return fail(stderr, fmt.Errorf("%s: %w", have, store.ErrExists), exitFail)
	}
	doc := append([]byte(fm), body...)
	rel, err := s.Write(store.TaskPath("inbox", slug), doc)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "task", len(doc), stdout, stderr)
}

func runTaskMv(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task mv", "nabu task mv <slug> <"+strings.Join(store.TaskStatuses, "|")+">\n\nmoves tasks/<current>/<slug>.md to tasks/<status>/<slug>.md; the folder is the task's only status")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 2, 2, stdout, stderr); !ok {
		return code
	}
	slug, status := fs.Arg(0), fs.Arg(1)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	if !store.TaskStatus(status) {
		return fail(stderr, fmt.Errorf("status must be one of %s: %s", strings.Join(store.TaskStatuses, ", "), status), exitUsage)
	}
	s, err := openStore(*root)
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
	return finishMv(s, from, to, "task mv", stdout, stderr)
}

// hasFrontmatter reports whether a body opens with a YAML block of its own.
func hasFrontmatter(body []byte) bool {
	return strings.HasPrefix(strings.TrimLeft(string(body), "\n"), "---")
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
