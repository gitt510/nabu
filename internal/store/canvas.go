package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
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

// CanvasOpen starts a draft. It refuses when the canvas already holds one.
func (s *Store) CanvasOpen(body []byte) (int, error) {
	cur, err := os.ReadFile(s.CanvasPath())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
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
	b, err := os.ReadFile(s.CanvasPath())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, ErrCanvasEmpty
	}
	return b, nil
}

// CanvasDiff is the unified diff from the agent's last write to the file
// as it is now: the user's edits. It is empty when the two agree.
func (s *Store) CanvasDiff() ([]byte, error) {
	if _, err := s.CanvasRead(); err != nil {
		return nil, err
	}
	snap := filepath.Join(s.Root, canvasSnapshot)
	if _, err := os.Stat(snap); errors.Is(err, fs.ErrNotExist) {
		snap = os.DevNull // the draft was started by hand; all of it is the user's
	}
	out, err := exec.Command("git", "-C", s.Root, "diff", "--no-index", "--", snap, s.CanvasPath()).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		err = nil // exit 1 is "differences found"
	}
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return out, nil
}

// CanvasSave writes the draft, prefixed with fm, to writing/<slug>.md,
// overwriting an earlier save of the same slug, and empties the canvas. The
// commit is the caller's.
func (s *Store) CanvasSave(slug, fm string) (string, error) {
	body, err := s.CanvasRead()
	if err != nil {
		return "", err
	}
	abs, err := s.Resolve(WritingDir + "/" + slug + ".md")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, append([]byte(fm), body...), 0o644); err != nil {
		return "", err
	}
	return s.Rel(abs), s.CanvasDrop()
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
