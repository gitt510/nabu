package main

import (
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/term"
	"github.com/gitt510/nabu/internal/api"
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
var statusColors = map[api.Status]string{
	api.Inbox:   "6", // cyan
	api.Doing:   "3", // yellow
	api.Done:    "2", // green
	api.Dropped: "8", // gray
}

var cellStyle = lipgloss.NewStyle().Padding(0, 1)

func statusStyle(status api.Status) lipgloss.Style {
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
