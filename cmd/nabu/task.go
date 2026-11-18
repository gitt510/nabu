package main

import (
	"encoding/json"
	"errors"
	"fmt"
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
		taskReadCmd(e), taskSchemaCmd(e),
		taskNewCmd(e), taskReplaceCmd(e), taskSetCmd(e), taskDoneCmd(e), taskCancelCmd(e), taskReopenCmd(e), taskRmCmd(e),
		taskItemCmd(e),
	)
	return cmd
}

func checkStatus(status api.Status) error {
	if !status.Valid() {
		return fmt.Errorf("no such status: %s", status)
	}
	return nil
}

// taskID reads a task's id as people write it: 6, or #6 as the viewer shows it.
func taskID(ref string) (int, error) {
	id, err := strconv.Atoi(strings.TrimPrefix(ref, "#"))
	if err != nil || id < 1 {
		return 0, fmt.Errorf("not a task id: %s (pass the number the viewer shows, as 6 or '#6')", ref)
	}
	return id, nil
}

// withTask runs f on the id args[0] names, failing with exitUsage when it is not one.
func withTask(e *env, f func(id int, rest []string) int) func(*cobra.Command, []string) error {
	return do(func(args []string) int {
		id, err := taskID(arg(args, 0))
		if err != nil {
			return fail(e.stderr, err, exitUsage)
		}
		return f(id, args[1:])
	})
}

// result is the JSON document of every command that changes a task.
// Frontmatter is the frontmatter as stored, for the commands that change it
// (task new, task set).
type result struct {
	Action      string     `json:"action"`
	ID          int        `json:"id"`
	Status      api.Status `json:"status,omitempty"`
	Frontmatter *meta      `json:"frontmatter,omitempty"`
	Items       []string   `json:"items,omitempty"`
}

// readTaskBody reads a body: a JSON object of a task's title, sent to the API as it is.
func readTaskBody(content string, e *env) (map[string]any, error) {
	body, err := readBody(content, e.stdin)
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return nil, errors.New("the body must be a JSON object of the task's title (see task schema)")
	}
	return fields, nil
}

// bodyHelp is the body as the help states it; every command that takes one points here.
const bodyHelp = "the body is a JSON object of the task's title, as task schema gives it; the API refuses one that breaks the task contract, and says why."

func taskNewCmd(e *env) *cobra.Command {
	var content string
	var mf *metaFlags
	cmd := &cobra.Command{
		Use:     "new",
		Short:   "file a new todo task",
		Long:    "files a todo task and prints the id the API gave it; the body is read from stdin unless --content is given.\nflags become the task's frontmatter, with created stamped by the API; the frontmatter as stored is printed.\n" + bodyHelp,
		GroupID: "update",
		Args:    cobra.NoArgs,
		RunE: do(func([]string) int {
			var m meta
			if err := mf.apply(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			fields, err := readTaskBody(content, e)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
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
			return emit(e.stdout, result{Action: "task new", ID: t.Id, Status: t.Status, Frontmatter: &stored})
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
		Use:     "replace <id>",
		Short:   "replace a task's body, keeping its frontmatter",
		Long:    "replaces the title the body gives, in a task of any status; its frontmatter is kept as is (see task set).\nthe body is read from stdin unless --content is given.\n" + bodyHelp,
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: withTask(e, func(id int, _ []string) int {
			fields, err := readTaskBody(content, e)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Patch(id, fields)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, result{Action: "task replace", ID: t.Id, Status: t.Status})
		}),
	}
	cmd.Flags().StringVar(&content, "content", "", "new body; stdin is read when omitted")
	return cmd
}

func taskSetCmd(e *env) *cobra.Command {
	var mf *metaFlags
	cmd := &cobra.Command{
		Use:     "set <id>",
		Short:   "change a task's frontmatter (area, project, scheduled, waiting, tickets, prs, links)",
		Long:    "changes the frontmatter of a task in any status and prints it as stored; the body is kept as is (see task replace).",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: withTask(e, func(id int, _ []string) int {
			if !mf.given() {
				return fail(e.stderr, errors.New("nothing to set: pass at least one frontmatter flag"), exitUsage)
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Get(id)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			m := metaOf(t)
			if err := mf.apply(&m); err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			if t, err = c.Patch(id, m.apiFields()); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			stored := metaOf(t)
			return emit(e.stdout, result{Action: "task set", ID: t.Id, Status: t.Status, Frontmatter: &stored})
		}),
	}
	mf = addMetaFlags(cmd, true)
	cmd.Long += "\n" + mf.help()
	return cmd
}

// move sets a task's status and why_canceled, refusing a task that has them
// already.
func move(e *env, action string, id int, status api.Status, why []string) int {
	c, err := client()
	if err != nil {
		return fail(e.stderr, err, exitFail)
	}
	t, err := c.Get(id)
	if err != nil {
		return fail(e.stderr, err, exitFail)
	}
	if t.Status == status && slices.Equal(t.WhyCanceled, why) {
		return fail(e.stderr, fmt.Errorf("#%d is already %s", id, status), exitFail)
	}
	if t, err = c.Patch(id, map[string]any{"status": status, "why_canceled": why}); err != nil {
		return fail(e.stderr, err, exitFail)
	}
	return emit(e.stdout, result{Action: action, ID: t.Id, Status: t.Status})
}

func taskDoneCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:     "done <id>",
		Short:   "mark a task done",
		Long:    "marks a task done; a canceled task becomes done and loses its why_canceled",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: withTask(e, func(id int, _ []string) int {
			return move(e, "task done", id, api.Done, []string{})
		}),
	}
}

func taskCancelCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:     "cancel <id> <why>...",
		Short:   "cancel a task, saying why",
		Long:    "marks a task canceled: work decided against. each <why> is one line of why_canceled, the reason the task keeps; a canceled task given new lines keeps those instead",
		GroupID: "update",
		Args:    cobra.MinimumNArgs(2),
		RunE: withTask(e, func(id int, why []string) int {
			return move(e, "task cancel", id, api.Canceled, why)
		}),
	}
}

func taskReopenCmd(e *env) *cobra.Command {
	return &cobra.Command{
		Use:     "reopen <id>",
		Short:   "make a done or canceled task todo again",
		Long:    "makes a done or canceled task todo again; a canceled task loses its why_canceled",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: withTask(e, func(id int, _ []string) int {
			return move(e, "task reopen", id, api.Todo, []string{})
		}),
	}
}

func taskRmCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rm <id>",
		Short:   "delete a task, whatever its status",
		Long:    "deletes a task; the API keeps its history.\nfor a task that should not have been filed (a duplicate, a mistake); work decided against is canceled (see task cancel)",
		GroupID: "update",
		Args:    cobra.ExactArgs(1),
		RunE: withTask(e, func(id int, _ []string) int {
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if err := c.Delete(id); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, result{Action: "task rm", ID: id})
		}),
	}
	return cmd
}

// taskItemCmd is the group of commands on a task's checklist.
func taskItemCmd(e *env) *cobra.Command {
	cmd := parentCmd(e, &cobra.Command{
		Use:     "item",
		Short:   "change a task's checklist",
		GroupID: "update",
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "add <id> <line>...",
		Short: "add lines to a task's checklist",
		Long:  "adds each <line>, not done, last in the checklist of a task in any status, in the order given",
		Args:  cobra.MinimumNArgs(2),
		RunE: withTask(e, func(id int, lines []string) int {
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Get(id)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			for _, line := range lines {
				if _, err := c.AddItem(t.Id, line); err != nil {
					return fail(e.stderr, err, exitFail)
				}
			}
			return emit(e.stdout, result{Action: "task item add", ID: t.Id, Status: t.Status, Items: lines})
		}),
	})
	return cmd
}

func taskSchemaCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schema",
		Short:   "print the rules a task must meet",
		Long:    "prints the task contract as JSON Schema, as the API serves it: create for task new, patch for the rest.\nthe API refuses a write that breaks it",
		GroupID: "read",
		Args:    cobra.NoArgs,
		RunE: do(func([]string) int {
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			s, err := c.Schema()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, s)
		}),
	}
	return cmd
}

// document is a task as task read prints it: its frontmatter, then its title and why it was canceled.
func document(t api.Task) string {
	m := metaOf(t)
	return m.render() + renderBody(t)
}

func taskReadCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "read <id>",
		Short:   "print a task, whatever its status",
		Long:    "prints a task as markdown to read: its frontmatter block, then its title and why it was canceled",
		GroupID: "read",
		Args:    cobra.ExactArgs(1),
		RunE: withTask(e, func(id int, _ []string) int {
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			t, err := c.Get(id)
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
	ID          int        `json:"id"`
	Status      api.Status `json:"status"`
	Title       string     `json:"title"`
	Frontmatter meta       `json:"frontmatter"`
	Updated     string     `json:"updated"`
	Body        string     `json:"body"`
}

func taskLsCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "ls [status]",
		Short:   "list tasks: todo, then done, then canceled",
		Long:    "lists every task in status order (todo, done, canceled), or those in one status.\non a terminal prints a table of status, title, and created (its date), colored by status;\npiped, prints the same columns tab-separated without a header or color; --json adds id, the frontmatter, updated, and the body",
		GroupID: "browse",
		Args:    cobra.MaximumNArgs(1),
		RunE: do(func(args []string) int {
			status := api.Status(arg(args, 0))
			if status != "" {
				if err := checkStatus(status); err != nil {
					return fail(e.stderr, err, exitUsage)
				}
			}
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			tasks, err := c.List()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if status != "" {
				tasks = slices.DeleteFunc(tasks, func(t api.Task) bool { return t.Status != status })
			}
			if asJSON {
				rows := make([]taskEntry, len(tasks))
				for i, t := range tasks {
					rows[i] = taskEntry{ID: t.Id, Status: t.Status, Title: t.Title, Frontmatter: metaOf(t), Updated: t.Updated, Body: renderBody(t)}
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
		cells[i] = []string{string(t.Status), t.Title, createdDate(t.Created)}
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
	ID     int        `json:"id"`
	Status api.Status `json:"status"`
	Line   int        `json:"line"`
	Text   string     `json:"text"`
}

func taskGrepCmd(e *env) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "grep <query>",
		Short:   "find lines containing query in tasks (case-insensitive)",
		Long:    "finds lines containing query, case-insensitive, in every task as task read prints it, in status order.\non a terminal prints a table of status, task, line, and text with the query highlighted;\npiped, prints id:line: text",
		GroupID: "browse",
		Args:    cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			query := arg(args, 0)
			c, err := client()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			tasks, err := c.List()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			matches := []match{}
			q := strings.ToLower(query)
			for _, t := range tasks {
				for i, l := range strings.Split(document(t), "\n") {
					if strings.Contains(strings.ToLower(l), q) {
						matches = append(matches, match{ID: t.Id, Status: t.Status, Line: i + 1, Text: l})
					}
				}
			}
			if asJSON {
				return emit(e.stdout, matches)
			}
			width, tty := terminal(e.stdout)
			if !tty {
				for _, m := range matches {
					fmt.Fprintf(e.stdout, "%d:%d: %s\n", m.ID, m.Line, m.Text)
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
		cells[i] = []string{string(m.Status), "#" + strconv.Itoa(m.ID), strconv.Itoa(m.Line), highlight(m.Text, query)}
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
