package main

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/api"
)

// taskCmd is the task command group. Its help lists the commands by what
// they do: browse, read one task, create and update.
func taskCmd(e *env) *cobra.Command {
	cmd := parentCmd(e, &cobra.Command{
		Use:   "task",
		Short: "file and move tasks",
	})
	cmd.AddGroup(
		&cobra.Group{ID: "browse", Title: "Browse:"},
		&cobra.Group{ID: "read", Title: "Read content:"},
		&cobra.Group{ID: "update", Title: "Create and update:"},
	)
	cmd.AddCommand(
		taskLsCmd(e), taskGrepCmd(e),
		taskReadCmd(e),
		taskNewCmd(e), taskReplaceCmd(e), taskSetCmd(e), taskMvCmd(e), taskRenameCmd(e), taskRmCmd(e),
	)
	return cmd
}

var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// isSlug reports whether s is lowercase kebab-case, the shape of a slug and a project.
func isSlug(s string) bool { return slugPattern.MatchString(s) }

func checkSlug(slug string) error {
	if !isSlug(slug) {
		return fmt.Errorf("slug must be lowercase kebab-case: %s", slug)
	}
	return nil
}

func checkStatus(status string) error {
	if !slices.Contains(api.Statuses, status) {
		return fmt.Errorf("status must be one of %s: %s", strings.Join(api.Statuses, ", "), status)
	}
	return nil
}

// result is the JSON document of every command that changes a task.
// Frontmatter is the frontmatter as stored, for the commands that change it
// (task new, task set); From and To are the old and new status or slug.
type result struct {
	Action      string `json:"action"`
	Slug        string `json:"slug"`
	Status      string `json:"status,omitempty"`
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	Frontmatter *meta  `json:"frontmatter,omitempty"`
}

// readTaskBody reads a body and splits it into its title and section fields.
// It refuses a body that opens with a frontmatter block: task metadata comes
// from flags only.
func readTaskBody(content string, e *env) (map[string]any, error) {
	body, err := readBody(content, e.stdin)
	if err != nil {
		return nil, err
	}
	if hasFrontmatter(body) {
		return nil, errors.New("body starts with a frontmatter block; task metadata comes from flags (see task set)")
	}
	title, fields, err := parseBody(body)
	if err != nil {
		return nil, err
	}
	fields["title"] = title
	return fields, nil
}

func taskNewCmd(e *env) *cobra.Command {
	var content string
	var mf *metaFlags
	cmd := &cobra.Command{
		Use:     "new <slug>",
		Short:   "file a new task in inbox",
		Long:    "files a task in inbox; the body is read from stdin unless --content is given.\na slug already taken is refused.\nflags become the task's frontmatter, with created stamped by the API; the body must not carry one of its own; the frontmatter as stored is printed.\n" + shapeHelp,
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if err := checkSlug(slug); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			var m meta
			if err := mf.apply(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if err := mf.missing(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			fields, err := readTaskBody(content, e)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			fields["slug"] = slug
			// a new task leaves out what it does not have; null is for dropping a value
			for k, v := range m.apiFields() {
				if l, ok := v.([]string); v != nil && (!ok || len(l) > 0) {
					fields[k] = v
				}
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Create(fields)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			stored := metaOf(t)
			return emit(e.stdout, result{Action: "task new", Slug: t.Slug, Status: t.Status, Frontmatter: &stored})
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
		Long:    "replaces the title and every section of a task in any status; its frontmatter is kept as is (see task set).\nthe body is read from stdin unless --content is given and must not carry a frontmatter block.\n" + shapeHelp,
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if err := checkSlug(slug); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			fields, err := readTaskBody(content, e)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Patch(slug, fields)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, result{Action: "task replace", Slug: t.Slug, Status: t.Status})
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
		Long:    "changes the frontmatter of a task in any status and prints it as stored; the body is kept as is (see task replace).",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if err := checkSlug(slug); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if !mf.given() {
				return fail(e.stderr, errors.New("nothing to set: pass at least one frontmatter flag"), exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Get(slug)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			m := metaOf(t)
			if err := mf.apply(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if t, err = c.Patch(slug, m.apiFields()); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			stored := metaOf(t)
			return emit(e.stdout, result{Action: "task set", Slug: t.Slug, Status: t.Status, Frontmatter: &stored})
		}),
	}
	mf = addMetaFlags(cmd, true)
	cmd.Long += "\n" + mf.help()
	return cmd
}

func taskMvCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mv <slug> <status>",
		Short:   "move a task to a status (inbox, doing, done, dropped)",
		Long:    "sets the status of a task.\ndone is finished work; dropped is work decided against, and is refused until the body carries " + strconv.Quote(whyDropped.heading) + " (see task replace)",
		GroupID: "update",
		Args:    cobra.ExactArgs(2),
		RunE: do(func(args []string) int {
			slug, status := arg(args, 0), arg(args, 1)
			if err := checkSlug(slug); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if err := checkStatus(status); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Get(slug)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if t.Status == status {
				return fail(e.stderr, fmt.Errorf("%s is already %s", slug, status), exitFail)
			}
			if _, err := c.Patch(slug, map[string]any{"status": status}); err != nil {
				var refused *api.Error
				if errors.As(err, &refused) && status == "dropped" {
					err = fmt.Errorf("%w (write it with task replace first)", err)
				}
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, result{Action: "task mv", Slug: slug, From: t.Status, To: status})
		}),
	}
	return cmd
}

func taskRenameCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rename <slug> <new-slug>",
		Short:   "give a task a new slug, keeping its status and content",
		Long:    "changes the slug of a task; status, frontmatter, body, and history stay as they are.\na new slug already taken is refused",
		GroupID: "update",
		Args:    cobra.ExactArgs(2),
		RunE: do(func(args []string) int {
			slug, newSlug := arg(args, 0), arg(args, 1)
			for _, v := range []string{slug, newSlug} {
				if err := checkSlug(v); err != nil {
					return fail(e.stderr, err, exitUsage)
				}
			}
			if slug == newSlug {
				return fail(e.stderr, fmt.Errorf("new slug is the same as the old one: %s", slug), exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Patch(slug, map[string]any{"slug": newSlug})
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, result{Action: "task rename", Slug: t.Slug, Status: t.Status, From: slug, To: newSlug})
		}),
	}
	return cmd
}

func taskRmCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm <slug>",
		Short:   "delete a task, whatever its status",
		Long:    "deletes a task; the API keeps its history.\nfor a task that should not have been filed (a duplicate, a mistake); work decided against goes to dropped (see task mv)",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if err := checkSlug(slug); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if err := c.Delete(slug); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, result{Action: "task rm", Slug: slug})
		}),
	}
	return cmd
}

// document is a task as task read prints it: its frontmatter, then its body.
func document(t api.Task) string {
	m := metaOf(t)
	return m.render() + renderBody(t)
}

func taskReadCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "read <slug>",
		Short:   "print a task, whatever its status",
		Long:    "prints a task as markdown: its frontmatter block, then its body in the shape task new and task replace take",
		GroupID: "read",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if err := checkSlug(slug); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Get(slug)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			fmt.Fprint(e.stdout, document(t))
			return exitOK
		}),
	}
	return cmd
}

// taskEntry is one row of task ls: the task's frontmatter apart from its
// body, as task read prints them.
type taskEntry struct {
	Slug        string `json:"slug"`
	Status      string `json:"status"`
	Title       string `json:"title"`
	Frontmatter meta   `json:"frontmatter"`
	Updated     string `json:"updated"`
	Body        string `json:"body"`
}

// byWorkflow orders tasks by status in workflow order (inbox, doing, done,
// dropped). Tasks within a status compare equal, so a stable sort keeps the
// API's slug order.
func byWorkflow(a, b api.Task) int {
	return slices.Index(api.Statuses, a.Status) - slices.Index(api.Statuses, b.Status)
}

func taskLsCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "ls [status]",
		Short:   "list tasks in workflow order",
		Long:    "lists every task in workflow order (inbox, doing, done, dropped), or those in one status.\non a terminal prints a table of status, title, and created (its date), colored by status;\npiped, prints the same columns tab-separated without a header or color; --json adds slug, the frontmatter, updated, and the body",
		GroupID: "browse",
		Args:    cobra.MaximumNArgs(1),
		RunE: do(func(args []string) int {
			status := arg(args, 0)
			if status != "" {
				if err := checkStatus(status); err != nil {
					return fail(e.stderr, err, exitUsage)
				}
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			tasks, err := c.List(status, "")
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			slices.SortStableFunc(tasks, byWorkflow)
			if asJSON {
				rows := make([]taskEntry, len(tasks))
				for i, t := range tasks {
					rows[i] = taskEntry{Slug: t.Slug, Status: t.Status, Title: t.Title, Frontmatter: metaOf(t), Updated: t.Updated, Body: renderBody(t)}
				}
				return emit(e.stdout, rows)
			}
			if len(tasks) == 0 {
				return exitOK
			}
			width, tty := terminal(e.stdout)
			if !tty {
				for _, t := range tasks {
					fmt.Fprintf(e.stdout, "%s\t%s\t%s\n", t.Status, t.Title, createdDate(t.Created))
				}
				return exitOK
			}
			_, _ = lipgloss.Fprintln(e.stdout, taskTable(tasks, width))
			return exitOK
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON")
	return cmd
}

// taskTable renders task ls rows for a terminal width wide (0 for no limit).
func taskTable(tasks []api.Task, width int) string {
	cells := make([][]string, len(tasks))
	for i, t := range tasks {
		cells[i] = []string{t.Status, t.Title, createdDate(t.Created)}
	}
	return drawTable([]string{"STATUS", "TITLE", "CREATED"}, cells, width, func(row, col int) lipgloss.Style {
		switch col {
		case 0:
			return statusStyle(tasks[row].Status)
		case 2:
			return cellStyle.Faint(true)
		}
		return cellStyle
	})
}

// createdDate shows the local date created falls on, whatever offset the
// API stamped it with, or "-" when it cannot be read.
func createdDate(created string) string {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return "-"
	}
	return t.Local().Format(time.DateOnly) //nolint:gosmopolitan // the user reads the date in their own zone
}

// match is one line of task grep: a line of the task as task read prints it.
type match struct {
	Slug   string `json:"slug"`
	Status string `json:"status"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
}

func taskGrepCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "grep <query>",
		Short:   "find lines containing query in tasks (case-insensitive)",
		Long:    "finds lines containing query, case-insensitive, in every task as task read prints it, in workflow order.\non a terminal prints a table of status, task, line, and text with the query highlighted;\npiped, prints slug:line: text",
		GroupID: "browse",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			query := arg(args, 0)
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			tasks, err := c.List("", query)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			slices.SortStableFunc(tasks, byWorkflow)
			matches := []match{}
			q := strings.ToLower(query)
			for _, t := range tasks {
				for i, l := range strings.Split(document(t), "\n") {
					if strings.Contains(strings.ToLower(l), q) {
						matches = append(matches, match{Slug: t.Slug, Status: t.Status, Line: i + 1, Text: l})
					}
				}
			}
			if asJSON {
				return emit(e.stdout, matches)
			}
			width, tty := terminal(e.stdout)
			if !tty {
				for _, m := range matches {
					fmt.Fprintf(e.stdout, "%s:%d: %s\n", m.Slug, m.Line, m.Text)
				}
				return exitOK
			}
			if len(matches) > 0 {
				_, _ = lipgloss.Fprintln(e.stdout, grepTable(matches, query, width))
			}
			return exitOK
		}),
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the result as JSON")
	return cmd
}

// grepTable renders task grep matches for a terminal, with the query
// highlighted in each line.
func grepTable(matches []match, query string, width int) string {
	cells := make([][]string, len(matches))
	for i, m := range matches {
		cells[i] = []string{m.Status, m.Slug, strconv.Itoa(m.Line), highlight(m.Text, query)}
	}
	return drawTable([]string{"STATUS", "TASK", "LINE", "TEXT"}, cells, width, func(row, col int) lipgloss.Style {
		switch col {
		case 0:
			return statusStyle(matches[row].Status)
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
