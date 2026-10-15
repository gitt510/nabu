package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/store"
)

// now is the clock task new stamps created with; tests pin it.
var now = time.Now

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
		taskNewCmd(e), taskReplaceCmd(e), taskSetCmd(e), taskMvCmd(e), taskRenameCmd(e), taskRmCmd(e),
	)
	return cmd
}

func taskNewCmd(e *env) *cobra.Command {
	var content string
	var mf *metaFlags
	cmd := &cobra.Command{
		Use:     "new <slug>",
		Short:   "create tasks/inbox/<slug>.md",
		Long:    "creates tasks/inbox/<slug>.md; the body is read from stdin unless --content is given.\na slug already present under tasks/ is refused.\nflags become the task's frontmatter, with created stamped as now; the body must not carry one of its own; the frontmatter as written is printed.\nthe body is \"# <title>\", then \"## For Human\" (every line at most 30 characters), a --- line, then \"## AI memo\" (free markdown).",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if !store.Slug(slug) {
				return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
			}
			m := meta{Created: now().Format(time.RFC3339)}
			if err := mf.apply(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if err := mf.missing(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			body, err := readTaskBody("inbox", content, e.stdin)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			s, err := openStore()
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
			return finishWrite(s, rel, "task", len(doc), &m, e.stdout, e.stderr)

		}),
	}
	cmd.Flags().StringVar(&content, "content", "", "task body; stdin is read when omitted")
	mf = addMetaFlags(cmd, false)
	cmd.Long += "\n" + mf.help()
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
			s, rel, m, _, code := openTask(arg(args, 0), e.stderr)
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
			return finishWrite(s, rel, "task replace", len(doc), nil, e.stdout, e.stderr)

		}),
	}
	cmd.Flags().StringVar(&content, "content", "", "new body; stdin is read when omitted")
	return cmd
}

func taskSetCmd(e *env) *cobra.Command {
	var mf *metaFlags
	cmd := &cobra.Command{
		Use:     "set <slug>",
		Short:   "change a task's frontmatter (area, project, scheduled, waiting, tickets, prs, links)",
		Long:    "rewrites the frontmatter of a task in any status folder and prints it as written; the body is kept as is (see task replace).",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			if !mf.given() {
				return fail(e.stderr, errors.New("nothing to set: pass at least one frontmatter flag"), exitUsage)
			}
			s, rel, m, body, code := openTask(arg(args, 0), e.stderr)
			if code != exitOK {
				return code
			}
			if err := mf.apply(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			doc := append([]byte(m.render()), body...)
			if _, err := s.Replace(rel, doc); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishWrite(s, rel, "task set", len(doc), &m, e.stdout, e.stderr)

		}),
	}
	mf = addMetaFlags(cmd, true)
	cmd.Long += "\n" + mf.help()
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
			s, err := openStore()
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
			s, err := openStore()
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

func taskRmCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm <slug>",
		Short:   "delete a task, whatever its status",
		Long:    "deletes tasks/<status>/<slug>.md and commits the deletion, so git history keeps the content.\nfor a task that should not have been filed (a duplicate, a mistake); work decided against goes to dropped (see task mv)",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if !store.Slug(slug) {
				return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
			}
			s, err := openStore()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			rel := s.FindTask(slug)
			if rel == "" {
				return fail(e.stderr, fmt.Errorf("no task %s under tasks/", slug), exitFail)
			}
			if _, err := s.Remove(rel); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			committed, err := s.Commit("nabu: task rm "+rel, rel)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, rmResult{Path: rel, Action: "task rm", Committed: committed, Push: pushAfter(s, committed)})

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
			s, rel, _, _, code := openTask(arg(args, 0), e.stderr)
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
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	Status      string `json:"status"`
	Title       string `json:"title"`
	Frontmatter meta   `json:"frontmatter"`
	Body        string `json:"body"`
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
				row.Frontmatter = m
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
			s, err := openStore()
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
			s, err := openStore()
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
func openTask(slug string, stderr io.Writer) (*store.Store, string, meta, []byte, int) {
	if !store.Slug(slug) {
		return nil, "", meta{}, nil, fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	s, err := openStore()
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
		Long:    "checks a task's body against the shape task new and task replace require: \"## For Human\" (every line at most 30 characters), ---, \"## AI memo\"; a task in dropped/ must also carry \"### Why dropped\" under For Human.\nthe frontmatter must carry an area (work or personal).\nwithout a slug every task is checked. prints one line per task (a colored table on a terminal); exit 1 when any task is out of shape.\nnothing is written",
		GroupID: "browse",
		Args:    cobra.MaximumNArgs(1),
		RunE: do(func(args []string) int {
			s, err := openStore()
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
				} else if m, body, err := splitMeta(doc); err != nil {
					v.Error = err.Error() + " (not written by nabu; fix it by hand)"
				} else if err := checkTaskShape(store.TaskStatusOf(rel), body); err != nil {
					v.Error = err.Error()
				} else if err := m.checkStored(); err != nil {
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
