package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gitt510/nabu/internal/store"
)

// canvasCmd is the canvas command group: one file at the root, CANVAS.md,
// edited by the user in an editor and by the agent through these commands.
func canvasCmd(e *env) *cobra.Command {
	cmd := parentCmd(e, &cobra.Command{
		Use:   "canvas",
		Short: "co-write a draft in CANVAS.md",
		Long:  "The canvas is one file at the root, CANVAS.md, edited by the user in an\neditor and by the agent through these commands. Git ignores it.",
	})
	cmd.AddCommand(
		canvasPutCmd(e, "open", "start a draft in CANVAS.md (refuses when one is there)",
			"starts a draft; the canvas must be empty (see canvas save / drop). the body is read from stdin unless --content is given"),
		canvasShowCmd(e, "read", "print the draft as it is now",
			"prints the draft as it is now, including the user's edits"),
		canvasPutCmd(e, "write", "replace the draft with a revision",
			"replaces the draft; the canvas must hold one (see canvas open). read or diff it first so the user's edits are not lost"),
		canvasShowCmd(e, "diff", "the user's edits since the agent last wrote (empty when none)",
			"unified diff from the agent's last open/write to the file as it is now: the user's edits. prints nothing when the user has not touched it"),
		canvasSaveCmd(e),
		canvasDropCmd(e),
	)
	return cmd
}

// canvasResult is the JSON document of open, write, and drop. File is the
// absolute path, for the user's editor.
type canvasResult struct {
	Path   string `json:"path"`
	File   string `json:"file"`
	Action string `json:"action"`
	Bytes  int    `json:"bytes"`
}

// canvasPutCmd is open and write: both take a body and record it as the
// agent's snapshot; they differ only in what state they accept.
func canvasPutCmd(e *env, action, short, long string) *cobra.Command {
	var content string
	cmd := &cobra.Command{
		Use:   action,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		RunE: do(func([]string) int {
			body, err := readBody(content, e.stdin)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			var n int
			if action == "open" {
				n, err = s.CanvasOpen(body)
			} else {
				n, err = s.CanvasWrite(body)
			}
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, canvasResult{Path: store.CanvasFile, File: s.CanvasPath(), Action: action, Bytes: n})
		}),
	}
	cmd.Flags().StringVar(&content, "content", "", "draft body; stdin is read when omitted")
	return cmd
}

// canvasShowCmd is read and diff: both print raw text.
func canvasShowCmd(e *env, action, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:   action,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		RunE: do(func([]string) int {
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			var b []byte
			if action == "read" {
				b, err = s.CanvasRead()
			} else {
				b, err = s.CanvasDiff()
			}
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			_, _ = e.stdout.Write(b)
			return exitOK
		}),
	}
}

func canvasSaveCmd(e *env) *cobra.Command {
	var tk []string
	cmd := &cobra.Command{
		Use:   "save <slug>",
		Short: "move the draft to gallery/<slug>.md and commit; empties the canvas",
		Long:  "writes the draft to gallery/<slug>.md (created or replaced) with a frontmatter block carrying created and any tickets, commits it, and empties the canvas.\nthe draft must not start with a frontmatter block of its own",
		Args:  cobra.ExactArgs(1),
		RunE: do(func(args []string) int {
			slug := arg(args, 0)
			if !store.Slug(slug) {
				return fail(e.stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
			}
			fm, err := frontmatter(field{"created", now().Format(time.RFC3339)}, tk)
			if err != nil {
				return fail(e.stderr, err, exitUsage)
			}
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			body, err := s.CanvasRead()
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if hasFrontmatter(body) {
				return fail(e.stderr, errors.New("draft already starts with frontmatter; drop it, metadata comes from flags"), exitFail)
			}
			rel, err := s.CanvasSave(slug, fm)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return finishWrite(s, rel, "canvas save", len(fm)+len(body), nil, e.stdout, e.stderr)

		}),
	}
	cmd.Flags().StringArrayVar(&tk, "ticket", nil, "related issue or PR URL (`https-url`); repeatable")
	return cmd
}

func canvasDropCmd(e *env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drop",
		Short: "empty the canvas",
		Long:  "empties CANVAS.md; the file stays so an open editor sees a reload, not a deletion",
		Args:  cobra.NoArgs,
		RunE: do(func(args []string) int {
			s, err := openStore(e.root)
			if err != nil {
				return fail(e.stderr, err, exitFail)
			}
			if err := s.CanvasDrop(); err != nil {
				return fail(e.stderr, err, exitFail)
			}
			return emit(e.stdout, canvasResult{Path: store.CanvasFile, File: s.CanvasPath(), Action: "drop"})

		}),
	}
	return cmd
}

// field is one scalar frontmatter entry; an empty value is left out.
type field struct{ key, value string }

// frontmatter renders the YAML block for the given field and tickets, or ""
// when both are empty so a plain draft stays a plain file.
func frontmatter(f field, tk []string) (string, error) {
	if f.value == "" && len(tk) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("---\n")
	if f.value != "" {
		fmt.Fprintf(&b, "%s: %q\n", f.key, f.value)
	}
	if len(tk) > 0 {
		b.WriteString("tickets:\n")
		for _, t := range tk {
			u, err := url.Parse(t)
			if err != nil || u.Scheme != "https" || u.Host == "" {
				return "", fmt.Errorf("--ticket must be a full https URL: %s", t)
			}
			fmt.Fprintf(&b, "  - %q\n", t)
		}
	}
	b.WriteString("---\n")
	return b.String(), nil
}
