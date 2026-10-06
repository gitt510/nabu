package main

import (
	"io"
	"os"
	"path"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"

	"github.com/gitt510/nabu/internal/store"
)

// terminal reports whether w is a terminal and, if so, its width in cells
// (0 when it cannot be read). Tables and color are for a terminal only;
// piped output stays plain so scripts and agents can read it.
func terminal(w io.Writer) (int, bool) {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(f.Fd()) {
		return 0, false
	}
	width, _, err := term.GetSize(f.Fd())
	if err != nil {
		return 0, true
	}
	return width, true
}

// statusColors are ANSI palette colors, so each status follows the
// terminal's own theme, light or dark.
var statusColors = map[string]string{
	"inbox":   "6", // cyan
	"doing":   "3", // yellow
	"done":    "2", // green
	"dropped": "8", // gray
	"stray":   "1", // red
}

var cellStyle = lipgloss.NewStyle().Padding(0, 1)

func statusStyle(status string) lipgloss.Style {
	return cellStyle.Foreground(lipgloss.Color(statusColors[status]))
}

// drawTable renders a bordered table no wider than width (0 for no limit);
// a cell that does not fit is cut. style picks each data cell's style; the
// header row is bold.
func drawTable(headers []string, rows [][]string, width int, style func(row, col int) lipgloss.Style) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		Headers(headers...).
		Rows(rows...).
		Wrap(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return cellStyle.Bold(true)
			}
			return style(row, col)
		})
	// Width also stretches a narrow table, so it is set only to shrink one
	if out := t.String(); width == 0 || lipgloss.Width(out) <= width {
		return out
	}
	return t.Width(width).String()
}

// taskStatusOrStray is the status folder of a task path, or "stray" for a
// tasks/<slug>.md outside every folder.
func taskStatusOrStray(rel string) string {
	if st := store.TaskStatusOf(rel); st != "" {
		return st
	}
	return "stray"
}

func taskSlug(rel string) string {
	return strings.TrimSuffix(path.Base(rel), ".md")
}

// byWorkflow orders task paths by status in workflow order (inbox, doing,
// done, dropped), stray last. Paths within a status compare equal, so a
// stable sort keeps their path order.
func byWorkflow(a, b string) int {
	rank := func(rel string) int {
		if i := slices.Index(store.TaskStatuses, store.TaskStatusOf(rel)); i >= 0 {
			return i
		}
		return len(store.TaskStatuses)
	}
	return rank(a) - rank(b)
}
