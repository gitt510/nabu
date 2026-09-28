package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gitt510/nabu/internal/store"
)

const taskUsage = `usage: nabu task <new|replace|set|mv|read|ls|grep> [flags] [args]

  new     <slug>           create tasks/inbox/<slug>.md
  replace <slug>           replace a task's body, keeping its frontmatter
  set     <slug>           change a task's frontmatter (scheduled, waiting, tickets)
  mv      <slug> <status>  move a task to tasks/<status>/ (inbox, doing, done)
  read    <slug>           print a task, whatever its status
  ls      [status]         list tasks with status, title, and frontmatter
  grep    <query>          find lines containing query in tasks (case-insensitive)

tasks/ belongs to task; the note commands do not see it.
See "nabu task <command> -h".
`

// tickets collects repeated --ticket flags.
type tickets []string

func (t *tickets) String() string     { return strings.Join(*t, ",") }
func (t *tickets) Set(v string) error { *t = append(*t, v); return nil }

// meta is a task's frontmatter. scheduled is the time the work is planned
// to happen, waiting names who or what the next action waits on (while it
// is set the ball is with someone else), tickets are related issue URLs.
type meta struct {
	scheduled, waiting string
	tickets            []string
}

// render writes the YAML block, or "" when every field is empty so a plain
// task stays a plain note. Only this function writes task frontmatter.
func (m meta) render() string {
	if m.scheduled == "" && m.waiting == "" && len(m.tickets) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("---\n")
	if m.scheduled != "" {
		fmt.Fprintf(&b, "scheduled: %q\n", m.scheduled)
	}
	if m.waiting != "" {
		fmt.Fprintf(&b, "waiting: %q\n", m.waiting)
	}
	if len(m.tickets) > 0 {
		b.WriteString("tickets:\n")
		for _, t := range m.tickets {
			fmt.Fprintf(&b, "  - %q\n", t)
		}
	}
	b.WriteString("---\n")
	return b.String()
}

// splitMeta separates a task document into its frontmatter and body. It
// reads only the shape render writes; a block in any other shape is an
// error, because nabu never wrote it and cannot promise to keep it.
func splitMeta(doc []byte) (meta, []byte, error) {
	var m meta
	text := string(doc)
	if !strings.HasPrefix(text, "---\n") {
		return m, doc, nil
	}
	rest := text[len("---\n"):]
	inTickets := false
	for {
		line, after, ok := strings.Cut(rest, "\n")
		if !ok {
			return m, nil, errors.New("frontmatter block is not closed by ---")
		}
		rest = after
		if line == "---" {
			return m, []byte(rest), nil
		}
		if item, ok := strings.CutPrefix(line, "  - "); ok && inTickets {
			v, err := strconv.Unquote(item)
			if err != nil {
				return m, nil, fmt.Errorf("frontmatter: unreadable ticket: %s", line)
			}
			m.tickets = append(m.tickets, v)
			continue
		}
		inTickets = false
		key, raw, ok := strings.Cut(line, ": ")
		if !ok && line == "tickets:" {
			inTickets = true
			continue
		}
		v, err := strconv.Unquote(raw)
		if !ok || err != nil {
			return m, nil, fmt.Errorf("frontmatter: unreadable line: %s", line)
		}
		switch key {
		case "scheduled":
			m.scheduled = v
		case "waiting":
			m.waiting = v
		default:
			return m, nil, fmt.Errorf("frontmatter: unknown key %s", key)
		}
	}
}

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
	case "replace":
		return runTaskReplace(args[1:], stdin, stdout, stderr)
	case "set":
		return runTaskSet(args[1:], stdout, stderr)
	case "mv":
		return runTaskMv(args[1:], stdout, stderr)
	case "read":
		return runTaskRead(args[1:], stdout, stderr)
	case "ls":
		return runTaskLs(args[1:], stdout, stderr)
	case "grep":
		return runTaskGrep(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown task command: %s\n\n%s", args[0], taskUsage)
	return exitUsage
}

func runTaskNew(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var tk tickets
	fs := newFlagSet("task new", "nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]... [--content <text>]\n\ncreates tasks/inbox/<slug>.md; the body is read from stdin unless --content is given.\na slug already present under tasks/ is refused.\nflags become the note's frontmatter; the body must not carry one of its own.\n--scheduled is the time the work is planned to happen, not a deadline")
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
	m := meta{scheduled: *scheduled}
	if err := m.add(tk); err != nil {
		return fail(stderr, err, exitUsage)
	}
	if err := m.check(); err != nil {
		return fail(stderr, err, exitUsage)
	}
	body, err := readTaskBody(*content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if have := s.FindTask(slug); have != "" {
		return fail(stderr, fmt.Errorf("%s: %w", have, store.ErrExists), exitFail)
	}
	doc := append([]byte(m.render()), body...)
	rel, err := s.Write(store.TaskPath("inbox", slug), doc)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "task", len(doc), stdout, stderr)
}

func runTaskReplace(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := newFlagSet("task replace", "nabu task replace <slug> [--content <text>]\n\nreplaces the body of a task in any status folder; its frontmatter is kept as is (see task set).\nthe body is read from stdin unless --content is given and must not carry a frontmatter block")
	root := bindRoot(fs)
	content := fs.String("content", "", "new body; stdin is read when omitted")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	body, err := readTaskBody(*content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, rel, m, _, code := openTask(*root, fs.Arg(0), stderr)
	if code != exitOK {
		return code
	}
	doc := append([]byte(m.render()), body...)
	if _, err := s.Replace(rel, doc); err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "task replace", len(doc), stdout, stderr)
}

func runTaskSet(args []string, stdout, stderr io.Writer) int {
	var tk tickets
	fs := newFlagSet("task set", "nabu task set <slug> [--scheduled <RFC3339> | --clear-scheduled] [--waiting <text> | --clear-waiting] [--ticket <https-url>]... [--clear-tickets]\n\nrewrites the frontmatter of a task in any status folder; the body is kept as is (see task replace).\n--waiting names who or what the next action waits on: while it is set the ball is with someone else, so the task stays in doing/.\n--ticket adds to the list; --clear-tickets empties it first")
	root := bindRoot(fs)
	scheduled := fs.String("scheduled", "", "planned work time, RFC3339 with offset")
	clearScheduled := fs.Bool("clear-scheduled", false, "drop scheduled")
	waiting := fs.String("waiting", "", "who or what the next action waits on")
	clearWaiting := fs.Bool("clear-waiting", false, "drop waiting: the ball is back")
	fs.Var(&tk, "ticket", "related issue URL (https) to add; repeatable")
	clearTickets := fs.Bool("clear-tickets", false, "drop every ticket")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	if *scheduled == "" && !*clearScheduled && *waiting == "" && !*clearWaiting && len(tk) == 0 && !*clearTickets {
		return fail(stderr, errors.New("nothing to set: pass at least one frontmatter flag"), exitUsage)
	}
	if (*scheduled != "" && *clearScheduled) || (*waiting != "" && *clearWaiting) {
		return fail(stderr, errors.New("a value and its --clear flag cannot be given together"), exitUsage)
	}
	s, rel, m, body, code := openTask(*root, fs.Arg(0), stderr)
	if code != exitOK {
		return code
	}
	if *clearScheduled {
		m.scheduled = ""
	}
	if *scheduled != "" {
		m.scheduled = *scheduled
	}
	if *clearWaiting {
		m.waiting = ""
	}
	if *waiting != "" {
		m.waiting = *waiting
	}
	if *clearTickets {
		m.tickets = nil
	}
	if err := m.add(tk); err != nil {
		return fail(stderr, err, exitUsage)
	}
	if err := m.check(); err != nil {
		return fail(stderr, err, exitUsage)
	}
	doc := append([]byte(m.render()), body...)
	if _, err := s.Replace(rel, doc); err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "task set", len(doc), stdout, stderr)
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

func runTaskRead(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task read", "nabu task read <slug>\n\nprints the task found by slug in any status folder, frontmatter included")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	s, rel, _, _, code := openTask(*root, fs.Arg(0), stderr)
	if code != exitOK {
		return code
	}
	b, err := s.Read(rel)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	_, _ = stdout.Write(b)
	return exitOK
}

// taskEntry is one row of task ls: the note plus its frontmatter. Status
// is the folder, or "stray" for a tasks/<slug>.md outside every folder.
type taskEntry struct {
	Path      string   `json:"path"`
	Slug      string   `json:"slug"`
	Status    string   `json:"status"`
	Title     string   `json:"title"`
	Scheduled string   `json:"scheduled,omitempty"`
	Waiting   string   `json:"waiting,omitempty"`
	Tickets   []string `json:"tickets,omitempty"`
}

func runTaskLs(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task ls", "nabu task ls [status] [--json]\n\nlists every task, or those in one status folder; a task outside every folder shows as stray.\nprints status, slug, title, and waiting; --json adds path, scheduled, and tickets")
	root := bindRoot(fs)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if ok, code := parse(fs, args, 0, 1, stdout, stderr); !ok {
		return code
	}
	status := fs.Arg(0)
	if status != "" && !store.TaskStatus(status) {
		return fail(stderr, fmt.Errorf("status must be one of %s: %s", strings.Join(store.TaskStatuses, ", "), status), exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	entries, err := s.List(path.Join("tasks", status))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	rows := []taskEntry{}
	for _, e := range entries {
		rest := strings.TrimPrefix(e.Path, "tasks/")
		st, file, ok := strings.Cut(rest, "/")
		if !ok {
			st, file = "stray", rest
		}
		row := taskEntry{Path: e.Path, Slug: strings.TrimSuffix(file, ".md"), Status: st, Title: e.Title}
		if doc, err := s.Read(e.Path); err == nil {
			if m, _, err := splitMeta(doc); err == nil {
				row.Scheduled, row.Waiting, row.Tickets = m.scheduled, m.waiting, m.tickets
			}
		}
		rows = append(rows, row)
	}
	if *asJSON {
		return emit(stdout, rows)
	}
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for _, r := range rows {
		w := ""
		if r.Waiting != "" {
			w = "waiting: " + r.Waiting
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Status, r.Slug, r.Title, w)
	}
	_ = tw.Flush()
	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		if line != "" {
			fmt.Fprintln(stdout, strings.TrimRight(line, " "))
		}
	}
	return exitOK
}

func runTaskGrep(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task grep", "nabu task grep <query> [--json]\n\nfinds lines containing query under tasks/, case-insensitive")
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
	return printMatches(stdout, matches, *asJSON, isTaskPath)
}

// openTask finds the task by slug and splits it. On failure it has already
// reported to stderr and returns the exit code.
func openTask(root, slug string, stderr io.Writer) (*store.Store, string, meta, []byte, int) {
	if !store.Slug(slug) {
		return nil, "", meta{}, nil, fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	s, err := openStore(root)
	if err != nil {
		return nil, "", meta{}, nil, fail(stderr, err, exitFail)
	}
	rel := s.FindTask(slug)
	if rel == "" {
		return nil, "", meta{}, nil, fail(stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
	}
	doc, err := s.Read(rel)
	if err != nil {
		return nil, "", meta{}, nil, fail(stderr, err, exitFail)
	}
	m, body, err := splitMeta(doc)
	if err != nil {
		return nil, "", meta{}, nil, fail(stderr, fmt.Errorf("%s: %w (not written by nabu; fix it by hand)", rel, err), exitFail)
	}
	return s, rel, m, body, exitOK
}

// add appends tickets, each a full https URL, skipping ones already there.
func (m *meta) add(tk tickets) error {
	for _, t := range tk {
		u, err := url.Parse(t)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("--ticket must be a full https URL: %s", t)
		}
		if !slices.Contains(m.tickets, t) {
			m.tickets = append(m.tickets, t)
		}
	}
	return nil
}

// check validates the scalar fields set from flags.
func (m meta) check() error {
	if m.scheduled != "" {
		if _, err := time.Parse(time.RFC3339, m.scheduled); err != nil {
			return fmt.Errorf("--scheduled must be RFC3339 with an offset, e.g. 2026-09-15T18:00:00+09:00: %s", m.scheduled)
		}
	}
	if m.waiting != "" && strings.TrimSpace(m.waiting) == "" {
		return errors.New("--waiting must not be blank")
	}
	return nil
}

// readTaskBody reads a body and refuses one that opens with a frontmatter
// block: task metadata comes from flags only.
func readTaskBody(content string, stdin io.Reader) ([]byte, error) {
	body, err := readBody(content, stdin)
	if err != nil {
		return nil, err
	}
	if hasFrontmatter(body) {
		return nil, errors.New("body starts with a frontmatter block; task metadata comes from flags (see task set)")
	}
	return body, nil
}

// hasFrontmatter reports whether a body opens with a YAML block of its own.
func hasFrontmatter(body []byte) bool {
	return strings.HasPrefix(strings.TrimLeft(string(body), "\n"), "---")
}

// isTaskPath reports whether a note path is under tasks/, which only the
// task commands see.
func isTaskPath(p string) bool {
	clean := path.Clean(strings.ReplaceAll(p, "\\", "/"))
	return clean == "tasks" || strings.HasPrefix(clean, "tasks/")
}

// errTaskPath is what the note commands return for a path under tasks/.
var errTaskPath = errors.New("is under tasks/, which belongs to nabu task")

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
