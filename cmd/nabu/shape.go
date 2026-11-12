package main

import (
	"strings"

	"github.com/gitt510/nabu/internal/api"
)

// renderBody writes a task's title as markdown, for reading, with the reason
// a dropped task was closed under it.
func renderBody(t api.Task) string {
	var b strings.Builder
	b.WriteString("# " + t.Title + "\n")
	if len(t.WhyDropped) > 0 {
		b.WriteString("\n## Why dropped\n\n")
		for _, l := range t.WhyDropped {
			b.WriteString("- " + l + "\n")
		}
	}
	return b.String()
}
