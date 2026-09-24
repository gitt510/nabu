// Package store performs the file operations on the notes root and records
// each write as a git commit. Every path is relative to the root, cleaned,
// confined to it, and ends in .md.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Store is a notes root.
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
const readme = "# notes\n\nManaged by nabu.\n"

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

// ErrExists is returned by Write when the note is already there.
var ErrExists = errors.New("note exists (use note replace to revise it)")

// ErrNotExist is returned by Replace and Move when the note is missing.
var ErrNotExist = errors.New("no such note")

// Resolve turns a note path into an absolute path, rejecting anything that
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
		return "", fmt.Errorf("%s is the canvas, not a note (use nabu canvas)", clean)
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

// Write creates a note. It refuses to overwrite unless force is set.
func (s *Store) Write(p string, content []byte, force bool) (string, error) {
	abs, err := s.Resolve(p)
	if err != nil {
		return "", err
	}
	if !force {
		if _, err := os.Stat(abs); err == nil {
			return "", fmt.Errorf("%s: %w", s.Rel(abs), ErrExists)
		}
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	return s.Rel(abs), os.WriteFile(abs, ensureNewline(content), 0o644)
}

// Replace overwrites the whole body of an existing note. It refuses to
// create one, so a typo in the path cannot silently start a new note.
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

// Move renames a note. The source must exist and the destination must not,
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

// Append adds content to the end of a note, creating it when absent. With a
// heading, the heading line is written first unless the note already ends
// in that section (its last heading is the same), so repeated appends under
// one heading stay in one section.
func (s *Store) Append(p string, content []byte, heading string) (string, error) {
	abs, err := s.Resolve(p)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	old, err := os.ReadFile(abs)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	content = ensureNewline(content)
	var buf bytes.Buffer
	buf.Write(old)
	if len(old) > 0 {
		if !bytes.HasSuffix(old, []byte("\n")) {
			buf.WriteByte('\n')
		}
		// consecutive list items stay one list; anything else is its own
		// paragraph, separated by a blank line
		sameHeading := heading == "" || lastHeading(old) == heading
		joinList := sameHeading && isListItem(lastLine(old)) && isListItem(firstLine(content))
		if !joinList {
			buf.WriteByte('\n')
		}
	}
	if heading != "" && lastHeading(old) != heading {
		buf.WriteString(heading)
		buf.WriteString("\n\n")
	}
	buf.Write(content)
	return s.Rel(abs), os.WriteFile(abs, buf.Bytes(), 0o644)
}

// Read returns a note's content.
func (s *Store) Read(p string) ([]byte, error) {
	abs, err := s.Resolve(p)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(abs)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("no such note: %s", p)
	}
	return b, err
}

// Entry is one note in a listing.
type Entry struct {
	Path     string    `json:"path"`
	Title    string    `json:"title"`
	Modified time.Time `json:"modified"`
	Bytes    int64     `json:"bytes"`
}

// List returns the notes under dir (root when empty), sorted by path.
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
		info, err := d.Info()
		if err != nil {
			return err
		}
		out = append(out, Entry{
			Path:     s.Rel(path),
			Title:    firstTitle(path),
			Modified: info.ModTime().UTC(),
			Bytes:    info.Size(),
		})
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

// Grep finds lines containing query (case-insensitive) across all notes.
func (s *Store) Grep(query string) ([]Match, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}
	entries, err := s.List("")
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var out []Match
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

// Finding is one convention warning from Lint.
type Finding struct {
	Path   string `json:"path"`
	Rule   string `json:"rule"`
	Detail string `json:"detail"`
}

// Lint walks every note and reports the ones that break the naming
// conventions: a filename that is not lowercase kebab-case, a body with no
// "# " title, or a note under tasks/ outside a status folder. It changes
// nothing; repairs go through Move and Replace.
func (s *Store) Lint() ([]Finding, error) {
	entries, err := s.List("")
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, e := range entries {
		if e.Path == "README.md" {
			continue
		}
		if !kebabPath(e.Path) {
			out = append(out, Finding{Path: e.Path, Rule: "filename", Detail: "not lowercase kebab-case (a-z, 0-9, -)"})
		}
		if e.Title == "" {
			out = append(out, Finding{Path: e.Path, Rule: "title", Detail: "no \"# \" heading"})
		}
		if !taskFolderOK(e.Path) {
			out = append(out, Finding{Path: e.Path, Rule: "task", Detail: "not in a status folder tasks/<" + strings.Join(TaskStatuses, "|") + ">/"})
		}
	}
	return out, nil
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
func TaskStatus(status string) bool {
	for _, st := range TaskStatuses {
		if st == status {
			return true
		}
	}
	return false
}

// TaskPath is the note path of a task in the given status folder.
func TaskPath(status, slug string) string {
	return "tasks/" + status + "/" + slug + ".md"
}

// FindTask returns the path of the task with this slug, whatever its
// status, or "" when no status folder has it.
func (s *Store) FindTask(slug string) string {
	for _, st := range TaskStatuses {
		rel := TaskPath(st, slug)
		if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(rel))); err == nil {
			return rel
		}
	}
	return ""
}

// taskFolderOK reports whether a note under tasks/ sits directly in a
// status folder. Notes outside tasks/ always pass.
func taskFolderOK(p string) bool {
	rest, ok := strings.CutPrefix(p, "tasks/")
	if !ok {
		return true
	}
	dir, _, ok := strings.Cut(rest, "/")
	return ok && TaskStatus(dir) && !strings.Contains(rest[len(dir)+1:], "/")
}

// kebabPath reports whether every segment of a slash path is lowercase
// kebab-case and the file ends in .md.
func kebabPath(p string) bool {
	for _, seg := range strings.Split(strings.TrimSuffix(p, ".md"), "/") {
		if seg == "" || seg[0] == '-' || seg[len(seg)-1] == '-' {
			return false
		}
		for _, r := range seg {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
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

func lastLine(b []byte) string {
	b = bytes.TrimRight(b, "\n")
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		b = b[i+1:]
	}
	return string(b)
}

func firstLine(b []byte) string {
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// isListItem reports a markdown bullet or numbered item (checkboxes included).
func isListItem(line string) bool {
	t := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "+ ") {
		return true
	}
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(t) && (t[i] == '.' || t[i] == ')') && t[i+1] == ' '
}

// lastHeading is the last markdown heading line in b, or "".
func lastHeading(b []byte) string {
	last := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "#") {
			last = strings.TrimRight(line, " \t")
		}
	}
	return last
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
