// Package store performs the file operations on the repository root and
// records each write as a git commit. Every path is relative to the root,
// cleaned, confined to it, and ends in .md.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Store is a repository root.
type Store struct {
	Root string
}

// Open validates that root exists and is a git repository.
func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("no root")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("root: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("root is not a directory: %s", abs)
	}
	if _, err := os.Stat(filepath.Join(abs, ".git")); err != nil {
		return nil, fmt.Errorf("root is not a git repository: %s", abs)
	}
	return &Store{Root: abs}, nil
}

// InitResult says what Init had to create.
type InitResult struct {
	Root       string `json:"root"`
	CreatedDir bool   `json:"created_dir"`
	InitedGit  bool   `json:"inited_git"`
	Committed  bool   `json:"committed"`
}

// readme is the seed file of a fresh root, so the first commit has content.
const readme = "# tasks\n\nManaged by nabu.\n"

// Init makes root usable: the directory exists, it is a git repository on
// main, and it has at least one commit. Each step is skipped when already
// done, so Init is safe to rerun.
func Init(root string) (InitResult, error) {
	var r InitResult
	if root == "" {
		return r, errors.New("no root")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return r, err
	}
	r.Root = abs
	if _, err := os.Stat(abs); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return r, err
		}
		r.CreatedDir = true
	} else if err != nil {
		return r, err
	}
	if _, err := os.Stat(filepath.Join(abs, ".git")); errors.Is(err, fs.ErrNotExist) {
		if out, err := exec.Command("git", "-C", abs, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
			return r, fmt.Errorf("git init: %s", strings.TrimSpace(string(out)))
		}
		r.InitedGit = true
	}
	s := &Store{Root: abs}
	if exec.Command("git", "-C", abs, "rev-parse", "--verify", "-q", "HEAD").Run() == nil {
		return r, nil // has history already; nothing to seed
	}
	if _, err := os.Stat(filepath.Join(abs, "README.md")); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(abs, "README.md"), []byte(readme), 0o644); err != nil {
			return r, err
		}
	}
	if err := os.WriteFile(filepath.Join(abs, ".gitignore"), []byte(CanvasFile+"\n"+canvasSnapshot+"\n"), 0o644); err != nil {
		return r, err
	}
	r.Committed, err = s.Commit("nabu: init", "README.md", ".gitignore")
	return r, err
}

// ErrExists is returned by Write and Move when the destination is taken.
var ErrExists = errors.New("already exists")

// ErrNotExist is returned by Replace and Move when the file is missing.
var ErrNotExist = errors.New("no such file")

// Resolve turns a path into an absolute path, rejecting anything that
// escapes the root or is not a markdown file.
func (s *Store) Resolve(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("path must be relative to root: %s", p)
	}
	clean := filepath.Clean(p)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("path escapes root: %s", p)
	}
	if strings.HasPrefix(clean, ".git"+string(filepath.Separator)) || clean == ".git" {
		return "", fmt.Errorf("path is inside .git: %s", p)
	}
	if isCanvasPath(filepath.ToSlash(clean)) {
		return "", fmt.Errorf("%s is the canvas (use nabu canvas)", clean)
	}
	if filepath.Ext(clean) != ".md" {
		return "", fmt.Errorf("path must end in .md: %s", p)
	}
	return filepath.Join(s.Root, clean), nil
}

// Rel is the inverse of Resolve for display.
func (s *Store) Rel(abs string) string {
	r, err := filepath.Rel(s.Root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(r)
}

// Write creates a file. It refuses to overwrite.
func (s *Store) Write(p string, content []byte) (string, error) {
	abs, err := s.Resolve(p)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err == nil {
		return "", fmt.Errorf("%s: %w", s.Rel(abs), ErrExists)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	return s.Rel(abs), os.WriteFile(abs, ensureNewline(content), 0o644)
}

// Replace overwrites the whole body of an existing file. It refuses to
// create one, so a typo in the path cannot silently start a new file.
func (s *Store) Replace(p string, content []byte) (string, error) {
	abs, err := s.Resolve(p)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%s: %w", s.Rel(abs), ErrNotExist)
	} else if err != nil {
		return "", err
	}
	return s.Rel(abs), os.WriteFile(abs, ensureNewline(content), 0o644)
}

// Move renames a file. The source must exist and the destination must not,
// so a move never overwrites. Both paths are returned for the commit.
func (s *Store) Move(from, to string) (string, string, error) {
	src, err := s.Resolve(from)
	if err != nil {
		return "", "", err
	}
	dst, err := s.Resolve(to)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
		return "", "", fmt.Errorf("%s: %w", s.Rel(src), ErrNotExist)
	} else if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(dst); err == nil {
		return "", "", fmt.Errorf("%s: %w", s.Rel(dst), ErrExists)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", "", err
	}
	return s.Rel(src), s.Rel(dst), os.Rename(src, dst)
}

// Read returns a file's content.
func (s *Store) Read(p string) ([]byte, error) {
	abs, err := s.Resolve(p)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s: %w", p, ErrNotExist)
	}
	return b, err
}

// Entry is one file in a listing.
type Entry struct {
	Path  string
	Title string
}

// List returns the markdown files under dir (root when empty), sorted by path.
func (s *Store) List(dir string) ([]Entry, error) {
	start := s.Root
	if dir != "" {
		clean := filepath.Clean(dir)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "..") {
			return nil, fmt.Errorf("dir escapes root: %s", dir)
		}
		start = filepath.Join(s.Root, clean)
	}
	var out []Entry
	err := filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" || isCanvasPath(s.Rel(path)) {
			return nil
		}
		out = append(out, Entry{Path: s.Rel(path), Title: firstTitle(path)})
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("no such dir: %s", dir)
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Match is one grep hit.
type Match struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// Grep finds lines containing query (case-insensitive) under dir.
func (s *Store) Grep(dir, query string) ([]Match, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}
	entries, err := s.List(dir)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	out := []Match{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(e.Path)))
		if err != nil {
			return nil, err
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(strings.ToLower(line), q) {
				out = append(out, Match{Path: e.Path, Line: i + 1, Text: line})
			}
		}
	}
	return out, nil
}

// Commit stages the given paths and commits them together. It is a no-op
// (false) when none of them has a change to record. A removed path (the
// source of a move) is staged as a deletion.
func (s *Store) Commit(message string, rels ...string) (bool, error) {
	if len(rels) == 0 {
		return false, errors.New("nothing to commit")
	}
	// A path that is neither on disk nor in the index (the source of a move
	// that was never committed) has nothing to stage, and naming it would
	// make git add fail on the pathspec.
	rels = s.stageable(rels)
	if len(rels) == 0 {
		return false, nil
	}
	if err := s.git(append([]string{"add", "-A", "--"}, rels...)...); err != nil {
		return false, err
	}
	out, err := exec.Command("git", append([]string{"-C", s.Root, "status", "--porcelain", "--"}, rels...)...).Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return false, nil
	}
	if err := s.git(append([]string{"commit", "-q", "-m", message, "--"}, rels...)...); err != nil {
		return false, err
	}
	return true, nil
}

// Push sends the current branch to its upstream and returns that upstream
// ("origin/main"). It refuses a branch without one, so the remote is set up
// by hand once and never guessed.
func (s *Store) Push() (string, error) {
	out, err := exec.Command("git", "-C", s.Root, "rev-parse", "--abbrev-ref", "@{u}").Output()
	if err != nil {
		return "", errors.New("no upstream branch; run git push -u <remote> <branch> in the root once")
	}
	upstream := strings.TrimSpace(string(out))
	if err := s.git("push", "-q"); err != nil {
		return "", err
	}
	return upstream, nil
}

// Slug reports whether s is a single lowercase kebab-case filename segment
// without a directory or extension.
func Slug(s string) bool {
	return !strings.ContainsAny(s, "/.") && kebabPath(s+".md")
}

// TaskStatuses are the folders under tasks/, in workflow order. A task's
// status is its folder; nothing else records it.
var TaskStatuses = []string{"inbox", "doing", "done"}

// TaskStatus reports whether status names one of the task folders.
func TaskStatus(status string) bool { return slices.Contains(TaskStatuses, status) }

// TaskPath is the path of a task in the given status folder.
func TaskPath(status, slug string) string {
	return "tasks/" + status + "/" + slug + ".md"
}

// FindTask returns the path of the task with this slug, whatever its
// status, or "" when tasks/ has no such file. A stray tasks/<slug>.md
// outside every status folder is found last, so task mv can put it away.
func (s *Store) FindTask(slug string) string {
	for _, rel := range append(taskPaths(slug), "tasks/"+slug+".md") {
		if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(rel))); err == nil {
			return rel
		}
	}
	return ""
}

func taskPaths(slug string) []string {
	out := make([]string, 0, len(TaskStatuses))
	for _, st := range TaskStatuses {
		out = append(out, TaskPath(st, slug))
	}
	return out
}

// kebabPath reports whether every segment of a slash path is lowercase
// kebab-case and the file ends in .md.
func kebabPath(p string) bool {
	for _, seg := range strings.Split(strings.TrimSuffix(p, ".md"), "/") {
		if seg == "" || seg[0] == '-' || seg[len(seg)-1] == '-' {
			return false
		}
		for _, r := range seg {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}

// stageable keeps the paths that exist on disk or are tracked in the index.
func (s *Store) stageable(rels []string) []string {
	var keep []string
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(s.Root, rel)); err == nil {
			keep = append(keep, rel)
			continue
		}
		if exec.Command("git", "-C", s.Root, "ls-files", "--error-unmatch", "--", rel).Run() == nil {
			keep = append(keep, rel)
		}
	}
	return keep
}

func (s *Store) git(args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", s.Root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(string(out)))
	}
	return nil
}

func ensureNewline(b []byte) []byte {
	if len(b) == 0 || bytes.HasSuffix(b, []byte("\n")) {
		return b
	}
	return append(b, '\n')
}

// firstTitle is the first "# " heading of the file, or "".
func firstTitle(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}
