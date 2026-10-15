package store

import (
	"maps"
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
		t.Fatal("expected error for a non-git notebook")
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
	if _, err := s.Write("a.md", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write("a.md", []byte("two")); err == nil {
		t.Fatal("expected ErrExists")
	}
	b, _ := s.Read("a.md")
	if string(b) != "one\n" {
		t.Fatalf("got %q", b)
	}
}

func TestListAndGrep(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Write("career/goal.md", []byte("# Goal\n\nIaC を進める")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}
	es, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 1 || es[0].Path != "career/goal.md" || es[0].Title != "Goal" {
		t.Fatalf("got %+v", es)
	}
	ms, err := s.Grep("", "iac")
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
	rel, err := s.Write("a.md", []byte("x"))
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
	out, _ := exec.Command("git", "-C", s.Dir, "log", "--format=%s").Output()
	if strings.TrimSpace(string(out)) != "nabu: write a.md" {
		t.Fatalf("log: %s", out)
	}
}

func TestInitIsIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "notes")
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@x")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@x")
	r, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !r.CreatedDir || !r.InitedGit || !r.Committed {
		t.Fatalf("first init: %+v", r)
	}
	r, err = Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.CreatedDir || r.InitedGit || r.Committed {
		t.Fatalf("second init should change nothing: %+v", r)
	}
	if _, err := Open(dir); err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command("git", "-C", dir, "log", "--format=%s").Output()
	if strings.TrimSpace(string(out)) != "nabu: init" {
		t.Fatalf("log: %s", out)
	}
}

func TestReplaceRequiresExisting(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Replace("a.md", []byte("x")); err == nil {
		t.Fatal("expected ErrNotExist")
	}
	if _, err := s.Write("a.md", []byte("one")); err != nil {
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
	if _, err := s.Write("a.md", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write("b.md", []byte("two")); err != nil {
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
	rel, _ := s.Write("a.md", []byte("one"))
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
	out, _ := exec.Command("git", "-C", s.Dir, "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("worktree not clean: %s", out)
	}
}

func TestRemoveRequiresExisting(t *testing.T) {
	s := newRepo(t)
	if _, err := s.Remove("a.md"); err == nil {
		t.Fatal("expected ErrNotExist")
	}
	rel, _ := s.Write("a.md", []byte("one"))
	if _, err := s.Commit("write", rel); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Remove("a.md"); err != nil {
		t.Fatal(err)
	}
	done, err := s.Commit("rm", rel)
	if err != nil || !done {
		t.Fatalf("done=%v err=%v", done, err)
	}
	out, _ := exec.Command("git", "-C", s.Dir, "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(out))) != 0 {
		t.Fatalf("worktree not clean: %s", out)
	}
}

func TestFindTask(t *testing.T) {
	s := newRepo(t)
	if s.FindTask("x") != "" {
		t.Fatal("found a task in an empty notebook")
	}
	if _, err := s.Write(TaskPath("done", "x"), []byte("# x")); err != nil {
		t.Fatal(err)
	}
	if got := s.FindTask("x"); got != "tasks/done/x.md" {
		t.Fatalf("got %q", got)
	}
}

func TestUpdatedIsTheLastCommitOfEachPath(t *testing.T) {
	s := newRepo(t)
	commit := func(when, rel, body string) {
		t.Helper()
		t.Setenv("GIT_COMMITTER_DATE", when)
		if err := os.MkdirAll(filepath.Dir(filepath.Join(s.Dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(s.Dir, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if ok, err := s.Commit("c", rel); err != nil || !ok {
			t.Fatalf("commit %s: %v %v", rel, ok, err)
		}
	}
	commit("2026-10-01T09:00:00+09:00", "tasks/inbox/a.md", "one")
	commit("2026-10-02T09:00:00+09:00", "tasks/inbox/b.md", "one")
	commit("2026-10-03T09:00:00+09:00", "tasks/inbox/a.md", "two")
	if _, err := s.Write("tasks/inbox/c.md", []byte("never committed")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Updated("tasks")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"tasks/inbox/a.md": "2026-10-03T09:00:00+09:00", "tasks/inbox/b.md": "2026-10-02T09:00:00+09:00"}
	if !maps.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
