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
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/store"
)

// meta is a task's frontmatter. created is when task new filed the task,
// stamped by nabu and never changed; scheduled is the time the work is
// planned to happen, waiting names who or what the next action waits on (while it
// is set the ball is with someone else). tags group tasks by topic, each
// a lowercase kebab-case word chosen by the user. The three URL lists are told
// apart by what the URL points at, so the choice is mechanical: tickets
// are work items (an issue, a Wrike task, a Zendesk request), prs are
// pull requests, links are everything else (a repo, an article, a post).
type meta struct {
	created, scheduled, waiting string
	tags, tickets, prs, links   []string
}

// now is the clock task new stamps created with; tests pin it.
var now = time.Now

// lists names the list fields in the order render writes them.
func (m *meta) lists() []struct {
	key string
	v   *[]string
} {
	return []struct {
		key string
		v   *[]string
	}{{"tags", &m.tags}, {"tickets", &m.tickets}, {"prs", &m.prs}, {"links", &m.links}}
}

// render writes the YAML block, or "" when every field is empty so a plain
// task stays a plain file. Only this function writes task frontmatter.
func (m meta) render() string {
	if m.created == "" && m.scheduled == "" && m.waiting == "" && len(m.tags) == 0 && len(m.tickets) == 0 && len(m.prs) == 0 && len(m.links) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("---\n")
	if m.created != "" {
		fmt.Fprintf(&b, "created: %q\n", m.created)
	}
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
		case "created":
			m.created = v
		case "scheduled":
			m.scheduled = v
		case "waiting":
			m.waiting = v
		default:
			return m, nil, fmt.Errorf("frontmatter: unknown key %s", key)
		}
	}
}

// taskCmd is the task command group. Its help lists the commands by what
// they do: browse and check, read one task, create and update.
func taskCmd(e *env) *cobra.Command {
	cmd := parentCmd(e, &cobra.Command{
		Use:   "task",
		Short: "file and move tasks under tasks/",
	})
	cmd.AddGroup(
		&cobra.Group{ID: "browse", Title: "Browse and check:"},
		&cobra.Group{ID: "read", Title: "Read content:"},
		&cobra.Group{ID: "update", Title: "Create and update:"},
	)
	cmd.AddCommand(
		taskLsCmd(e), taskGrepCmd(e), taskValidateCmd(e),
		taskReadCmd(e),
		taskNewCmd(e), taskReplaceCmd(e), taskSetCmd(e), taskMvCmd(e), taskRenameCmd(e),
	)
	return cmd
}

func taskNewCmd(e *env) *cobra.Command {
	var tags, tk, pr, ln []string
	var content string
	var scheduled string
	cmd := &cobra.Command{
		Use:     "new <slug>",
		Short:   "create tasks/inbox/<slug>.md",
		Long:    "creates tasks/inbox/<slug>.md; the body is read from stdin unless --content is given.\na slug already present under tasks/ is refused.\nflags become the task's frontmatter, with created stamped as now; the body must not carry one of its own.\nthe body is \"# <title>\", then \"## For Human\" (every line at most 30 characters), a --- line, then \"## AI memo\" (free markdown).\n--scheduled is the time the work is planned to happen, not a deadline: RFC3339 with an offset, or a date (YYYY-MM-DD) for the whole day.\n--tag groups the task by topic: a lowercase kebab-case word.\n" + urlFlagsHelp,
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if !store.Slug(slug) {
				return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
			}
			m := meta{created: now().Format(time.RFC3339), scheduled: scheduled}
			if err := m.add(tags, tk, pr, ln); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if err := m.check(); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			body, err := readTaskBody("inbox", content, e.stdin)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if have := s.FindTask(slug); have != "" {
				return fail(e.stderr, fmt.Errorf("%s: %w", have, store.ErrExists), exitFail)
			}
			doc := append([]byte(m.render()), body...)
			rel, err := s.Write(store.TaskPath("inbox", slug), doc)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishWrite(s, rel, "task", len(doc), e.stdout, e.stderr)

		}),
	}
	cmd.Flags().StringVar(&content, "content", "", "task body; stdin is read when omitted")
	cmd.Flags().StringVar(&scheduled, "scheduled", "", "planned work time, RFC3339 with offset (2026-09-15T18:00:00+09:00) or a date for the whole day (2026-09-15)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "topic tag, lowercase kebab-case; repeatable")
	cmd.Flags().StringArrayVar(&tk, "ticket", nil, "work item URL: issue, Wrike, Zendesk (`https-url`); repeatable")
	cmd.Flags().StringArrayVar(&pr, "pr", nil, "pull request URL (`https-url`); repeatable")
	cmd.Flags().StringArrayVar(&ln, "link", nil, "any other URL: repo, article, post (`https-url`); repeatable")
	return cmd
}

func taskReplaceCmd(e *env) *cobra.Command {
	var content string
	cmd := &cobra.Command{
		Use:     "replace <slug>",
		Short:   "replace a task's body, keeping its frontmatter",
		Long:    "replaces the body of a task in any status folder; its frontmatter is kept as is (see task set).\nthe body is read from stdin unless --content is given and must not carry a frontmatter block.\nit is in the same shape task new requires: \"## For Human\" (every line at most 30 characters), ---, \"## AI memo\"",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			s, rel, m, _, code := openTask(e.root, arg(args, 0), e.stderr)
			if code != exitOK {
				return code
			}
			body, err := readTaskBody(store.TaskStatusOf(rel), content, e.stdin)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			doc := append([]byte(m.render()), body...)
			if _, err := s.Replace(rel, doc); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishWrite(s, rel, "task replace", len(doc), e.stdout, e.stderr)

		}),
	}
	cmd.Flags().StringVar(&content, "content", "", "new body; stdin is read when omitted")
	return cmd
}

func taskSetCmd(e *env) *cobra.Command {
	var tags, tk, pr, ln []string
	var scheduled string
	var clearScheduled bool
	var waiting string
	var clearWaiting bool
	var clearTags bool
	var clearTickets bool
	var clearPrs bool
	var clearLinks bool
	cmd := &cobra.Command{
		Use:     "set <slug>",
		Short:   "change a task's frontmatter (scheduled, waiting, tags, tickets, prs, links)",
		Long:    "rewrites the frontmatter of a task in any status folder; the body is kept as is (see task replace).\n--waiting names who or what the next action waits on: while it is set the ball is with someone else, so the task stays in doing/.\n--tag, --ticket, --pr and --link add to their list; the matching --clear-* empties it first.\n" + urlFlagsHelp,
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			if scheduled == "" && !clearScheduled && waiting == "" && !clearWaiting && len(tags) == 0 && !clearTags && len(tk) == 0 && !clearTickets && len(pr) == 0 && !clearPrs && len(ln) == 0 && !clearLinks {
				return fail(e.stderr, errors.New("nothing to set: pass at least one frontmatter flag"), exitUsage)
			}
			if (scheduled != "" && clearScheduled) || (waiting != "" && clearWaiting) {
				return fail(e.stderr, errors.New("a value and its --clear flag cannot be given together"), exitUsage)
			}
			s, rel, m, body, code := openTask(e.root, arg(args, 0), e.stderr)
			if code != exitOK {
				return code
			}
			if clearScheduled {
				m.scheduled = ""
			}
			if scheduled != "" {
				m.scheduled = scheduled
			}
			if clearWaiting {
				m.waiting = ""
			}
			if waiting != "" {
				m.waiting = waiting
			}
			if clearTags {
				m.tags = nil
			}
			if clearTickets {
				m.tickets = nil
			}
			if clearPrs {
				m.prs = nil
			}
			if clearLinks {
				m.links = nil
			}
			if err := m.add(tags, tk, pr, ln); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if err := m.check(); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			doc := append([]byte(m.render()), body...)
			if _, err := s.Replace(rel, doc); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishWrite(s, rel, "task set", len(doc), e.stdout, e.stderr)

		}),
	}
	cmd.Flags().StringVar(&scheduled, "scheduled", "", "planned work time, RFC3339 with offset or a date for the whole day")
	cmd.Flags().BoolVar(&clearScheduled, "clear-scheduled", false, "drop scheduled")
	cmd.Flags().StringVar(&waiting, "waiting", "", "who or what the next action waits on")
	cmd.Flags().BoolVar(&clearWaiting, "clear-waiting", false, "drop waiting: the ball is back")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "topic tag to add, lowercase kebab-case; repeatable")
	cmd.Flags().BoolVar(&clearTags, "clear-tags", false, "drop every tag")
	cmd.Flags().StringArrayVar(&tk, "ticket", nil, "work item URL: issue, Wrike, Zendesk (`https-url`) to add; repeatable")
	cmd.Flags().BoolVar(&clearTickets, "clear-tickets", false, "drop every ticket")
	cmd.Flags().StringArrayVar(&pr, "pr", nil, "pull request URL (`https-url`) to add; repeatable")
	cmd.Flags().BoolVar(&clearPrs, "clear-prs", false, "drop every pr")
	cmd.Flags().StringArrayVar(&ln, "link", nil, "any other URL: repo, article, post (`https-url`) to add; repeatable")
	cmd.Flags().BoolVar(&clearLinks, "clear-links", false, "drop every link")
	return cmd
}

func taskMvCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mv <slug> <status>",
		Short:   "move a task to tasks/<status>/ (inbox, doing, done, dropped)",
		Long:    "moves tasks/<current>/<slug>.md to tasks/<status>/<slug>.md; the folder is the task's only status.\ndone is finished work; dropped is work decided against, and is refused until the body carries \"### Why dropped\" under For Human (see task replace)",
		GroupID: "update",
		Args:    cobra.ExactArgs(2),
		RunE: do(func(args []string) int {
			slug, status := arg(args, 0), arg(args, 1)
			if !store.Slug(slug) {
				return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
			}
			if !store.TaskStatus(status) {
				return fail(e.stderr, fmt.Errorf("status must be one of %s: %s", strings.Join(store.TaskStatuses, ", "), status), exitUsage)
			}
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			from := s.FindTask(slug)
			if from == "" {
				return fail(e.stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
			}
			to := store.TaskPath(status, slug)
			if from == to {
				return fail(e.stderr, fmt.Errorf("%s is already %s", slug, status), exitFail)
			}
			if status == "dropped" {
				doc, err := s.Read(from)
				if err != nil {
					return fail(e.stderr, err, exitFail)
				}
				_, body, err := splitMeta(doc)
				if err != nil {
					return fail(e.stderr, fmt.Errorf("%s: %w (not written by nabu; fix it by hand)", from, err), exitFail)
				}
				if err := checkTaskShape(status, body); err != nil {
					return fail(e.stderr, fmt.Errorf("%s: %w (write it with task replace first)", from, err), exitUsage)
				}
			}
			if _, _, err := s.Move(from, to); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishMv(s, from, to, "task mv", e.stdout, e.stderr)

		}),
	}
	return cmd
}

func taskRenameCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename <slug> <new-slug>",
		Short:   "give a task a new slug, keeping its status and content",
		Long:    "moves tasks/<status>/<slug>.md to tasks/<status>/<new-slug>.md; status, frontmatter, and body stay as they are.\na new slug already present under tasks/ is refused",
		GroupID: "update",
		Args:    cobra.ExactArgs(2),
		RunE: do(func(args []string) int {
			slug, newSlug := arg(args, 0), arg(args, 1)
			for _, v := range []string{slug, newSlug} {
				if !store.Slug(v) {
					return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", v), exitUsage)
				}
			}
			if slug == newSlug {
				return fail(e.stderr, fmt.Errorf("new slug is the same as the old one: %s", slug), exitUsage)
			}
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			from := s.FindTask(slug)
			if from == "" {
				return fail(e.stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
			}
			if taken := s.FindTask(newSlug); taken != "" {
				return fail(e.stderr, fmt.Errorf("%s: %w", taken, store.ErrExists), exitFail)
			}
			to := path.Dir(from) + "/" + newSlug + ".md"
			if _, _, err := s.Move(from, to); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishMv(s, from, to, "task rename", e.stdout, e.stderr)

		}),
	}
	return cmd
}

func taskReadCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "read <slug>",
		Short:   "print a task, whatever its status",
		Long:    "prints the task found by slug in any status folder, frontmatter included",
		GroupID: "read",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			s, rel, _, _, code := openTask(e.root, arg(args, 0), e.stderr)
			if code != exitOK {
				return code
			}
			b, err := s.Read(rel)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			_, _ = e.stdout.Write(b)
			return exitOK

		}),
	}
	return cmd
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
	Created   string   `json:"created,omitempty"`
	Scheduled string   `json:"scheduled,omitempty"`
	Waiting   string   `json:"waiting,omitempty"`
	Tags      []string `json:"tags,omitempty"`
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
		row := taskEntry{Path: e.Path, Slug: taskSlug(e.Path), Status: taskStatusOrStray(e.Path), Title: e.Title}
		if doc, err := s.Read(e.Path); err == nil {
			if m, body, err := splitMeta(doc); err == nil {
				row.Frontmatter = taskMeta{m.created, m.scheduled, m.waiting, m.tags, m.tickets, m.prs, m.links}
				row.Body = string(body)
			}
		}
		rows = append(rows, row)
	}
	slices.SortStableFunc(rows, func(a, b taskEntry) int { return byWorkflow(a.Path, b.Path) })
	return rows, nil
}

func taskLsCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "ls [status]",
		Short:   "list tasks in workflow order",
		Long:    "lists every task in workflow order (inbox, doing, done, dropped), or those in one status folder; a task outside every folder shows as stray, last.\non a terminal prints a table of status, title, and created (its date), colored by status;\npiped, prints the same columns tab-separated without a header or color; --json adds path, slug, the frontmatter, and the body",
		GroupID: "browse",
		Args:    cobra.MaximumNArgs(1),
		RunE: do(func(args []string) int {
			status := arg(args, 0)
			if status != "" && !store.TaskStatus(status) {
				return fail(e.stderr, fmt.Errorf("status must be one of %s: %s", strings.Join(store.TaskStatuses, ", "), status), exitUsage)
			}
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			rows, err := listTasks(s, status)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if asJSON {
				return emit(e.stdout, rows)
			}
			if len(rows) == 0 {
				return exitOK
			}
			width, tty := terminal(e.stdout)
			if !tty {
				for _, r := range rows {
					fmt.Fprintf(e.stdout, "%s\t%s\t%s\n", r.Status, r.Title, createdDate(r.Frontmatter.Created))
				}
				return exitOK
			}
			_, _ = lipgloss.Fprintln(e.stdout, taskTable(rows, width))
			return exitOK

		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON")
	return cmd
}

// taskTable renders task ls rows for a terminal width wide (0 for no limit).
func taskTable(rows []taskEntry, width int) string {
	cells := make([][]string, len(rows))
	for i, r := range rows {
		cells[i] = []string{r.Status, r.Title, createdDate(r.Frontmatter.Created)}
	}
	return drawTable([]string{"STATUS", "TITLE", "CREATED"}, cells, width, func(row, col int) lipgloss.Style {
		switch col {
		case 0:
			return statusStyle(rows[row].Status)
		case 2:
			return cellStyle.Faint(true)
		}
		return cellStyle
	})
}

// createdDate shows the date created was stamped on, in the offset it was
// stamped with, or "-" for a task filed before nabu stamped one.
func createdDate(created string) string {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return "-"
	}
	return t.Format(time.DateOnly)
}

func taskGrepCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "grep <query>",
		Short:   "find lines containing query in tasks (case-insensitive)",
		Long:    "finds lines containing query under tasks/, case-insensitive, in workflow order.\non a terminal prints a table of status, task, line, and text with the query highlighted;\npiped, prints path:line: text",
		GroupID: "browse",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			matches, err := s.Grep("tasks", arg(args, 0))
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			slices.SortStableFunc(matches, func(a, b store.Match) int { return byWorkflow(a.Path, b.Path) })
			if asJSON {
				return emit(e.stdout, matches)
			}
			width, tty := terminal(e.stdout)
			if !tty {
				for _, m := range matches {
					fmt.Fprintf(e.stdout, "%s:%d: %s\n", m.Path, m.Line, m.Text)
				}
				return exitOK
			}
			if len(matches) > 0 {
				_, _ = lipgloss.Fprintln(e.stdout, grepTable(matches, arg(args, 0), width))
			}
			return exitOK

		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON")
	return cmd
}

// grepTable renders task grep matches for a terminal, with the query
// highlighted in each line.
func grepTable(matches []store.Match, query string, width int) string {
	cells := make([][]string, len(matches))
	statuses := make([]string, len(matches))
	for i, m := range matches {
		statuses[i] = taskStatusOrStray(m.Path)
		cells[i] = []string{statuses[i], taskSlug(m.Path), strconv.Itoa(m.Line), highlight(m.Text, query)}
	}
	return drawTable([]string{"STATUS", "TASK", "LINE", "TEXT"}, cells, width, func(row, col int) lipgloss.Style {
		switch col {
		case 0:
			return statusStyle(statuses[row])
		case 2:
			return cellStyle.Faint(true).Align(lipgloss.Right)
		}
		return cellStyle
	})
}

// highlight marks every case-insensitive occurrence of query in text.
func highlight(text, query string) string {
	lower, q := strings.ToLower(text), strings.ToLower(query)
	// lowering can change byte lengths outside ASCII; then offsets would not line up
	if len(lower) != len(text) || q == "" {
		return text
	}
	mark := lipgloss.NewStyle().Bold(true).Underline(true)
	var b strings.Builder
	for {
		i := strings.Index(lower, q)
		if i < 0 {
			b.WriteString(text)
			return b.String()
		}
		b.WriteString(text[:i])
		b.WriteString(mark.Render(text[i : i+len(q)]))
		text, lower = text[i+len(q):], lower[i+len(q):]
	}
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

// urlFlagsHelp says how the three URL lists are told apart.
const urlFlagsHelp = "a URL goes by what it points at, whoever opened it and whatever it is to this task: --ticket is a work item (an issue, a Wrike task, a Zendesk request), --pr is a pull request, --link is everything else (a repo, an article, a post)"

// add appends the tags and URLs given by flag to their lists, in order and
// without duplicates; a tag must be lowercase kebab-case, a URL a full
// https URL.
func (m *meta) add(tags, tk, pr, ln []string) error {
	for _, t := range tags {
		if !store.Slug(t) {
			return fmt.Errorf("--tag must be lowercase kebab-case: %s", t)
		}
		if !slices.Contains(m.tags, t) {
			m.tags = append(m.tags, t)
		}
	}
	for _, in := range []struct {
		flag string
		vals []string
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
		// a date alone means the whole day; a time must carry its offset so the instant is not guessed
		_, errTime := time.Parse(time.RFC3339, m.scheduled)
		_, errDate := time.Parse(time.DateOnly, m.scheduled)
		if errTime != nil && errDate != nil {
			return fmt.Errorf("--scheduled must be RFC3339 with an offset (2026-09-15T18:00:00+09:00) or a date (2026-09-15): %s", m.scheduled)
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

func taskValidateCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "validate [slug]",
		Short:   "check one task, or every task, against the task shape",
		Long:    "checks a task's body against the shape task new and task replace require: \"## For Human\" (every line at most 30 characters), ---, \"## AI memo\"; a task in dropped/ must also carry \"### Why dropped\" under For Human.\nwithout a slug every task is checked. prints one line per task (a colored table on a terminal); exit 1 when any task is out of shape.\nnothing is written",
		GroupID: "browse",
		Args:    cobra.MaximumNArgs(1),
		RunE: do(func(args []string) int {
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			var paths []string
			if slug := arg(args, 0); slug != "" {
				if !store.Slug(slug) {
					return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
				}
				rel := s.FindTask(slug)
				if rel == "" {
					return fail(e.stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
				}
				paths = []string{rel}
			} else {
				entries, err := s.List("tasks")
				if err != nil {
					return fail(e.stderr, err, exitFail)
				}
				for _, e := range entries {
					paths = append(paths, e.Path)
				}
				slices.SortStableFunc(paths, byWorkflow)
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
			width, tty := terminal(e.stdout)
			switch {
			case asJSON:
				emit(e.stdout, rows)
			case tty && len(rows) > 0:
				_, _ = lipgloss.Fprintln(e.stdout, validateTable(rows, width))
			case !tty:
				for _, v := range rows {
					if v.Error == "" {
						fmt.Fprintf(e.stdout, "ok    %s\n", v.Path)
					} else {
						fmt.Fprintf(e.stdout, "bad   %s: %s\n", v.Path, v.Error)
					}
				}
			}
			if bad {
				return exitFail
			}
			return exitOK

		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON")
	return cmd
}

// validateTable renders task validate verdicts for a terminal.
func validateTable(rows []verdict, width int) string {
	cells := make([][]string, len(rows))
	statuses := make([]string, len(rows))
	for i, v := range rows {
		statuses[i] = taskStatusOrStray(v.Path)
		result := "ok"
		if v.Error != "" {
			result = "bad"
		}
		cells[i] = []string{result, statuses[i], taskSlug(v.Path), v.Error}
	}
	return drawTable([]string{"RESULT", "STATUS", "TASK", "PROBLEM"}, cells, width, func(row, col int) lipgloss.Style {
		switch col {
		case 0:
			if rows[row].Error != "" {
				return cellStyle.Foreground(lipgloss.Color("1")).Bold(true)
			}
			return cellStyle.Foreground(lipgloss.Color("2"))
		case 1:
			return statusStyle(statuses[row])
		}
		return cellStyle
	})
}
