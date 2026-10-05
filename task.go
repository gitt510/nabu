package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gitt510/nabu/internal/store"
)

const taskUsage = `usage: nabu task <new|replace|set|mv|rename|read|ls|grep|validate> [flags] [args]

  new     <slug>           create tasks/inbox/<slug>.md
  replace <slug>           replace a task's body, keeping its frontmatter
  set     <slug>           change a task's frontmatter (scheduled, waiting, tickets, prs, links)
  mv      <slug> <status>  move a task to tasks/<status>/ (inbox, doing, done, dropped)
  rename  <slug> <new>     give a task a new slug, keeping its status and content
  read    <slug>           print a task, whatever its status
  ls      [status]         list tasks with status, title, and frontmatter
  grep    <query>          find lines containing query in tasks (case-insensitive)
  validate [slug]          check one task, or every task, against the task shape

See "nabu task <command> -h".
`

// tickets collects a repeated URL flag (--ticket, --pr, --link).
type tickets []string

func (t *tickets) String() string     { return strings.Join(*t, ",") }
func (t *tickets) Set(v string) error { *t = append(*t, v); return nil }

// meta is a task's frontmatter. scheduled is the time the work is planned
// to happen, waiting names who or what the next action waits on (while it
// is set the ball is with someone else). The three URL lists are told
// apart by what the URL points at, so the choice is mechanical: tickets
// are work items (an issue, a Wrike task, a Zendesk request), prs are
// pull requests, links are everything else (a repo, an article, a post).
type meta struct {
	scheduled, waiting  string
	tickets, prs, links []string
}

// lists names the URL lists in the order render writes them.
func (m *meta) lists() []struct {
	key string
	v   *[]string
} {
	return []struct {
		key string
		v   *[]string
	}{{"tickets", &m.tickets}, {"prs", &m.prs}, {"links", &m.links}}
}

// render writes the YAML block, or "" when every field is empty so a plain
// task stays a plain file. Only this function writes task frontmatter.
func (m meta) render() string {
	if m.scheduled == "" && m.waiting == "" && len(m.tickets) == 0 && len(m.prs) == 0 && len(m.links) == 0 {
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
	for _, l := range m.lists() {
		if len(*l.v) == 0 {
			continue
		}
		b.WriteString(l.key + ":\n")
		for _, t := range *l.v {
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
	var inList *[]string
	for {
		line, after, ok := strings.Cut(rest, "\n")
		if !ok {
			return m, nil, errors.New("frontmatter block is not closed by ---")
		}
		rest = after
		if line == "---" {
			return m, []byte(rest), nil
		}
		if item, ok := strings.CutPrefix(line, "  - "); ok && inList != nil {
			v, err := strconv.Unquote(item)
			if err != nil {
				return m, nil, fmt.Errorf("frontmatter: unreadable list item: %s", line)
			}
			*inList = append(*inList, v)
			continue
		}
		inList = nil
		key, raw, ok := strings.Cut(line, ": ")
		if !ok {
			for _, l := range m.lists() {
				if line == l.key+":" {
					inList = l.v
					break
				}
			}
			if inList != nil {
				continue
			}
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
	case "rename":
		return runTaskRename(args[1:], stdout, stderr)
	case "read":
		return runTaskRead(args[1:], stdout, stderr)
	case "ls":
		return runTaskLs(args[1:], stdout, stderr)
	case "grep":
		return runTaskGrep(args[1:], stdout, stderr)
	case "validate":
		return runTaskValidate(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown task command: %s\n\n%s", args[0], taskUsage)
	return exitUsage
}

func runTaskNew(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var tk, pr, ln tickets
	fs := newFlagSet("task new", "nabu task new <slug> [--scheduled <RFC3339>] [--ticket <https-url>]... [--pr <https-url>]... [--link <https-url>]... [--content <text>]\n\ncreates tasks/inbox/<slug>.md; the body is read from stdin unless --content is given.\na slug already present under tasks/ is refused.\nflags become the task's frontmatter; the body must not carry one of its own.\nthe body is \"# <title>\", then \"## For Human\" (every line at most 30 characters), a --- line, then \"## AI memo\" (free markdown).\n--scheduled is the time the work is planned to happen, not a deadline.\n"+urlFlagsHelp)
	root := bindRoot(fs)
	content := fs.String("content", "", "task body; stdin is read when omitted")
	scheduled := fs.String("scheduled", "", "planned work time, RFC3339 with offset (2026-09-15T18:00:00+09:00)")
	fs.Var(&tk, "ticket", "work item URL: issue, Wrike, Zendesk (https); repeatable")
	fs.Var(&pr, "pr", "pull request URL (https); repeatable")
	fs.Var(&ln, "link", "any other URL: repo, article, post (https); repeatable")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	slug := fs.Arg(0)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	m := meta{scheduled: *scheduled}
	if err := m.add(tk, pr, ln); err != nil {
		return fail(stderr, err, exitUsage)
	}
	if err := m.check(); err != nil {
		return fail(stderr, err, exitUsage)
	}
	body, err := readTaskBody("inbox", *content, stdin)
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
	fs := newFlagSet("task replace", "nabu task replace <slug> [--content <text>]\n\nreplaces the body of a task in any status folder; its frontmatter is kept as is (see task set).\nthe body is read from stdin unless --content is given and must not carry a frontmatter block.\nit is in the same shape task new requires: \"## For Human\" (every line at most 30 characters), ---, \"## AI memo\"")
	root := bindRoot(fs)
	content := fs.String("content", "", "new body; stdin is read when omitted")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	s, rel, m, _, code := openTask(*root, fs.Arg(0), stderr)
	if code != exitOK {
		return code
	}
	body, err := readTaskBody(store.TaskStatusOf(rel), *content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	doc := append([]byte(m.render()), body...)
	if _, err := s.Replace(rel, doc); err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "task replace", len(doc), stdout, stderr)
}

func runTaskSet(args []string, stdout, stderr io.Writer) int {
	var tk, pr, ln tickets
	fs := newFlagSet("task set", "nabu task set <slug> [--scheduled <RFC3339> | --clear-scheduled] [--waiting <text> | --clear-waiting] [--ticket <https-url>]... [--clear-tickets] [--pr <https-url>]... [--clear-prs] [--link <https-url>]... [--clear-links]\n\nrewrites the frontmatter of a task in any status folder; the body is kept as is (see task replace).\n--waiting names who or what the next action waits on: while it is set the ball is with someone else, so the task stays in doing/.\n--ticket, --pr and --link add to their list; the matching --clear-* empties it first.\n"+urlFlagsHelp)
	root := bindRoot(fs)
	scheduled := fs.String("scheduled", "", "planned work time, RFC3339 with offset")
	clearScheduled := fs.Bool("clear-scheduled", false, "drop scheduled")
	waiting := fs.String("waiting", "", "who or what the next action waits on")
	clearWaiting := fs.Bool("clear-waiting", false, "drop waiting: the ball is back")
	fs.Var(&tk, "ticket", "work item URL: issue, Wrike, Zendesk (https) to add; repeatable")
	clearTickets := fs.Bool("clear-tickets", false, "drop every ticket")
	fs.Var(&pr, "pr", "pull request URL (https) to add; repeatable")
	clearPrs := fs.Bool("clear-prs", false, "drop every pr")
	fs.Var(&ln, "link", "any other URL: repo, article, post (https) to add; repeatable")
	clearLinks := fs.Bool("clear-links", false, "drop every link")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	if *scheduled == "" && !*clearScheduled && *waiting == "" && !*clearWaiting && len(tk) == 0 && !*clearTickets && len(pr) == 0 && !*clearPrs && len(ln) == 0 && !*clearLinks {
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
	if *clearPrs {
		m.prs = nil
	}
	if *clearLinks {
		m.links = nil
	}
	if err := m.add(tk, pr, ln); err != nil {
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
	fs := newFlagSet("task mv", "nabu task mv <slug> <"+strings.Join(store.TaskStatuses, "|")+">\n\nmoves tasks/<current>/<slug>.md to tasks/<status>/<slug>.md; the folder is the task's only status.\ndone is finished work; dropped is work decided against, and is refused until the body carries \"### Why dropped\" under For Human (see task replace)")
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
	if status == "dropped" {
		doc, err := s.Read(from)
		if err != nil {
			return fail(stderr, err, exitFail)
		}
		_, body, err := splitMeta(doc)
		if err != nil {
			return fail(stderr, fmt.Errorf("%s: %w (not written by nabu; fix it by hand)", from, err), exitFail)
		}
		if err := checkTaskShape(status, body); err != nil {
			return fail(stderr, fmt.Errorf("%s: %w (write it with task replace first)", from, err), exitUsage)
		}
	}
	if _, _, err := s.Move(from, to); err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishMv(s, from, to, "task mv", stdout, stderr)
}

func runTaskRename(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task rename", "nabu task rename <slug> <new-slug>\n\nmoves tasks/<status>/<slug>.md to tasks/<status>/<new-slug>.md; status, frontmatter, and body stay as they are.\na new slug already present under tasks/ is refused")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 2, 2, stdout, stderr); !ok {
		return code
	}
	slug, newSlug := fs.Arg(0), fs.Arg(1)
	for _, v := range []string{slug, newSlug} {
		if !store.Slug(v) {
			return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", v), exitUsage)
		}
	}
	if slug == newSlug {
		return fail(stderr, fmt.Errorf("new slug is the same as the old one: %s", slug), exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	from := s.FindTask(slug)
	if from == "" {
		return fail(stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
	}
	if taken := s.FindTask(newSlug); taken != "" {
		return fail(stderr, fmt.Errorf("%s: %w", taken, store.ErrExists), exitFail)
	}
	to := path.Dir(from) + "/" + newSlug + ".md"
	if _, _, err := s.Move(from, to); err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishMv(s, from, to, "task rename", stdout, stderr)
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

// taskEntry is one row of task ls. Path, slug, status, and title are the
// values nabu derives by its own rules (status is the folder, or "stray"
// for a tasks/<slug>.md outside every folder); frontmatter and body are
// the file as written, so a consumer never has to know those rules.
type taskEntry struct {
	Path        string   `json:"path"`
	Slug        string   `json:"slug"`
	Status      string   `json:"status"`
	Title       string   `json:"title"`
	Frontmatter taskMeta `json:"frontmatter"`
	Body        string   `json:"body"`
}

// taskMeta is meta as task ls --json prints it.
type taskMeta struct {
	Scheduled string   `json:"scheduled,omitempty"`
	Waiting   string   `json:"waiting,omitempty"`
	Tickets   []string `json:"tickets,omitempty"`
	Prs       []string `json:"prs,omitempty"`
	Links     []string `json:"links,omitempty"`
}

// listTasks reads every task, or those in one status folder, as ls rows.
func listTasks(s *store.Store, status string) ([]taskEntry, error) {
	entries, err := s.List(path.Join("tasks", status))
	if err != nil {
		// a status folder appears with the first task moved into it;
		// until then the status is simply empty
		if status != "" && errors.Is(err, fs.ErrNotExist) {
			return []taskEntry{}, nil
		}
		return nil, err
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
			if m, body, err := splitMeta(doc); err == nil {
				row.Frontmatter = taskMeta{m.scheduled, m.waiting, m.tickets, m.prs, m.links}
				row.Body = string(body)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func runTaskLs(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task ls", "nabu task ls [status] [--json]\n\nlists every task, or those in one status folder; a task outside every folder shows as stray.\nprints status, slug, title, and waiting; --json adds path, the frontmatter, and the body")
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
	rows, err := listTasks(s, status)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if *asJSON {
		return emit(stdout, rows)
	}
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for _, r := range rows {
		w := ""
		if r.Frontmatter.Waiting != "" {
			w = "waiting: " + r.Frontmatter.Waiting
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
	matches, err := s.Grep("tasks", fs.Arg(0))
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if *asJSON {
		return emit(stdout, matches)
	}
	for _, m := range matches {
		fmt.Fprintf(stdout, "%s:%d: %s\n", m.Path, m.Line, m.Text)
	}
	return exitOK
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
// urlFlagsHelp says how the three URL lists are told apart.
const urlFlagsHelp = "a URL goes by what it points at, whoever opened it and whatever it is to this task: --ticket is a work item (an issue, a Wrike task, a Zendesk request), --pr is a pull request, --link is everything else (a repo, an article, a post)"

// add appends the URLs given by flag to their lists, in order and without
// duplicates; each must be a full https URL.
func (m *meta) add(tk, pr, ln tickets) error {
	for _, in := range []struct {
		flag string
		vals tickets
		dst  *[]string
	}{{"ticket", tk, &m.tickets}, {"pr", pr, &m.prs}, {"link", ln, &m.links}} {
		for _, t := range in.vals {
			u, err := url.Parse(t)
			if err != nil || u.Scheme != "https" || u.Host == "" {
				return fmt.Errorf("--%s must be a full https URL: %s", in.flag, t)
			}
			if !slices.Contains(*in.dst, t) {
				*in.dst = append(*in.dst, t)
			}
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
// block (task metadata comes from flags only) or that is not in the task
// shape for the status it will sit in (see checkTaskShape).
func readTaskBody(status, content string, stdin io.Reader) ([]byte, error) {
	body, err := readBody(content, stdin)
	if err != nil {
		return nil, err
	}
	if hasFrontmatter(body) {
		return nil, errors.New("body starts with a frontmatter block; task metadata comes from flags (see task set)")
	}
	if err := checkTaskShape(status, body); err != nil {
		return nil, err
	}
	return body, nil
}

// verdict is one task validate result; Error is empty when the task is in shape.
type verdict struct {
	Path  string `json:"path"`
	Error string `json:"error,omitempty"`
}

func runTaskValidate(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("task validate", "nabu task validate [slug] [--json]\n\nchecks a task's body against the shape task new and task replace require: \"## For Human\" (every line at most 30 characters), ---, \"## AI memo\"; a task in dropped/ must also carry \"### Why dropped\" under For Human.\nwithout a slug every task is checked. prints one line per task; exit 1 when any task is out of shape.\nnothing is written")
	root := bindRoot(fs)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if ok, code := parse(fs, args, 0, 1, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	var paths []string
	if slug := fs.Arg(0); slug != "" {
		if !store.Slug(slug) {
			return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
		}
		rel := s.FindTask(slug)
		if rel == "" {
			return fail(stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
		}
		paths = []string{rel}
	} else {
		entries, err := s.List("tasks")
		if err != nil {
			return fail(stderr, err, exitFail)
		}
		for _, e := range entries {
			paths = append(paths, e.Path)
		}
	}
	rows := []verdict{}
	bad := false
	for _, rel := range paths {
		v := verdict{Path: rel}
		doc, err := s.Read(rel)
		if err != nil {
			v.Error = err.Error()
		} else if _, body, err := splitMeta(doc); err != nil {
			v.Error = err.Error() + " (not written by nabu; fix it by hand)"
		} else if err := checkTaskShape(store.TaskStatusOf(rel), body); err != nil {
			v.Error = err.Error()
		}
		if v.Error != "" {
			bad = true
		}
		rows = append(rows, v)
	}
	if *asJSON {
		emit(stdout, rows)
	} else {
		for _, v := range rows {
			if v.Error == "" {
				fmt.Fprintf(stdout, "ok    %s\n", v.Path)
			} else {
				fmt.Fprintf(stdout, "bad   %s: %s\n", v.Path, v.Error)
			}
		}
	}
	if bad {
		return exitFail
	}
	return exitOK
}
