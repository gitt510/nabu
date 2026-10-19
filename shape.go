package main

import (
	"fmt"
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

// shapeHelp is the task shape as the help states it; every command that
// takes a body points here instead of restating it.
var shapeHelp = fmt.Sprintf("the body is \"# <title>\", then %q, %q, %q, %q in that order, each followed by its lines, and no other \"## \" heading; %q may follow, and a dropped task needs it.\nthe API refuses a line over 45 characters, a heading inside a section, and a dropped task without a reason.", sections[0].heading, sections[1].heading, sections[2].heading, sections[3].heading, whyDropped.heading)

// parseBody splits a task body into its title and the API fields of its
// sections. It checks only the structure; the API checks every line.
func parseBody(body []byte) (string, map[string]any, error) {
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	title := ""
	fields := map[string]any{}
	var cur *section
	next := 0
	for i, l := range lines {
		l = strings.TrimRight(l, " \t")
		switch {
		case strings.TrimSpace(l) == "":
			continue
		case title == "":
			t, ok := strings.CutPrefix(l, "# ")
			if !ok || strings.TrimSpace(t) == "" {
				return "", nil, fmt.Errorf("line %d: the body must open with \"# <title>\"", i+1)
			}
			title = strings.TrimSpace(t)
		case strings.HasPrefix(l, "## "):
			switch {
			case next < len(sections) && l == sections[next].heading:
				cur = &sections[next]
				next++
			case next == len(sections) && l == whyDropped.heading && cur != &whyDropped:
				cur = &whyDropped
			case next < len(sections):
				return "", nil, fmt.Errorf("line %d: expected %q, got %q", i+1, sections[next].heading, l)
			default:
				return "", nil, fmt.Errorf("line %d: %q is not a task section", i+1, l)
			}
			fields[cur.field] = []string{}
		case cur == nil:
			return "", nil, fmt.Errorf("line %d: text before %q: %s", i+1, sections[0].heading, l)
		default:
			fields[cur.field] = append(fields[cur.field].([]string), l)
		}
	}
	if title == "" {
		return "", nil, fmt.Errorf("the body must open with \"# <title>\"")
	}
	if next < len(sections) {
		return "", nil, fmt.Errorf("body has no %q section", sections[next].heading)
	}
	if _, ok := fields[whyDropped.field]; !ok {
		fields[whyDropped.field] = []string{}
	}
	return title, fields, nil
}

// renderBody writes a task's body in the shape parseBody reads.
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
			b.WriteString(l + "\n")
		}
	}
	return b.String()
}
