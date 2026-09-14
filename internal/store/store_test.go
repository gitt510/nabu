package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newRepo(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.com"},
		{"config", "user.name", "t"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestOpenRequiresGit(t *testing.T) {
	if _, err := Open(t.TempDir()); err == nil {
		t.Fatal("expected error for non-git root")
	}
}

func TestResolveRejects(t *testing.T) {
	s := newRepo(t)
	for _, p := range []string{"", "/abs.md", "../x.md", "a/../../x.md", "x.txt", ".git/x.md", "."} {
		if _, err := s.Resolve(p); err == nil {
			t.Errorf("%q should be rejected", p)
		}
	}
	if _, err := s.Resolve("career/mygoal_2.md"); err != nil {
		t.Fatal(err)
	}
}

func TestWriteRefusesOverwrite(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Write("a.md", []byte("one"), false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write("a.md", []byte("two"), false); err == nil {
		t.Fatal("expected ErrExists")
	}
	if _, err := s.Write("a.md", []byte("two"), true); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Read("a.md")
	if string(b) != "two\n" {
		t.Fatalf("got %q", b)
	}
}

func TestAppendHeadingOnce(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Append("j.md", []byte("first"), "## 2026-09-03"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append("j.md", []byte("second"), "## 2026-09-03"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Append("j.md", []byte("third"), "## 2026-09-04"); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Read("j.md")
	want := "## 2026-09-03\n\nfirst\n\nsecond\n\n## 2026-09-04\n\nthird\n"
	if string(b) != want {
		t.Fatalf("got:\n%s", b)
	}
}

func TestAppendJoinsListItems(t *testing.T) {
	s := newRepo(t)
	for _, item := range []string{"- a", "- b", "1. c", "para"} {
		if _, err := s.Append("l.md", []byte(item), "## today"); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := s.Read("l.md")
	want := "## today\n\n- a\n- b\n1. c\n\npara\n"
	if string(b) != want {
		t.Fatalf("got:\n%s", b)
	}
}

func TestListAndGrep(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Write("career/goal.md", []byte("# Goal\n\nIaC を進める"), false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	es, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 1 || es[0].Path != "career/goal.md" || es[0].Title != "Goal" {
		t.Fatalf("got %+v", es)
	}
	ms, err := s.Grep("iac")
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].Line != 3 || !strings.Contains(ms[0].Text, "IaC") {
		t.Fatalf("got %+v", ms)
	}
	if _, err := s.List("nope"); err == nil {
		t.Fatal("expected error for missing dir")
	}
}

func TestCommit(t *testing.T) {
	s := newRepo(t)
	rel, err := s.Write("a.md", []byte("x"), false)
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.Commit("nabu: write a.md", rel)
	if err != nil || !done {
		t.Fatalf("got %v, %v", done, err)
	}
	done, err = s.Commit("again", rel)
	if err != nil || done {
		t.Fatalf("second commit should be a no-op: %v, %v", done, err)
	}
	out, _ := exec.Command("git", "-C", s.Root, "log", "--format=%s").Output()
	if strings.TrimSpace(string(out)) != "nabu: write a.md" {
		t.Fatalf("log: %s", out)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "notes")
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@x")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@x")
	r, err := Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if !r.CreatedDir || !r.InitedGit || !r.Committed {
		t.Fatalf("first init: %+v", r)
	}
	r, err = Init(root)
	if err != nil {
		t.Fatal(err)
	}
	if r.CreatedDir || r.InitedGit || r.Committed {
		t.Fatalf("second init should change nothing: %+v", r)
	}
	if _, err := Open(root); err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command("git", "-C", root, "log", "--format=%s").Output()
	if strings.TrimSpace(string(out)) != "nabu: init" {
		t.Fatalf("log: %s", out)
	}
}

func TestReplaceRequiresExisting(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Replace("a.md", []byte("x")); err == nil {
		t.Fatal("expected ErrNotExist")
	}
	if _, err := s.Write("a.md", []byte("one"), false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Replace("a.md", []byte("two")); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Read("a.md")
	if string(b) != "two\n" {
		t.Fatalf("got %q", b)
	}
}

func TestMoveNeverOverwrites(t *testing.T) {
	s := newRepo(t)
	if _, _, err := s.Move("a.md", "b.md"); err == nil {
		t.Fatal("expected ErrNotExist for missing source")
	}
	if _, err := s.Write("a.md", []byte("one"), false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write("b.md", []byte("two"), false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Move("a.md", "b.md"); err == nil {
		t.Fatal("expected ErrExists for existing destination")
	}
	from, to, err := s.Move("a.md", "dir/c.md")
	if err != nil {
		t.Fatal(err)
	}
	if from != "a.md" || to != "dir/c.md" {
		t.Fatalf("from=%q to=%q", from, to)
	}
	if _, err := s.Read("a.md"); err == nil {
		t.Fatal("source should be gone")
	}
	b, _ := s.Read("dir/c.md")
	if string(b) != "one\n" {
		t.Fatalf("got %q", b)
	}
}

func TestCommitMove(t *testing.T) {
	s := newRepo(t)
	rel, _ := s.Write("a.md", []byte("one"), false)
	if _, err := s.Commit("write", rel); err != nil {
		t.Fatal(err)
	}
	from, to, err := s.Move("a.md", "b.md")
	if err != nil {
		t.Fatal(err)
	}
	done, err := s.Commit("mv", from, to)
	if err != nil || !done {
		t.Fatalf("done=%v err=%v", done, err)
	}
	out, _ := exec.Command("git", "-C", s.Root, "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("worktree not clean: %s", out)
	}
}

func TestLint(t *testing.T) {
	s := newRepo(t)
	for p, body := range map[string]string{
		"work/good-note.md": "# Good\n",
		"Bad Name.md":       "# Titled\n",
		"work/no-title.md":  "just text\n",
		"README.md":         "no title, but exempt\n",
	} {
		if _, err := s.Write(p, []byte(body), false); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := s.Lint()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range fs {
		got[f.Path] += f.Rule + ";"
	}
	if got["work/good-note.md"] != "" || got["README.md"] != "" {
		t.Fatalf("false positives: %v", got)
	}
	if got["Bad Name.md"] != "filename;" || got["work/no-title.md"] != "title;" {
		t.Fatalf("got %v", got)
	}
}
