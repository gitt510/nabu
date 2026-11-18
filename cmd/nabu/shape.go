package main

import (
	"strings"

	"github.com/gitt510/nabu/internal/api"
)

// renderBody writes a task's title as markdown, for reading, with the reason
// a canceled task was closed under it.
func renderBody(t api.Task) string {
	var b strings.Builder
	b.WriteString("# " + t.Title + "\n")
	if len(t.WhyCanceled) > 0 {
		b.WriteString("\n## Why canceled\n\n")
		for _, l := range t.WhyCanceled {
			b.WriteString("- " + l + "\n")
		}
	}
	return b.String()
}
