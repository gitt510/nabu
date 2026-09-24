package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRoot returns an initialized notes root with a git identity, plus a
// runner that invokes nabu against it and fails the test on the wrong
// exit code. The returned string is stdout.
func newRoot(t *testing.T) (string, func(want int, stdin string, args ...string) string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	nabu := func(want int, stdin string, args ...string) string {
		t.Helper()
		var out, errb bytes.Buffer
		if code := run(append(args, "--root", dir), strings.NewReader(stdin), &out, &errb); code != want {
			t.Fatalf("%v: exit %d want %d: %s%s", args, code, want, out.String(), errb.String())
		}
		return out.String()
	}
	return dir, nabu
}

func gitClean(t *testing.T, dir string) {
	t.Helper()
	st, err := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if err != nil || len(bytes.TrimSpace(st)) != 0 {
		t.Fatalf("worktree not clean: %q %v", st, err)
	}
}

func TestParseInterspersed(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	j := fs.Bool("json", false, "")
	c := fs.String("content", "", "")
	if err := parseInterspersed(fs, []string{"a.md", "--json", "--content", "x y", "b"}); err != nil {
		t.Fatal(err)
	}
	if !*j || *c != "x y" || strings.Join(fs.Args(), ",") != "a.md,b" {
		t.Fatalf("json=%v content=%q args=%v", *j, *c, fs.Args())
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"note", "write", "-h"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out.String(), "usage: nabu note write") || errb.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errb.String())
	}
}

func TestUnknownCommandExits2(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"bogus"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d", code)
	}
}

func TestNoRootExits2(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errb bytes.Buffer
	if code := run([]string{"note", "ls"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "no root declared") {
		t.Fatalf("stderr=%q", errb.String())
	}
}

func TestDoctorReportsMissingConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errb bytes.Buffer
	if code := run([]string{"doctor"}, strings.NewReader(""), &out, &errb); code != exitFail {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "fail config") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestDoctorPassesOnRepo(t *testing.T) {
	_, nabu := newRoot(t)
	if got := nabu(exitOK, "", "doctor", "--json"); !strings.Contains(got, `"ok": true`) {
		t.Fatalf("stdout=%q", got)
	}
}

func TestDoctorNotesWarns(t *testing.T) {
	dir, nabu := newRoot(t)
	if err := os.WriteFile(filepath.Join(dir, "Bad Name.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := nabu(exitOK, "", "doctor", "--notes")
	if !strings.Contains(got, "warn notes") || !strings.Contains(got, "Bad Name.md") {
		t.Fatalf("stdout=%q", got)
	}
}

func TestMvOfUntrackedNoteCommits(t *testing.T) {
	dir, nabu := newRoot(t)
	// A note dropped into the root by hand is on disk but not in the index.
	if err := os.WriteFile(filepath.Join(dir, "Bad Name.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nabu(exitOK, "", "note", "mv", "Bad Name.md", "bad-name.md"); !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	gitClean(t, dir)
}

func TestNoteWriteReplaceMvLs(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitFail, "", "note", "replace", "a.md", "--content", "x")
	nabu(exitOK, "", "note", "write", "a.md", "--content", "# One\n\nbody")
	nabu(exitFail, "", "note", "write", "a.md", "--content", "again")
	if got := nabu(exitOK, "", "note", "replace", "a.md", "--content", "# Two"); !strings.Contains(got, `"action": "replace"`) {
		t.Fatalf("stdout=%q", got)
	}
	if got := nabu(exitOK, "", "note", "mv", "a.md", "b.md"); !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	if got := nabu(exitOK, "", "note", "read", "b.md"); got != "# Two\n" {
		t.Fatalf("read=%q", got)
	}
	if got := nabu(exitOK, "", "note", "ls"); !strings.HasPrefix(got, "b.md") || !strings.Contains(got, "Two") {
		t.Fatalf("ls=%q", got)
	}
	nabu(exitUsage, "", "note", "ls", "a", "b")
	gitClean(t, dir)
}

func TestTaskNewViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	for _, args := range [][]string{
		{"task", "new", "Bad Slug", "--content", "# x"},
		{"task", "new", "a/b", "--content", "# x"},
		{"task", "new", "ok", "--content", "# x", "--scheduled", "2026-09-15T18:00"},
		{"task", "new", "ok", "--content", "# x", "--ticket", "#7"},
		{"task", "new", "ok", "--content", "# x", "--ticket", "http://example.com/1"},
		{"task", "new", "ok", "--content", "---\na: b\n---\n# x", "--ticket", "https://example.com/1"},
	} {
		nabu(exitUsage, "", args...)
	}
	body := "# Retire legacy domain\n\n- [ ] delete it\n"
	args := []string{"task", "new", "retire-legacy",
		"--scheduled", "2026-09-15T18:00:00+09:00",
		"--ticket", "https://github.com/o/r/issues/7", "--ticket", "https://github.com/o/r/issues/8"}
	got := nabu(exitOK, body, args...)
	if !strings.Contains(got, `"path": "tasks/inbox/retire-legacy.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "retire-legacy.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nscheduled: \"2026-09-15T18:00:00+09:00\"\ntickets:\n  - \"https://github.com/o/r/issues/7\"\n  - \"https://github.com/o/r/issues/8\"\n---\n" + body
	if string(saved) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", saved, want)
	}
	nabu(exitFail, body, args...)
	nabu(exitOK, "", "task", "new", "plain", "--content", "# Plain\n")
	if saved, _ := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "plain.md")); string(saved) != "# Plain\n" {
		t.Fatalf("plain task got frontmatter: %q", saved)
	}
	if got := nabu(exitOK, "", "note", "ls", "tasks", "--json"); !strings.Contains(got, `"title": "Retire legacy domain"`) {
		t.Fatalf("frontmatter broke title extraction: %s", got)
	}
}

func TestTaskMvViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, "", "task", "new", "ship", "--content", "# Ship\n")
	for _, args := range [][]string{
		{"task", "mv", "ship", "progress"},
		{"task", "mv", "Bad Slug", "doing"},
		{"task", "mv", "ship"},
	} {
		nabu(exitUsage, "", args...)
	}
	nabu(exitFail, "", "task", "mv", "nope", "doing")
	nabu(exitFail, "", "task", "mv", "ship", "inbox")
	got := nabu(exitOK, "", "task", "mv", "ship", "doing")
	if !strings.Contains(got, `"from": "tasks/inbox/ship.md"`) || !strings.Contains(got, `"to": "tasks/doing/ship.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "inbox", "ship.md")); err == nil {
		t.Fatal("source still exists")
	}
	// the slug is taken whatever its status
	nabu(exitFail, "", "task", "new", "ship", "--content", "# Ship\n")
	nabu(exitOK, "", "task", "mv", "ship", "done")
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "-1", "--format=%s").Output()
	if strings.TrimSpace(string(log)) != "nabu: task mv tasks/doing/ship.md -> tasks/done/ship.md" {
		t.Fatalf("log: %s", log)
	}
}

func TestCanvasFlowViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, "", "init") // seeds .gitignore so the canvas stays out of git

	// read / write / diff / save need a draft
	nabu(exitFail, "", "canvas", "read")
	nabu(exitFail, "", "canvas", "write", "--content", "x")
	nabu(exitFail, "", "canvas", "save", "s")

	nabu(exitOK, "", "canvas", "open", "--content", "# Draft\n\nfirst")
	if got := nabu(exitOK, "", "canvas", "read"); got != "# Draft\n\nfirst\n" {
		t.Fatalf("read=%q", got)
	}
	if got := nabu(exitOK, "", "canvas", "diff"); got != "" {
		t.Fatalf("diff before edit=%q", got)
	}
	// a second open refuses; the canvas is not a note
	nabu(exitFail, "", "canvas", "open", "--content", "again")
	nabu(exitFail, "", "note", "write", "CANVAS.md", "--content", "x")
	nabu(exitFail, "", "note", "read", "CANVAS.md")
	if got := nabu(exitOK, "", "note", "ls"); strings.Contains(got, "CANVAS") {
		t.Fatalf("ls shows canvas: %q", got)
	}

	// the user edits by hand; diff shows it
	if err := os.WriteFile(filepath.Join(dir, "CANVAS.md"), []byte("# Draft\n\nfirst, edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nabu(exitOK, "", "canvas", "diff"); !strings.Contains(got, "+first, edited") || !strings.Contains(got, "-first") {
		t.Fatalf("diff=%q", got)
	}
	// the agent revises; the snapshot moves with it
	nabu(exitOK, "", "canvas", "write", "--content", "# Draft\n\nfirst, edited, polished")
	if got := nabu(exitOK, "", "canvas", "diff"); got != "" {
		t.Fatalf("diff after write=%q", got)
	}

	nabu(exitUsage, "", "canvas", "save", "Bad Slug")
	got := nabu(exitOK, "", "canvas", "save", "pr68-comment", "--ticket", "https://github.com/x/y/pull/68")
	if !strings.Contains(got, `"path": "writing/pr68-comment.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("save=%q", got)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "writing", "pr68-comment.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(saved), "---\ncreated: \"") || !strings.Contains(string(saved), "  - \"https://github.com/x/y/pull/68\"\n---\n# Draft\n\nfirst, edited, polished\n") {
		t.Fatalf("saved=%q", saved)
	}
	// the canvas file stays, empty, and nothing is left uncommitted
	if b, err := os.ReadFile(filepath.Join(dir, "CANVAS.md")); err != nil || len(b) != 0 {
		t.Fatalf("canvas after save=%q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".CANVAS.agent.md")); !os.IsNotExist(err) {
		t.Fatalf("snapshot left behind: %v", err)
	}
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "--format=%s").Output()
	if want := "nabu: canvas save writing/pr68-comment.md\nnabu: init\n"; string(log) != want {
		t.Fatalf("log=%q", log)
	}

	// saving again to the same slug replaces; a draft with its own frontmatter is refused
	nabu(exitOK, "", "canvas", "open", "--content", "---\nx: 1\n---\n# Two")
	nabu(exitFail, "", "canvas", "save", "pr68-comment")
	nabu(exitOK, "", "canvas", "drop")
	nabu(exitOK, "", "canvas", "open", "--content", "# Two")
	nabu(exitOK, "", "canvas", "save", "pr68-comment")
	if saved, _ := os.ReadFile(filepath.Join(dir, "writing", "pr68-comment.md")); !strings.HasSuffix(string(saved), "---\n# Two\n") {
		t.Fatalf("replaced=%q", saved)
	}
}

func TestInitSeedsGitignore(t *testing.T) {
	for _, kv := range [][2]string{{"GIT_AUTHOR_NAME", "t"}, {"GIT_AUTHOR_EMAIL", "t@x"}, {"GIT_COMMITTER_NAME", "t"}, {"GIT_COMMITTER_EMAIL", "t@x"}} {
		t.Setenv(kv[0], kv[1])
	}
	dir := filepath.Join(t.TempDir(), "root")
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", dir}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"committed": true`) {
		t.Fatalf("stdout=%q", out.String())
	}
	ig, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil || string(ig) != "CANVAS.md\n.CANVAS.agent.md\n" {
		t.Fatalf(".gitignore=%q %v", ig, err)
	}
}
