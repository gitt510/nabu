package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gitt510/nabu/internal/store"
)

const canvasUsage = `usage: nabu canvas <open|read|write|diff|save|drop> [flags] [args]

  open          start a draft in CANVAS.md (refuses when one is there)
  read          print the draft as it is now
  write         replace the draft with a revision
  diff          the user's edits since the agent last wrote
  save <slug>   move the draft to writing/<slug>.md and commit; empties the canvas
  drop          empty the canvas

The canvas is one file at the root, CANVAS.md, edited by the user in an
editor and by the agent through these commands. Git ignores it.

See "nabu canvas <command> -h".
`

func runCanvas(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, canvasUsage)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		fmt.Fprint(stdout, canvasUsage)
		return exitOK
	case "open":
		return runCanvasPut(args[1:], "open", stdin, stdout, stderr)
	case "write":
		return runCanvasPut(args[1:], "write", stdin, stdout, stderr)
	case "read":
		return runCanvasRead(args[1:], stdout, stderr)
	case "diff":
		return runCanvasDiff(args[1:], stdout, stderr)
	case "save":
		return runCanvasSave(args[1:], stdout, stderr)
	case "drop":
		return runCanvasDrop(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown canvas command: %s\n\n%s", args[0], canvasUsage)
	return exitUsage
}

// canvasResult is the JSON document of open, write, and drop. File is the
// absolute path, for the user's editor.
type canvasResult struct {
	Path   string `json:"path"`
	File   string `json:"file"`
	Action string `json:"action"`
	Bytes  int    `json:"bytes"`
}

// runCanvasPut is open and write: both take a body and record it as the
// agent's snapshot; they differ only in what state they accept.
func runCanvasPut(args []string, action string, stdin io.Reader, stdout, stderr io.Writer) int {
	var c common
	var content string
	synopsis := map[string]string{
		"open":  "nabu canvas open [--content <text>] [--json]\n\nstarts a draft; the canvas must be empty (see canvas save / drop). the body is read from stdin unless --content is given",
		"write": "nabu canvas write [--content <text>] [--json]\n\nreplaces the draft; the canvas must hold one (see canvas open). read or diff it first so the user's edits are not lost",
	}[action]
	fs := newFlagSet("canvas "+action, synopsis, stderr)
	c.bind(fs)
	fs.StringVar(&content, "content", "", "draft body; stdin is read when omitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 0 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	body, err := readBody(content, fs.Lookup("content"), stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	var n int
	if action == "open" {
		n, err = s.CanvasOpen(body)
	} else {
		n, err = s.CanvasWrite(body)
	}
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		return emit(stdout, canvasResult{Path: store.CanvasFile, File: s.CanvasPath(), Action: action, Bytes: n})
	}
	fmt.Fprintf(stdout, "%s %s (%d bytes)\n", action, s.CanvasPath(), n)
	return exitOK
}

func runCanvasRead(args []string, stdout, stderr io.Writer) int {
	var c common
	fs := newFlagSet("canvas read", "nabu canvas read [--json]", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 0 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	b, err := s.CanvasRead()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		return emit(stdout, map[string]string{"path": store.CanvasFile, "content": string(b)})
	}
	_, _ = stdout.Write(b)
	return exitOK
}

func runCanvasDiff(args []string, stdout, stderr io.Writer) int {
	var c common
	fs := newFlagSet("canvas diff", "nabu canvas diff [--json]\n\nunified diff from the agent's last open/write to the file as it is now: the user's edits. empty when the user has not touched it", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 0 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	changed, patch, err := s.CanvasDiff()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		return emit(stdout, map[string]any{"changed": changed, "patch": patch})
	}
	fmt.Fprint(stdout, patch)
	return exitOK
}

func runCanvasSave(args []string, stdout, stderr io.Writer) int {
	var c common
	var tk tickets
	var noCommit bool
	fs := newFlagSet("canvas save", "nabu canvas save <slug> [--ticket <https-url>]... [--no-commit] [--json]\n\nwrites the draft to writing/<slug>.md (created or replaced) with a frontmatter block carrying created and any tickets, commits it, and empties the canvas.\nthe draft must not start with a frontmatter block of its own", stderr)
	c.bind(fs)
	fs.Var(&tk, "ticket", "related issue or PR URL (https); repeatable")
	fs.BoolVar(&noCommit, "no-commit", false, "leave the change uncommitted")
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	slug := fs.Arg(0)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	fm, err := frontmatter(field{"created", time.Now().Format(time.RFC3339)}, tk)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	body, err := s.CanvasRead()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if strings.HasPrefix(strings.TrimLeft(string(body), "\n"), "---") {
		return fail(stderr, errors.New("draft already starts with frontmatter; drop it, metadata comes from flags"), exitFail)
	}
	rel, _, err := s.CanvasSave(slug, fm)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "canvas save", len(fm)+len(body), noCommit, c.json, stdout, stderr)
}

func runCanvasDrop(args []string, stdout, stderr io.Writer) int {
	var c common
	fs := newFlagSet("canvas drop", "nabu canvas drop [--json]\n\nempties CANVAS.md; the file stays so an open editor sees a reload, not a deletion", stderr)
	c.bind(fs)
	if ok, code := parse(fs, args, stdout, stderr); !ok {
		return code
	}
	if fs.NArg() != 0 {
		fs.SetOutput(stderr)
		fs.Usage()
		return exitUsage
	}
	s, err := c.open()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if err := s.CanvasDrop(); err != nil {
		return fail(stderr, err, exitFail)
	}
	if c.json {
		return emit(stdout, canvasResult{Path: store.CanvasFile, File: s.CanvasPath(), Action: "drop"})
	}
	fmt.Fprintf(stdout, "drop %s (empty)\n", s.CanvasPath())
	return exitOK
}
