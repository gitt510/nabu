package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CanvasFile is the one draft the user edits by hand while the agent
// revises it through the CLI. It sits at the root, is never committed, and
// is not a note: List skips it and Resolve refuses it.
const CanvasFile = "CANVAS.md"

// canvasSnapshot is the body as the agent last wrote it, kept next to the
// canvas so Diff can show what the user changed since.
const canvasSnapshot = ".CANVAS.agent.md"

// WritingDir is where a saved canvas lands, as writing/<slug>.md.
const WritingDir = "writing"

// ErrCanvasBusy is returned by CanvasOpen when the canvas is not empty.
var ErrCanvasBusy = errors.New("canvas is not empty (save or drop it first)")

// ErrCanvasEmpty is returned when a command needs a draft and there is none.
var ErrCanvasEmpty = errors.New("canvas is empty (use canvas open)")

// isCanvasPath reports whether a root-relative slash path is the canvas or
// its snapshot.
func isCanvasPath(rel string) bool {
	return rel == CanvasFile || rel == canvasSnapshot
}

// CanvasPath is the absolute path of the canvas, for the user's editor.
func (s *Store) CanvasPath() string { return filepath.Join(s.Root, CanvasFile) }

// canvasBody reads the canvas; a missing file is an empty canvas.
func (s *Store) canvasBody() ([]byte, error) {
	b, err := os.ReadFile(s.CanvasPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

// CanvasOpen starts a draft. It refuses when the canvas already holds one,
// and on first use makes sure the root ignores the canvas files.
func (s *Store) CanvasOpen(body []byte) (int, error) {
	if err := s.ignoreCanvas(); err != nil {
		return 0, err
	}
	cur, err := s.canvasBody()
	if err != nil {
		return 0, err
	}
	if len(bytes.TrimSpace(cur)) > 0 {
		return 0, ErrCanvasBusy
	}
	return s.canvasPut(body)
}

// CanvasWrite replaces the draft with the agent's revision. It refuses an
// empty canvas so a stale session cannot start a draft by accident.
func (s *Store) CanvasWrite(body []byte) (int, error) {
	if _, err := s.CanvasRead(); err != nil {
		return 0, err
	}
	return s.canvasPut(body)
}

// canvasPut writes body to the canvas and records it as the snapshot.
func (s *Store) canvasPut(body []byte) (int, error) {
	body = ensureNewline(body)
	if err := os.WriteFile(s.CanvasPath(), body, 0o644); err != nil {
		return 0, err
	}
	return len(body), os.WriteFile(filepath.Join(s.Root, canvasSnapshot), body, 0o644)
}

// CanvasRead returns the draft as it is now, including the user's edits.
func (s *Store) CanvasRead() ([]byte, error) {
	b, err := s.canvasBody()
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, ErrCanvasEmpty
	}
	return b, nil
}

// CanvasDiff is the unified diff from the agent's last write to the file
// as it is now: the user's edits. changed is false when the two agree.
func (s *Store) CanvasDiff() (changed bool, patch string, err error) {
	if _, err := s.CanvasRead(); err != nil {
		return false, "", err
	}
	snap := filepath.Join(s.Root, canvasSnapshot)
	if _, err := os.Stat(snap); errors.Is(err, fs.ErrNotExist) {
		snap = os.DevNull // the draft was started by hand; all of it is the user's
	}
	cmd := exec.Command("git", "-C", s.Root, "diff", "--no-index", "--", snap, s.CanvasPath())
	out, err := cmd.Output()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return false, "", nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return true, string(out), nil
	default:
		return false, "", fmt.Errorf("git diff: %w", err)
	}
}

// CanvasSave writes the draft, prefixed with fm, to writing/<slug>.md and
// empties the canvas. The note is created when absent and replaced when
// present; action says which. The commit is the caller's.
func (s *Store) CanvasSave(slug, fm string) (rel, action string, err error) {
	body, err := s.CanvasRead()
	if err != nil {
		return "", "", err
	}
	p := WritingDir + "/" + slug + ".md"
	doc := append([]byte(fm), body...)
	abs, err := s.Resolve(p)
	if err != nil {
		return "", "", err
	}
	action = "write"
	if _, err := os.Stat(abs); err == nil {
		action = "replace"
	}
	if action == "replace" {
		rel, err = s.Replace(p, doc)
	} else {
		rel, err = s.Write(p, doc, false)
	}
	if err != nil {
		return "", "", err
	}
	return rel, action, s.CanvasDrop()
}

// CanvasDrop empties the canvas and forgets the snapshot. The file stays so
// an editor that has it open sees a reload, not a deletion.
func (s *Store) CanvasDrop() error {
	if err := os.WriteFile(s.CanvasPath(), nil, 0o644); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.Root, canvasSnapshot)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// ignoreCanvas adds the canvas files to the root's .gitignore and commits
// that once. A root that already ignores them is left alone.
func (s *Store) ignoreCanvas() error {
	path := filepath.Join(s.Root, ".gitignore")
	old, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	have := map[string]bool{}
	for _, line := range strings.Split(string(old), "\n") {
		have[strings.TrimSpace(line)] = true
	}
	var add []string
	for _, f := range []string{CanvasFile, canvasSnapshot} {
		if !have[f] && !have["/"+f] {
			add = append(add, f)
		}
	}
	if len(add) == 0 {
		return nil
	}
	var buf bytes.Buffer
	buf.Write(old)
	if len(old) > 0 && !bytes.HasSuffix(old, []byte("\n")) {
		buf.WriteByte('\n')
	}
	buf.WriteString(strings.Join(add, "\n") + "\n")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return err
	}
	_, err = s.Commit("nabu: ignore "+strings.Join(add, " "), ".gitignore")
	return err
}
