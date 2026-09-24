package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gitt510/nabu/internal/store"
)

const canvasUsage = `usage: nabu canvas <open|read|write|diff|save|drop> [flags] [args]

  open          start a draft in CANVAS.md (refuses when one is there)
  read          print the draft as it is now
  write         replace the draft with a revision
  diff          the user's edits since the agent last wrote (empty when none)
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
	case "open", "write":
		return runCanvasPut(args[1:], args[0], stdin, stdout, stderr)
	case "read", "diff":
		return runCanvasShow(args[1:], args[0], stdout, stderr)
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
	synopsis := map[string]string{
		"open":  "nabu canvas open [--content <text>]\n\nstarts a draft; the canvas must be empty (see canvas save / drop). the body is read from stdin unless --content is given",
		"write": "nabu canvas write [--content <text>]\n\nreplaces the draft; the canvas must hold one (see canvas open). read or diff it first so the user's edits are not lost",
	}[action]
	fs := newFlagSet("canvas "+action, synopsis)
	root := bindRoot(fs)
	content := fs.String("content", "", "draft body; stdin is read when omitted")
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	body, err := readBody(*content, stdin)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := openStore(*root)
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
	return emit(stdout, canvasResult{Path: store.CanvasFile, File: s.CanvasPath(), Action: action, Bytes: n})
}

// runCanvasShow is read and diff: both print raw text.
func runCanvasShow(args []string, action string, stdout, stderr io.Writer) int {
	synopsis := map[string]string{
		"read": "nabu canvas read\n\nprints the draft as it is now, including the user's edits",
		"diff": "nabu canvas diff\n\nunified diff from the agent's last open/write to the file as it is now: the user's edits. prints nothing when the user has not touched it",
	}[action]
	fs := newFlagSet("canvas "+action, synopsis)
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	var b []byte
	if action == "read" {
		b, err = s.CanvasRead()
	} else {
		b, err = s.CanvasDiff()
	}
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	_, _ = stdout.Write(b)
	return exitOK
}

func runCanvasSave(args []string, stdout, stderr io.Writer) int {
	var tk tickets
	fs := newFlagSet("canvas save", "nabu canvas save <slug> [--ticket <https-url>]...\n\nwrites the draft to writing/<slug>.md (created or replaced) with a frontmatter block carrying created and any tickets, commits it, and empties the canvas.\nthe draft must not start with a frontmatter block of its own")
	root := bindRoot(fs)
	fs.Var(&tk, "ticket", "related issue or PR URL (https); repeatable")
	if ok, code := parse(fs, args, 1, 1, stdout, stderr); !ok {
		return code
	}
	slug := fs.Arg(0)
	if !store.Slug(slug) {
		return fail(stderr, fmt.Errorf("slug must be lowercase kebab-case without / or .md: %s", slug), exitUsage)
	}
	fm, err := frontmatter(field{"created", time.Now().Format(time.RFC3339)}, tk)
	if err != nil {
		return fail(stderr, err, exitUsage)
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	body, err := s.CanvasRead()
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if hasFrontmatter(body) {
		return fail(stderr, errors.New("draft already starts with frontmatter; drop it, metadata comes from flags"), exitFail)
	}
	rel, err := s.CanvasSave(slug, fm)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	return finishWrite(s, rel, "canvas save", len(fm)+len(body), stdout, stderr)
}

func runCanvasDrop(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("canvas drop", "nabu canvas drop\n\nempties CANVAS.md; the file stays so an open editor sees a reload, not a deletion")
	root := bindRoot(fs)
	if ok, code := parse(fs, args, 0, 0, stdout, stderr); !ok {
		return code
	}
	s, err := openStore(*root)
	if err != nil {
		return fail(stderr, err, exitFail)
	}
	if err := s.CanvasDrop(); err != nil {
		return fail(stderr, err, exitFail)
	}
	return emit(stdout, canvasResult{Path: store.CanvasFile, File: s.CanvasPath(), Action: "drop"})
}
