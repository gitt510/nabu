package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/api"
)

// meta is a task's frontmatter, and also its JSON form in every result.
// fields says what each key means and accepts.
type meta struct {
	Created   string   `json:"created,omitempty"`
	Area      string   `json:"area,omitempty"`
	Project   string   `json:"project,omitempty"`
	Scheduled string   `json:"scheduled,omitempty"`
	Waiting   string   `json:"waiting,omitempty"`
	Tickets   []string `json:"tickets,omitempty"`
	Prs       []string `json:"prs,omitempty"`
	Links     []string `json:"links,omitempty"`
}

// field is one frontmatter key: how it is written, set, and checked.
type field struct {
	key string
	// flag sets the value on task new and task set; "" when nabu stamps it.
	flag  string
	usage string
	// doc is the sentence the command's long help gives the flag.
	doc string
	// clear is the help of --clear-<key> on task set; "" when the value
	// cannot be dropped.
	clear string
	// onNew says task new takes the flag too; task set takes every flag.
	onNew bool
	// one of these points at the value: a scalar or a list.
	str  func(m *meta) *string
	list func(m *meta) *[]string
}

// fields is the frontmatter, in the order render writes it. It is the one
// definition of every key; render, the API fields, the flags, and their help
// derive from it.
var fields = []field{
	{
		key: "created",
		str: func(m *meta) *string { return &m.Created },
	},
	{
		key: "area", flag: "area", onNew: true,
		usage: "which side of life the task belongs to; required",
		doc:   "--area is which side of life the task belongs to.",
		str:   func(m *meta) *string { return &m.Area },
	},
	{
		key: "project", flag: "project", onNew: true,
		usage: "the service or repo the task ships into",
		doc:   "--project is what the task ships into: a service at work, a repo at home; leave it out when the task belongs to none.",
		clear: "drop project: the task belongs to none",
		str:   func(m *meta) *string { return &m.Project },
	},
	{
		key: "scheduled", flag: "scheduled", onNew: true,
		usage: "planned work time, or a date for the whole day",
		doc:   "--scheduled is the time the work is planned to happen, not a deadline, or a date for the whole day.",
		clear: "drop scheduled",
		str:   func(m *meta) *string { return &m.Scheduled },
	},
	{
		key: "waiting", flag: "waiting",
		usage: "who or what the next action waits on",
		doc:   "--waiting names who or what the next action waits on: while it is set the ball is with someone else, so the task stays in doing/.",
		clear: "drop waiting: the ball is back",
		str:   func(m *meta) *string { return &m.Waiting },
	},
	{
		key: "tickets", flag: "ticket", onNew: true,
		usage: "work item URL: issue, Wrike, Zendesk; repeatable",
		clear: "drop every ticket",
		list:  func(m *meta) *[]string { return &m.Tickets },
	},
	{
		key: "prs", flag: "pr", onNew: true,
		usage: "pull request URL; repeatable",
		clear: "drop every pr",
		list:  func(m *meta) *[]string { return &m.Prs },
	},
	{
		key: "links", flag: "link", onNew: true,
		usage: "any other URL: repo, article, post; repeatable",
		clear: "drop every link",
		list:  func(m *meta) *[]string { return &m.Links },
	},
}

// urlDoc says how the three URL lists are told apart.
const urlDoc = "a URL goes by what it points at, whoever opened it and whatever it is to this task: --ticket is a work item (an issue, a Wrike task, a Zendesk request), --pr is a pull request, --link is everything else (a repo, an article, a post). each URL is added once."

// render writes the YAML block task read prints above the body, or "" when
// every field is empty.
func (m *meta) render() string {
	var b strings.Builder
	for _, f := range fields {
		if f.str != nil {
			if v := *f.str(m); v != "" {
				fmt.Fprintf(&b, "%s: %q\n", f.key, v)
			}
			continue
		}
		if l := *f.list(m); len(l) > 0 {
			b.WriteString(f.key + ":\n")
			for _, v := range l {
				fmt.Fprintf(&b, "  - %q\n", v)
			}
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "---\n" + b.String() + "---\n"
}

// metaOf is a task's frontmatter.
func metaOf(t api.Task) meta {
	return meta{Created: t.Created, Area: t.Area, Project: t.Project, Scheduled: t.Scheduled, Waiting: t.Waiting, Tickets: t.Tickets, Prs: t.Prs, Links: t.Links}
}

// apiFields are the API fields of every key a flag sets, with null for an
// empty scalar so the API drops it.
func (m *meta) apiFields() map[string]any {
	out := map[string]any{}
	for _, f := range fields {
		if f.flag == "" {
			continue
		}
		if f.str != nil {
			if v := *f.str(m); v != "" {
				out[f.key] = v
			} else {
				out[f.key] = nil
			}
			continue
		}
		out[f.key] = append([]string{}, *f.list(m)...)
	}
	return out
}

// metaFlags holds the frontmatter flags of task new or task set as parsed.
type metaFlags struct {
	set   bool // task set: every flag, with --clear-*
	str   map[string]*string
	list  map[string]*[]string
	clear map[string]*bool
}

// addMetaFlags registers the frontmatter flags on cmd: those of task new,
// or every one with its --clear-* on task set.
func addMetaFlags(cmd *cobra.Command, set bool) *metaFlags {
	mf := &metaFlags{set: set, str: map[string]*string{}, list: map[string]*[]string{}, clear: map[string]*bool{}}
	for _, f := range mf.fields() {
		if f.str != nil {
			mf.str[f.key] = cmd.Flags().String(f.flag, "", f.usage)
		} else {
			mf.list[f.key] = cmd.Flags().StringArray(f.flag, nil, f.usage)
		}
		if set && f.clear != "" {
			mf.clear[f.key] = cmd.Flags().Bool("clear-"+f.key, false, f.clear)
		}
	}
	return mf
}

// fields are the keys the command takes a flag for.
func (mf *metaFlags) fields() []field {
	var out []field
	for _, f := range fields {
		if f.flag != "" && (mf.set || f.onNew) {
			out = append(out, f)
		}
	}
	return out
}

// help is the long help the command gives its frontmatter flags.
func (mf *metaFlags) help() string {
	var lines []string
	for _, f := range mf.fields() {
		if f.doc != "" {
			lines = append(lines, f.doc)
		}
	}
	if mf.set {
		lines = append(lines, "--ticket, --pr and --link add to their list; the matching --clear-* empties it first.")
	}
	return strings.Join(append(lines, urlDoc), "\n")
}

// given reports whether any frontmatter flag was passed.
func (mf *metaFlags) given() bool {
	for _, f := range mf.fields() {
		if (f.str != nil && *mf.str[f.key] != "") || (f.list != nil && len(*mf.list[f.key]) > 0) {
			return true
		}
		if c := mf.clear[f.key]; c != nil && *c {
			return true
		}
	}
	return false
}

// apply writes the flags into m: a --clear-* empties its key first, a
// scalar replaces the value, and a URL joins its list once.
func (mf *metaFlags) apply(m *meta) error {
	for _, f := range mf.fields() {
		cleared := mf.clear[f.key] != nil && *mf.clear[f.key]
		if f.str != nil {
			v := *mf.str[f.key]
			if v != "" && cleared {
				return fmt.Errorf("--%s and --clear-%s cannot be given together", f.flag, f.key)
			}
			if cleared {
				*f.str(m) = ""
			}
			if v == "" {
				continue
			}
			*f.str(m) = v
			continue
		}
		dst := f.list(m)
		if cleared {
			*dst = nil
		}
		for _, v := range *mf.list[f.key] {
			if !slices.Contains(*dst, v) {
				*dst = append(*dst, v)
			}
		}
	}
	return nil
}
