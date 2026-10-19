package main

import (
	"strings"

	"github.com/gitt510/nabu/internal/api"
)

// section is one heading of the task body and the API field its lines go to.
type section struct {
	heading, field string
	lines          func(t *api.Task) []string
}

// sections are the headings every task carries, in this order: SCQA, the
// introduction Barbara Minto lays out in The Pyramid Principle. whyDropped
// may follow them; a dropped task must carry it.
var sections = []section{
	{"## Situation", "situation", func(t *api.Task) []string { return t.Situation }},
	{"## Complication", "complication", func(t *api.Task) []string { return t.Complication }},
	{"## Question", "question", func(t *api.Task) []string { return t.Question }},
	{"## Answer", "answer", func(t *api.Task) []string { return t.Answer }},
}

var whyDropped = section{"## Why dropped", "why_dropped", func(t *api.Task) []string { return t.WhyDropped }}

// renderBody writes a task's title and sections as markdown, for reading.
func renderBody(t api.Task) string {
	var b strings.Builder
	b.WriteString("# " + t.Title + "\n")
	all := sections
	if len(t.WhyDropped) > 0 {
		all = append(all[:len(all):len(all)], whyDropped)
	}
	for _, s := range all {
		b.WriteString("\n" + s.heading + "\n\n")
		for _, l := range s.lines(&t) {
			b.WriteString("- " + l + "\n")
		}
	}
	return b.String()
}
