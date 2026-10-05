package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// newRoot returns an initialized root with a git identity, plus a
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
	if code := run([]string{"task", "new", "-h"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out.String(), "usage: nabu task new") || errb.Len() != 0 {
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
	if code := run([]string{"task", "ls"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "no root declared") {
		t.Fatalf("stderr=%q", errb.String())
	}
}

func TestTaskNewViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	for _, args := range [][]string{
		{"task", "new", "Bad Slug", "--content", "# x"},
		{"task", "new", "a/b", "--content", "# x"},
		{"task", "new", "ok", "--content", "# x", "--scheduled", "2026-09-15T18:00"},
		{"task", "new", "ok", "--content", "# x", "--ticket", "#7"},
		{"task", "new", "ok", "--content", "# x", "--ticket", "http://example.com/1"},
		{"task", "new", "ok", "--content", "# x", "--pr", "http://example.com/1"},
		{"task", "new", "ok", "--content", "# x", "--link", "example.com"},
		{"task", "new", "ok", "--content", "---\na: b\n---\n# x", "--ticket", "https://example.com/1"},
	} {
		nabu(exitUsage, "", args...)
	}
	body := shaped("Retire legacy domain", "- [ ] delete it\n")
	args := []string{"task", "new", "retire-legacy",
		"--scheduled", "2026-09-15T18:00:00+09:00",
		"--ticket", "https://github.com/o/r/issues/7", "--ticket", "https://github.com/o/r/issues/8",
		"--link", "https://example.com/doc", "--pr", "https://github.com/o/r/pull/9"}
	got := nabu(exitOK, body, args...)
	if !strings.Contains(got, `"path": "tasks/inbox/retire-legacy.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "retire-legacy.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nscheduled: \"2026-09-15T18:00:00+09:00\"\ntickets:\n  - \"https://github.com/o/r/issues/7\"\n  - \"https://github.com/o/r/issues/8\"\nprs:\n  - \"https://github.com/o/r/pull/9\"\nlinks:\n  - \"https://example.com/doc\"\n---\n" + body
	if string(saved) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", saved, want)
	}
	nabu(exitFail, body, args...)
	nabu(exitOK, "", "task", "new", "plain", "--content", shaped("Plain", ""))
	if saved, _ := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "plain.md")); string(saved) != shaped("Plain", "") {
		t.Fatalf("plain task got frontmatter: %q", saved)
	}
	if got := nabu(exitOK, "", "task", "ls", "--json"); !strings.Contains(got, `"title": "Retire legacy domain"`) {
		t.Fatalf("frontmatter broke title extraction: %s", got)
	}
}

func TestTaskMvViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, "", "task", "new", "ship", "--content", shaped("Ship", ""))
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
	nabu(exitFail, "", "task", "new", "ship", "--content", shaped("Ship", ""))
	nabu(exitOK, "", "task", "mv", "ship", "done")
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "-1", "--format=%s").Output()
	if strings.TrimSpace(string(log)) != "nabu: task mv tasks/doing/ship.md -> tasks/done/ship.md" {
		t.Fatalf("log: %s", log)
	}
}

func TestTaskRenameViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, "", "task", "new", "ship", "--content", shaped("Ship", ""), "--ticket", "https://example.com/1")
	nabu(exitOK, "", "task", "new", "sail", "--content", shaped("Sail", ""))
	nabu(exitOK, "", "task", "mv", "ship", "doing")
	for _, args := range [][]string{
		{"task", "rename", "ship"},
		{"task", "rename", "ship", "Bad Slug"},
		{"task", "rename", "ship", "ship"},
	} {
		nabu(exitUsage, "", args...)
	}
	nabu(exitFail, "", "task", "rename", "nope", "launch")
	nabu(exitFail, "", "task", "rename", "ship", "sail") // taken, whatever its status
	got := nabu(exitOK, "", "task", "rename", "ship", "launch")
	if !strings.Contains(got, `"from": "tasks/doing/ship.md"`) || !strings.Contains(got, `"to": "tasks/doing/launch.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "doing", "ship.md")); err == nil {
		t.Fatal("source still exists")
	}
	read := nabu(exitOK, "", "task", "read", "launch")
	if !strings.Contains(read, "https://example.com/1") || !strings.Contains(read, "# Ship") {
		t.Fatalf("content changed: %q", read)
	}
	nabu(exitOK, "", "task", "new", "ship", "--content", shaped("Ship", "")) // the old slug is free again
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "-2", "--format=%s").Output()
	if !strings.Contains(string(log), "nabu: task rename tasks/doing/ship.md -> tasks/doing/launch.md") {
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
	// a second open refuses
	nabu(exitFail, "", "canvas", "open", "--content", "again")

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
	if !strings.Contains(got, `"path": "gallery/pr68-comment.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("save=%q", got)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "gallery", "pr68-comment.md"))
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
	if want := "nabu: canvas save gallery/pr68-comment.md\nnabu: init\n"; string(log) != want {
		t.Fatalf("log=%q", log)
	}

	// saving again to the same slug replaces; a draft with its own frontmatter is refused
	nabu(exitOK, "", "canvas", "open", "--content", "---\nx: 1\n---\n# Two")
	nabu(exitFail, "", "canvas", "save", "pr68-comment")
	nabu(exitOK, "", "canvas", "drop")
	nabu(exitOK, "", "canvas", "open", "--content", "# Two")
	nabu(exitOK, "", "canvas", "save", "pr68-comment")
	if saved, _ := os.ReadFile(filepath.Join(dir, "gallery", "pr68-comment.md")); !strings.HasSuffix(string(saved), "---\n# Two\n") {
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

func TestPushViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(1, "", "push") // no upstream yet

	bare := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "--bare", bare},
		{"-C", dir, "remote", "add", "origin", bare},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	nabu(0, shaped("A", ""), "task", "new", "a")
	if out, err := exec.Command("git", "-C", dir, "push", "-q", "-u", "origin", "main").CombinedOutput(); err != nil {
		t.Fatalf("git push -u: %s", out)
	}

	nabu(0, "", "task", "mv", "a", "doing")
	out := nabu(0, "", "push")
	if !strings.Contains(out, `"upstream": "origin/main"`) {
		t.Fatalf("push output: %s", out)
	}
	log, _ := exec.Command("git", "-C", bare, "log", "--format=%s", "main").Output()
	if !strings.Contains(string(log), "nabu: task mv tasks/inbox/a.md -> tasks/doing/a.md") {
		t.Fatalf("remote log: %s", log)
	}
}

func TestTaskReplaceAndSetViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	read := func() string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "wait.md"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	nabu(exitOK, shaped("Wait", "- [ ] send\n"), "task", "new", "wait", "--ticket", "https://example.com/1")
	nabu(exitOK, "", "task", "set", "wait", "--waiting", "担当者からの LINE 返信", "--scheduled", "2026-10-02T10:00:00+09:00", "--ticket", "https://example.com/2", "--ticket", "https://example.com/1", "--pr", "https://example.com/pull/3", "--link", "https://example.com/4")
	want := "---\nscheduled: \"2026-10-02T10:00:00+09:00\"\nwaiting: \"担当者からの LINE 返信\"\ntickets:\n  - \"https://example.com/1\"\n  - \"https://example.com/2\"\nprs:\n  - \"https://example.com/pull/3\"\nlinks:\n  - \"https://example.com/4\"\n---\n" + shaped("Wait", "- [ ] send\n")
	if got := read(); got != want {
		t.Fatalf("after set:\n%s\nwant:\n%s", got, want)
	}
	nabu(exitOK, shaped("Wait", "- [x] send\n"), "task", "replace", "wait")
	want = strings.Replace(want, "- [ ] send", "- [x] send", 1)
	if got := read(); got != want {
		t.Fatalf("after replace:\n%s\nwant:\n%s", got, want)
	}
	nabu(exitOK, "", "task", "set", "wait", "--clear-tickets", "--clear-links")
	if got := read(); got != "---\nscheduled: \"2026-10-02T10:00:00+09:00\"\nwaiting: \"担当者からの LINE 返信\"\nprs:\n  - \"https://example.com/pull/3\"\n---\n"+shaped("Wait", "- [x] send\n") {
		t.Fatalf("after clearing two lists:\n%s", got)
	}
	// a date alone is the whole day; it is kept as written
	nabu(exitOK, "", "task", "set", "wait", "--scheduled", "2026-10-06")
	if got := read(); got != "---\nscheduled: \"2026-10-06\"\nwaiting: \"担当者からの LINE 返信\"\nprs:\n  - \"https://example.com/pull/3\"\n---\n"+shaped("Wait", "- [x] send\n") {
		t.Fatalf("after a date-only scheduled:\n%s", got)
	}
	nabu(exitOK, "", "task", "set", "wait", "--clear-waiting", "--clear-scheduled", "--clear-prs")
	if got := read(); got != shaped("Wait", "- [x] send\n") {
		t.Fatalf("after clear: %q", got)
	}
	for _, args := range [][]string{
		{"task", "set", "wait"},
		{"task", "set", "wait", "--waiting", "x", "--clear-waiting"},
		{"task", "set", "wait", "--waiting", "  "},
		{"task", "set", "wait", "--scheduled", "2026-10-02T10:00"},
		{"task", "set", "wait", "--scheduled", "2026-10-2"},
		{"task", "set", "wait", "--ticket", "http://example.com/1"},
		{"task", "set", "wait", "--pr", "http://example.com/1"},
		{"task", "set", "wait", "--link", "ftp://example.com/1"},
		{"task", "replace", "wait", "--content", "---\nwaiting: \"x\"\n---\n# Wait\n"},
		{"task", "new", "other", "--content", "---\nwaiting: \"x\"\n---\n# Other\n"},
	} {
		nabu(exitUsage, "", args...)
	}
	nabu(exitFail, "", "task", "set", "missing", "--waiting", "x")
	nabu(exitFail, shaped("x", ""), "task", "replace", "missing")
	if err := os.WriteFile(filepath.Join(dir, "tasks", "inbox", "wait.md"), []byte("---\nwating: \"typo\"\n---\n# Wait\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nabu(exitFail, "", "task", "set", "wait", "--waiting", "x")
	nabu(exitFail, shaped("Wait", ""), "task", "replace", "wait")
}

func TestTaskStray(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, shaped("Wait", ""), "task", "new", "wait")
	// a task dropped into tasks/ by hand is on disk but not in the index
	if err := os.WriteFile(filepath.Join(dir, "tasks", "stray.md"), []byte("# Stray\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nabu(exitFail, shaped("x", ""), "task", "new", "stray")
	if got := nabu(exitOK, "", "task", "ls"); !strings.Contains(got, "stray  stray  Stray") {
		t.Fatalf("task ls stray: %q", got)
	}
	got := nabu(exitOK, "", "task", "mv", "stray", "inbox")
	if !strings.Contains(got, `"from": "tasks/stray.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stray mv: %s", got)
	}
	gitClean(t, dir)
}

func TestTaskReadLsGrepViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, shaped("Wait", "ping them\n"), "task", "new", "wait", "--ticket", "https://example.com/1")
	nabu(exitOK, shaped("Other", ""), "task", "new", "other")
	nabu(exitOK, "", "task", "mv", "wait", "doing")
	nabu(exitOK, "", "task", "set", "wait", "--waiting", "their reply")
	// a file outside tasks/ is not a task and stays out of task grep
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work", "free.md"), []byte("# Free\n\nping them\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nabu(exitOK, "", "task", "read", "wait"); got != "---\nwaiting: \"their reply\"\ntickets:\n  - \"https://example.com/1\"\n---\n"+shaped("Wait", "ping them\n") {
		t.Fatalf("read: %q", got)
	}
	nabu(exitFail, "", "task", "read", "missing")
	if got := nabu(exitOK, "", "task", "ls"); got != "doing  wait   Wait   waiting: their reply\ninbox  other  Other\n" {
		t.Fatalf("ls: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "inbox"); got != "inbox  other  Other\n" {
		t.Fatalf("ls inbox: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "dropped"); got != "" {
		t.Fatalf("ls of a status with no folder yet: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "dropped", "--json"); got != "[]\n" {
		t.Fatalf("ls --json of a status with no folder yet: %q", got)
	}
	nabu(exitUsage, "", "task", "ls", "later")
	got := nabu(exitOK, "", "task", "ls", "doing", "--json")
	for _, want := range []string{`"slug": "wait"`, `"status": "doing"`, `"frontmatter": {`, `"waiting": "their reply"`, `"https://example.com/1"`, `"body": "# Wait`} {
		if !strings.Contains(got, want) {
			t.Fatalf("ls --json lacks %s: %s", want, got)
		}
	}
	if strings.Contains(got, `"scheduled"`) {
		t.Fatalf("ls --json prints empty scheduled: %s", got)
	}
	if got := nabu(exitOK, "", "task", "grep", "PING"); got != "tasks/doing/wait.md:14: ping them\n" {
		t.Fatalf("grep: %q", got)
	}
	if got := nabu(exitOK, "", "task", "grep", "nothing-here", "--json"); strings.TrimSpace(got) != "[]" {
		t.Fatalf("task grep --json: %q", got)
	}
}

// shaped is the smallest body task new and task replace accept: a title, an
// empty For Human, the rule, and memo as the AI memo.
func shaped(title, memo string) string {
	return "# " + title + "\n\n## For Human\n\n---\n\n## AI memo\n\n" + memo
}

func TestTaskShape(t *testing.T) {
	_, nabu := newRoot(t)
	head := "# T\n\n## For Human\n\n"
	tail := "\n---\n\n## AI memo\n\nfree *prose*, any length, --- lines too\n---\n"
	for i, body := range []string{
		head + "### Now\n\n- 鍵は 1Password 経由、disk に無い\n\n### Next\n\n- [ ] 蓋を閉じても tailnet で届く設定\n- [x] " + strings.Repeat("あ", 24) + tail,
		head + "a sentence outside a bullet" + tail,
		head + "## Now\n" + tail,
		"# T\n## For Human\n---\n## AI memo\n",
		head + tail + "\n## For Human\n",
	} {
		nabu(exitOK, body, "task", "new", "ok"+strconv.Itoa(i))
	}
	for _, body := range []string{
		"# T\n\n- [ ] x\n",
		head + "- x\n",
		"# T\n\n## AI memo\n\n---\n\n## For Human\n\n- x\n",
		head + "- x\n\n## AI memo\n",
		head + strings.Repeat("あ", 31) + tail,
		head + "- [ ] " + strings.Repeat("a", 25) + tail,
		head + "### " + strings.Repeat("a", 27) + tail,
	} {
		nabu(exitUsage, body, "task", "new", "bad")
		nabu(exitUsage, body, "task", "replace", "ok0")
	}
	if got := nabu(exitUsage, head+strings.Repeat("あ", 31)+tail, "task", "new", "bad"); got != "" {
		t.Fatalf("stdout on refusal: %q", got)
	}
}

func TestTaskValidateViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	nabu(exitOK, shaped("Ok", ""), "task", "new", "ok")
	if err := os.MkdirAll(filepath.Join(dir, "tasks", "doing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "tasks", "doing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "doing", "old.md"), []byte("# Old\n\n- [ ] x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nabu(exitOK, "", "task", "validate", "ok"); got != "ok    tasks/inbox/ok.md\n" {
		t.Fatalf("one: %q", got)
	}
	var out, errb bytes.Buffer
	if code := run([]string{"task", "validate", "--root", dir}, strings.NewReader(""), &out, &errb); code != exitFail {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := out.String(); got != "bad   tasks/doing/old.md: body has no \"## For Human\" section\nok    tasks/inbox/ok.md\n" {
		t.Fatalf("all: %q", got)
	}
	out.Reset()
	if code := run([]string{"task", "validate", "old", "--json", "--root", dir}, strings.NewReader(""), &out, &errb); code != exitFail {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), `"error": "body has no`) {
		t.Fatalf("json: %q", out.String())
	}
	nabu(exitFail, "", "task", "validate", "missing")
	nabu(exitUsage, "", "task", "validate", "Bad Slug")
}

func TestTaskDroppedViaCLI(t *testing.T) {
	dir, nabu := newRoot(t)
	head := "# T\n\n## For Human\n\n"
	tail := "\n---\n\n## AI memo\n\nlong story\n"
	why := head + "### Why dropped\n\n- 主導権が自分にない\n" + tail
	nabu(exitOK, shaped("T", ""), "task", "new", "t")

	// mv to dropped is refused until the body says why
	var out, errb bytes.Buffer
	if code := run([]string{"task", "mv", "t", "dropped", "--root", dir}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), `needs a "### Why dropped" heading`) || !strings.Contains(errb.String(), "task replace first") {
		t.Fatalf("stderr: %q", errb.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "inbox", "t.md")); err != nil {
		t.Fatalf("task moved despite refusal: %v", err)
	}

	// a heading with nothing under it is not a reason
	nabu(exitOK, head+"### Why dropped\n"+tail, "task", "replace", "t")
	if code := run([]string{"task", "mv", "t", "dropped", "--root", dir}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}

	nabu(exitOK, why, "task", "replace", "t")
	got := nabu(exitOK, "", "task", "mv", "t", "dropped")
	if !strings.Contains(got, `"to": "tasks/dropped/t.md"`) {
		t.Fatalf("mv: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "dropped"); got != "dropped  t  T\n" {
		t.Fatalf("ls: %q", got)
	}

	// once dropped, a body without the reason is refused by replace and reported by validate
	nabu(exitUsage, shaped("T", ""), "task", "replace", "t")
	if got := nabu(exitOK, "", "task", "validate", "t"); got != "ok    tasks/dropped/t.md\n" {
		t.Fatalf("validate: %q", got)
	}

	// the heading is allowed, not required, in any other status
	nabu(exitOK, why, "task", "new", "u")
	gitClean(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "tasks", "dropped", "t.md"), []byte(shaped("T", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	nabu(exitFail, "", "task", "validate", "t")
}
