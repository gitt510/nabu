package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/gitt510/nabu/internal/store"
)

// created is the created line task new stamps under the clock newNotebook pins.
const created = "created: \"2026-10-06T09:00:00+09:00\"\narea: \"work\"\n"

// useNotebook points a fresh config file at dir for the rest of the test.
func useNotebook(t *testing.T, dir string, autoPush bool) {
	t.Helper()
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	b, err := json.Marshal(map[string]any{"notebook": dir, "auto_push": autoPush})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cfgDir, "nabu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "nabu", "config.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// newNotebook returns an initialized notebook with a git identity, declared
// in the config file, plus a runner that invokes nabu against it and fails
// the test on the wrong exit code. The returned string is stdout.
func newNotebook(t *testing.T) (string, func(want int, stdin string, args ...string) string) {
	t.Helper()
	dir := t.TempDir()
	now = func() time.Time { return time.Date(2026, 10, 6, 9, 0, 0, 0, time.FixedZone("JST", 9*60*60)) }
	t.Cleanup(func() { now = time.Now })
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	useNotebook(t, dir, false)
	nabu := func(want int, stdin string, args ...string) string {
		t.Helper()
		var out, errb bytes.Buffer
		if code := run(args, strings.NewReader(stdin), &out, &errb); code != want {
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

func TestTaskHelpGroupsCommands(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"task", "-h"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	got := out.String()
	at := func(s string) int {
		i := strings.Index(got, s)
		if i < 0 {
			t.Fatalf("task -h lacks %q:\n%s", s, got)
		}
		return i
	}
	if !(at("Browse and check:") < at("  ls ") && at("  ls ") < at("Read content:") &&
		at("Read content:") < at("  read ") && at("  read ") < at("Create and update:") && at("Create and update:") < at("  new ")) {
		t.Fatalf("commands are not under their groups:\n%s", got)
	}
}

func TestGroupAloneExits2(t *testing.T) {
	for _, args := range [][]string{{}, {"task"}, {"task", "bogus"}, {"task", "ls", "--nope"}, {"task", "read"}} {
		var out, errb bytes.Buffer
		if code := run(args, strings.NewReader(""), &out, &errb); code != exitUsage || out.Len() != 0 || !strings.Contains(errb.String(), "Usage:") {
			t.Fatalf("%v: exit %d stdout=%q stderr=%q", args, code, out.String(), errb.String())
		}
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"task", "new", "-h"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "nabu task new <slug> [flags]") || errb.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", out.String(), errb.String())
	}
}

func TestUnknownCommandExits2(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"bogus"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d", code)
	}
}

func TestNoNotebookExits2(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out, errb bytes.Buffer
	if code := run([]string{"task", "ls"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "no notebook declared") {
		t.Fatalf("stderr=%q", errb.String())
	}
}

func TestTaskNewViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	for _, args := range [][]string{
		{"task", "new", "Bad Slug", "--area", "work", "--content", "# x"},
		{"task", "new", "a/b", "--area", "work", "--content", "# x"},
		{"task", "new", "ok", "--area", "work", "--content", "# x", "--scheduled", "2026-09-15T18:00"},
		{"task", "new", "ok", "--area", "work", "--content", "# x", "--ticket", "#7"},
		{"task", "new", "ok", "--area", "work", "--content", "# x", "--ticket", "http://example.com/1"},
		{"task", "new", "ok", "--area", "work", "--content", "# x", "--pr", "http://example.com/1"},
		{"task", "new", "ok", "--area", "work", "--content", "# x", "--link", "example.com"},
		{"task", "new", "ok", "--area", "work", "--content", "---\na: b\n---\n# x", "--ticket", "https://example.com/1"},
		{"task", "new", "ok", "--content", shaped("Ok", "")},
		{"task", "new", "ok", "--area", "life", "--content", shaped("Ok", "")},
		{"task", "new", "ok", "--area", "work", "--project", "Bad Project", "--content", shaped("Ok", "")},
	} {
		nabu(exitUsage, "", args...)
	}
	body := shaped("Retire legacy domain", "- [ ] delete it\n")
	args := []string{"task", "new", "retire-legacy", "--area", "work",
		"--scheduled", "2026-09-15T18:00:00+09:00",
		"--ticket", "https://github.com/o/r/issues/7", "--ticket", "https://github.com/o/r/issues/8",
		"--link", "https://example.com/doc", "--pr", "https://github.com/o/r/pull/9"}
	got := nabu(exitOK, body, args...)
	if !strings.Contains(got, `"path": "tasks/inbox/retire-legacy.md"`) || !strings.Contains(got, `"committed": true`) || !strings.Contains(got, `"area": "work"`) || !strings.Contains(got, `"scheduled": "2026-09-15T18:00:00+09:00"`) {
		t.Fatalf("stdout=%q", got)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "retire-legacy.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "---\n" + created + "scheduled: \"2026-09-15T18:00:00+09:00\"\ntickets:\n  - \"https://github.com/o/r/issues/7\"\n  - \"https://github.com/o/r/issues/8\"\nprs:\n  - \"https://github.com/o/r/pull/9\"\nlinks:\n  - \"https://example.com/doc\"\n---\n" + body
	if string(saved) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", saved, want)
	}
	nabu(exitFail, body, args...)
	nabu(exitOK, "", "task", "new", "plain", "--area", "work", "--content", shaped("Plain", ""))
	if saved, _ := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "plain.md")); string(saved) != "---\n"+created+"---\n"+shaped("Plain", "") {
		t.Fatalf("task without flags: %q", saved)
	}
	if got := nabu(exitOK, "", "task", "ls", "--json"); !strings.Contains(got, `"title": "Retire legacy domain"`) {
		t.Fatalf("frontmatter broke title extraction: %s", got)
	}
}

func TestTaskMvViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	nabu(exitOK, "", "task", "new", "ship", "--area", "work", "--content", shaped("Ship", ""))
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
	nabu(exitFail, "", "task", "new", "ship", "--area", "work", "--content", shaped("Ship", ""))
	nabu(exitOK, "", "task", "mv", "ship", "done")
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "-1", "--format=%s").Output()
	if strings.TrimSpace(string(log)) != "nabu: task mv tasks/doing/ship.md -> tasks/done/ship.md" {
		t.Fatalf("log: %s", log)
	}
}

func TestTaskRenameViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	nabu(exitOK, "", "task", "new", "ship", "--area", "work", "--content", shaped("Ship", ""), "--ticket", "https://example.com/1")
	nabu(exitOK, "", "task", "new", "sail", "--area", "work", "--content", shaped("Sail", ""))
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
	nabu(exitOK, "", "task", "new", "ship", "--area", "work", "--content", shaped("Ship", "")) // the old slug is free again
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "-2", "--format=%s").Output()
	if !strings.Contains(string(log), "nabu: task rename tasks/doing/ship.md -> tasks/doing/launch.md") {
		t.Fatalf("log: %s", log)
	}
}

func TestTaskRmViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	nabu(exitOK, "", "task", "new", "ship", "--area", "work", "--content", shaped("Ship", ""))
	nabu(exitOK, "", "task", "mv", "ship", "done")
	nabu(exitUsage, "", "task", "rm")
	nabu(exitUsage, "", "task", "rm", "Bad Slug")
	nabu(exitFail, "", "task", "rm", "nope")
	got := nabu(exitOK, "", "task", "rm", "ship") // any status
	if !strings.Contains(got, `"path": "tasks/done/ship.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stdout=%q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "done", "ship.md")); err == nil {
		t.Fatal("task still exists")
	}
	gitClean(t, dir)
	log, _ := exec.Command("git", "-C", dir, "log", "-1", "--format=%s").Output()
	if !strings.Contains(string(log), "nabu: task rm tasks/done/ship.md") {
		t.Fatalf("log: %s", log)
	}
	nabu(exitOK, "", "task", "new", "ship", "--area", "work", "--content", shaped("Ship", "")) // the slug is free again
}

func TestInitSeedsReadme(t *testing.T) {
	for _, kv := range [][2]string{{"GIT_AUTHOR_NAME", "t"}, {"GIT_AUTHOR_EMAIL", "t@x"}, {"GIT_COMMITTER_NAME", "t"}, {"GIT_COMMITTER_EMAIL", "t@x"}} {
		t.Setenv(kv[0], kv[1])
	}
	dir := filepath.Join(t.TempDir(), "notebook")
	useNotebook(t, dir, false)
	var out, errb bytes.Buffer
	if code := run([]string{"init"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"committed": true`) {
		t.Fatalf("stdout=%q", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Fatal(err)
	}
}

func TestPushViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
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
	nabu(0, shaped("A", ""), "task", "new", "a", "--area", "work")
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
	dir, nabu := newNotebook(t)
	read := func() string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, "tasks", "inbox", "wait.md"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	nabu(exitOK, shaped("Wait", "- [ ] send\n"), "task", "new", "wait", "--area", "work", "--ticket", "https://example.com/1")
	nabu(exitOK, "", "task", "set", "wait", "--waiting", "担当者からの LINE 返信", "--scheduled", "2026-10-02T10:00:00+09:00", "--ticket", "https://example.com/2", "--ticket", "https://example.com/1", "--pr", "https://example.com/pull/3", "--link", "https://example.com/4")
	want := "---\n" + created + "scheduled: \"2026-10-02T10:00:00+09:00\"\nwaiting: \"担当者からの LINE 返信\"\ntickets:\n  - \"https://example.com/1\"\n  - \"https://example.com/2\"\nprs:\n  - \"https://example.com/pull/3\"\nlinks:\n  - \"https://example.com/4\"\n---\n" + shaped("Wait", "- [ ] send\n")
	if got := read(); got != want {
		t.Fatalf("after set:\n%s\nwant:\n%s", got, want)
	}
	// replace keeps the frontmatter, so it does not print one
	if got := nabu(exitOK, shaped("Wait", "- [x] send\n"), "task", "replace", "wait"); strings.Contains(got, `"frontmatter"`) {
		t.Fatalf("replace stdout=%q", got)
	}
	want = strings.Replace(want, "- [ ] send", "- [x] send", 1)
	if got := read(); got != want {
		t.Fatalf("after replace:\n%s\nwant:\n%s", got, want)
	}
	// set prints the frontmatter as written, so the caller can show it
	if got := nabu(exitOK, "", "task", "set", "wait", "--area", "personal", "--project", "nabu"); !strings.Contains(got, `"area": "personal"`) || !strings.Contains(got, `"project": "nabu"`) || !strings.Contains(got, `"created": "2026-10-06T09:00:00+09:00"`) {
		t.Fatalf("set stdout=%q", got)
	}
	want = strings.Replace(want, "area: \"work\"\n", "area: \"personal\"\nproject: \"nabu\"\n", 1)
	if got := read(); got != want {
		t.Fatalf("after area and project:\n%s\nwant:\n%s", got, want)
	}
	nabu(exitOK, "", "task", "set", "wait", "--area", "work", "--clear-project")
	want = strings.Replace(want, "area: \"personal\"\nproject: \"nabu\"\n", "area: \"work\"\n", 1)
	nabu(exitOK, "", "task", "set", "wait", "--clear-tickets", "--clear-links")
	if got := read(); got != "---\n"+created+"scheduled: \"2026-10-02T10:00:00+09:00\"\nwaiting: \"担当者からの LINE 返信\"\nprs:\n  - \"https://example.com/pull/3\"\n---\n"+shaped("Wait", "- [x] send\n") {
		t.Fatalf("after clearing two lists:\n%s", got)
	}
	// a date alone is the whole day; it is kept as written
	nabu(exitOK, "", "task", "set", "wait", "--scheduled", "2026-10-06")
	if got := read(); got != "---\n"+created+"scheduled: \"2026-10-06\"\nwaiting: \"担当者からの LINE 返信\"\nprs:\n  - \"https://example.com/pull/3\"\n---\n"+shaped("Wait", "- [x] send\n") {
		t.Fatalf("after a date-only scheduled:\n%s", got)
	}
	nabu(exitOK, "", "task", "set", "wait", "--clear-waiting", "--clear-scheduled", "--clear-prs")
	// created is not a flag, so clearing every flag leaves it
	if got := read(); got != "---\n"+created+"---\n"+shaped("Wait", "- [x] send\n") {
		t.Fatalf("after clear: %q", got)
	}
	for _, args := range [][]string{
		{"task", "set", "wait"},
		{"task", "set", "wait", "--waiting", "x", "--clear-waiting"},
		{"task", "set", "wait", "--waiting", "  "},
		{"task", "set", "wait", "--area", "life"},
		{"task", "set", "wait", "--project", "Nabu"},
		{"task", "set", "wait", "--project", "x", "--clear-project"},
		{"task", "set", "wait", "--scheduled", "2026-10-02T10:00"},
		{"task", "set", "wait", "--scheduled", "2026-10-2"},
		{"task", "set", "wait", "--ticket", "http://example.com/1"},
		{"task", "set", "wait", "--pr", "http://example.com/1"},
		{"task", "set", "wait", "--link", "ftp://example.com/1"},
		{"task", "replace", "wait", "--content", "---\nwaiting: \"x\"\n---\n# Wait\n"},
		{"task", "new", "other", "--area", "work", "--content", "---\nwaiting: \"x\"\n---\n# Other\n"},
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
	dir, nabu := newNotebook(t)
	nabu(exitOK, shaped("Wait", ""), "task", "new", "wait", "--area", "work")
	// a task dropped into tasks/ by hand is on disk but not in the index
	if err := os.WriteFile(filepath.Join(dir, "tasks", "stray.md"), []byte("# Stray\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nabu(exitFail, shaped("x", ""), "task", "new", "stray", "--area", "work")
	if got := nabu(exitOK, "", "task", "ls"); !strings.Contains(got, "stray\tStray\t-\n") {
		t.Fatalf("task ls stray: %q", got)
	}
	got := nabu(exitOK, "", "task", "mv", "stray", "inbox")
	if !strings.Contains(got, `"from": "tasks/stray.md"`) || !strings.Contains(got, `"committed": true`) {
		t.Fatalf("stray mv: %s", got)
	}
	gitClean(t, dir)
}

func TestTaskTable(t *testing.T) {
	rows := []taskEntry{
		{Status: "inbox", Title: "検索UI", Frontmatter: meta{Created: "2026-10-06T09:00:00+09:00"}},
		{Status: "doing", Title: "ssh"},
	}
	want := `╭────────┬────────┬────────────╮
│ STATUS │ TITLE  │ CREATED    │
├────────┼────────┼────────────┤
│ inbox  │ 検索UI │ 2026-10-06 │
│ doing  │ ssh    │ -          │
╰────────┴────────┴────────────╯`
	if got := ansi.Strip(taskTable(rows, 0)); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	for _, line := range strings.Split(ansi.Strip(taskTable(rows, 28)), "\n") {
		if w := ansi.StringWidth(line); w > 28 {
			t.Fatalf("line is %d cells wide, over 28: %s", w, line)
		}
	}
}

func TestGrepAndValidateTables(t *testing.T) {
	got := ansi.Strip(grepTable([]store.Match{{Path: "tasks/doing/wait.md", Line: 15, Text: "Ping them"}}, "ping", 0))
	want := `╭────────┬──────┬──────┬───────────╮
│ STATUS │ TASK │ LINE │ TEXT      │
├────────┼──────┼──────┼───────────┤
│ doing  │ wait │   15 │ Ping them │
╰────────┴──────┴──────┴───────────╯`
	if got != want {
		t.Fatalf("grep:\n%s\nwant:\n%s", got, want)
	}
	if h := highlight("Ping them, ping", "PING"); ansi.Strip(h) != "Ping them, ping" || strings.Count(h, "\x1b[") < 2 {
		t.Fatalf("highlight: %q", h)
	}
	got = ansi.Strip(validateTable([]verdict{{Path: "tasks/inbox/ok.md"}, {Path: "tasks/x.md", Error: "no ## Situation"}}, 0))
	want = `╭────────┬────────┬──────┬─────────────────╮
│ RESULT │ STATUS │ TASK │ PROBLEM         │
├────────┼────────┼──────┼─────────────────┤
│ ok     │ inbox  │ ok   │                 │
│ bad    │ stray  │ x    │ no ## Situation │
╰────────┴────────┴──────┴─────────────────╯`
	if got != want {
		t.Fatalf("validate:\n%s\nwant:\n%s", got, want)
	}
}

func TestTaskReadLsGrepViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	nabu(exitOK, shaped("Wait", "ping them\n"), "task", "new", "wait", "--area", "work", "--ticket", "https://example.com/1")
	nabu(exitOK, shaped("Other", ""), "task", "new", "other", "--area", "work")
	nabu(exitOK, "", "task", "mv", "wait", "doing")
	nabu(exitOK, "", "task", "set", "wait", "--waiting", "their reply")
	// a file outside tasks/ is not a task and stays out of task grep
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "work", "free.md"), []byte("# Free\n\nping them\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nabu(exitOK, "", "task", "read", "wait"); got != "---\n"+created+"waiting: \"their reply\"\ntickets:\n  - \"https://example.com/1\"\n---\n"+shaped("Wait", "ping them\n") {
		t.Fatalf("read: %q", got)
	}
	nabu(exitFail, "", "task", "read", "missing")
	if got := nabu(exitOK, "", "task", "ls"); got != "inbox\tOther\t2026-10-06\ndoing\tWait\t2026-10-06\n" {
		t.Fatalf("ls: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "inbox"); got != "inbox\tOther\t2026-10-06\n" {
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
	if got := nabu(exitOK, "", "task", "grep", "PING"); got != "tasks/doing/wait.md:18: ping them\n" {
		t.Fatalf("grep: %q", got)
	}
	if got := nabu(exitOK, "", "task", "grep", "nothing-here", "--json"); strings.TrimSpace(got) != "[]" {
		t.Fatalf("task grep --json: %q", got)
	}
}

// shaped is the smallest body task new and task replace accept: a title,
// the four sections, and answer under the last one.
func shaped(title, answer string) string {
	return "# " + title + "\n\n## Situation\n\n## Complication\n\n## Question\n\n## Answer\n\n" + answer
}

func TestTaskShape(t *testing.T) {
	_, nabu := newNotebook(t)
	head := "# T\n\n## Situation\n\n- 本番と検証が旧 runtime\n\n## Complication\n\n"
	tail := "\n## Question\n\n- 期限前に止めずに移れる？\n\n## Answer\n\n"
	for i, body := range []string{
		head + "- " + strings.Repeat("あ", 28) + "\n### sub\n" + tail + "- 未定\n",
		head + tail,
		"# " + strings.Repeat("a", 40) + "\n## Situation\n## Complication\n## Question\n## Answer\n",
		head + tail + "## Why dropped\n\n- 主導権が自分にない\n",
	} {
		nabu(exitOK, body, "task", "new", "ok"+strconv.Itoa(i), "--area", "work")
	}
	for _, body := range []string{
		"# T\n\n- [ ] x\n",
		head + "- x\n",
		"# T\n\n## Complication\n\n## Situation\n\n## Question\n\n## Answer\n",
		head + tail + "---\n\n## AI memo\n\nfree prose\n",
		head + "## Note\n" + tail,
		head + tail + "## Answer\n",
		head + strings.Repeat("あ", 31) + "\n" + tail,
		head + "- [ ] " + strings.Repeat("a", 25) + "\n" + tail,
		head + "### " + strings.Repeat("a", 27) + "\n" + tail,
	} {
		nabu(exitUsage, body, "task", "new", "bad", "--area", "work")
		nabu(exitUsage, body, "task", "replace", "ok0")
	}
	if got := nabu(exitUsage, head+strings.Repeat("あ", 31)+"\n"+tail, "task", "new", "bad", "--area", "work"); got != "" {
		t.Fatalf("stdout on refusal: %q", got)
	}
}

func TestTaskValidateViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	nabu(exitOK, shaped("Ok", ""), "task", "new", "ok", "--area", "work")
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
	if code := run([]string{"task", "validate"}, strings.NewReader(""), &out, &errb); code != exitFail {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if got := out.String(); got != "ok    tasks/inbox/ok.md\nbad   tasks/doing/old.md: body has no \"## Situation\" section\n" {
		t.Fatalf("all: %q", got)
	}
	out.Reset()
	if code := run([]string{"task", "validate", "old", "--json"}, strings.NewReader(""), &out, &errb); code != exitFail {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), `"error": "body has no`) {
		t.Fatalf("json: %q", out.String())
	}
	nabu(exitFail, "", "task", "validate", "missing")
	nabu(exitUsage, "", "task", "validate", "Bad Slug")
	// a shaped body without an area is out of shape too
	if err := os.WriteFile(filepath.Join(dir, "tasks", "doing", "old.md"), []byte(shaped("Old", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := nabu(exitFail, "", "task", "validate", "old"); !strings.Contains(got, "bad   tasks/doing/old.md: frontmatter has no area") {
		t.Fatalf("no area: %q", got)
	}
}

func TestTaskDroppedViaCLI(t *testing.T) {
	dir, nabu := newNotebook(t)
	empty := shaped("T", "")
	why := empty + "## Why dropped\n\n- 主導権が自分にない\n"
	nabu(exitOK, shaped("T", ""), "task", "new", "t", "--area", "work")

	// mv to dropped is refused until the body says why
	var out, errb bytes.Buffer
	if code := run([]string{"task", "mv", "t", "dropped"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), `needs a "## Why dropped" section`) || !strings.Contains(errb.String(), "task replace first") {
		t.Fatalf("stderr: %q", errb.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "tasks", "inbox", "t.md")); err != nil {
		t.Fatalf("task moved despite refusal: %v", err)
	}

	// a heading with nothing under it is not a reason
	nabu(exitOK, empty+"## Why dropped\n", "task", "replace", "t")
	if code := run([]string{"task", "mv", "t", "dropped"}, strings.NewReader(""), &out, &errb); code != exitUsage {
		t.Fatalf("exit %d: %s", code, errb.String())
	}

	nabu(exitOK, why, "task", "replace", "t")
	got := nabu(exitOK, "", "task", "mv", "t", "dropped")
	if !strings.Contains(got, `"to": "tasks/dropped/t.md"`) {
		t.Fatalf("mv: %q", got)
	}
	if got := nabu(exitOK, "", "task", "ls", "dropped"); got != "dropped\tT\t2026-10-06\n" {
		t.Fatalf("ls: %q", got)
	}

	// once dropped, a body without the reason is refused by replace and reported by validate
	nabu(exitUsage, shaped("T", ""), "task", "replace", "t")
	if got := nabu(exitOK, "", "task", "validate", "t"); got != "ok    tasks/dropped/t.md\n" {
		t.Fatalf("validate: %q", got)
	}

	// the heading is allowed, not required, in any other status
	nabu(exitOK, why, "task", "new", "u", "--area", "work")
	gitClean(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "tasks", "dropped", "t.md"), []byte(shaped("T", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	nabu(exitFail, "", "task", "validate", "t")
}

func TestAutoPushReportsEachCommit(t *testing.T) {
	dir, nabu := newNotebook(t)
	useNotebook(t, dir, true)

	// a failed push is reported, and the commit stands
	out := nabu(0, shaped("A", ""), "task", "new", "a", "--area", "work")
	if !strings.Contains(out, `"committed": true`) || !strings.Contains(out, `"push": "failed: no upstream branch`) {
		t.Fatalf("new output: %s", out)
	}

	bare := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "--bare", bare},
		{"-C", dir, "remote", "add", "origin", bare},
		{"-C", dir, "push", "-q", "-u", "origin", "main"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	out = nabu(0, "", "task", "mv", "a", "doing")
	if !strings.Contains(out, `"push": "ok"`) {
		t.Fatalf("mv output: %s", out)
	}
	log, _ := exec.Command("git", "-C", bare, "log", "-1", "--format=%s", "main").Output()
	if !strings.Contains(string(log), "nabu: task mv tasks/inbox/a.md -> tasks/doing/a.md") {
		t.Fatalf("remote log: %s", log)
	}
}

func TestPublishPushesThenRunsTheWorkflow(t *testing.T) {
	dir, nabu := newNotebook(t)
	bare := t.TempDir()
	// a stand-in gh that records where and how it ran and what the remote held
	// by then, prints a run URL, and fails when told to
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")
	script := "#!/bin/sh\n" +
		"[ -f " + filepath.Join(bin, "fail") + " ] && { echo 'HTTP 404: workflow not found' >&2; exit 1; }\n" +
		"echo \"$(pwd -P) $* remote=$(git -C " + bare + " log -1 --format=%s main)\" >> " + calls + "\n" +
		"echo https://github.com/o/notes/actions/runs/1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	nabu(exitFail, "", "publish") // no upstream yet: nothing is started
	if _, err := os.Stat(calls); err == nil {
		t.Fatal("gh ran without a push")
	}

	nabu(exitOK, shaped("A", ""), "task", "new", "a", "--area", "work")
	for _, args := range [][]string{
		{"init", "-q", "--bare", bare},
		{"-C", dir, "remote", "add", "origin", bare},
		{"-C", dir, "push", "-q", "-u", "origin", "main"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	nabu(exitOK, "", "task", "mv", "a", "doing")
	out := nabu(exitOK, "", "publish")
	if !strings.Contains(out, `"run": "https://github.com/o/notes/actions/runs/1"`) || !strings.Contains(out, `"ref": "main"`) {
		t.Fatalf("publish output: %s", out)
	}
	// gh ran in the notebook, on its branch, after the move had reached the remote
	got, _ := os.ReadFile(calls)
	real, _ := filepath.EvalSymlinks(dir)
	if want := real + " workflow run publish.yml --ref main remote=nabu: task mv tasks/inbox/a.md -> tasks/doing/a.md\n"; string(got) != want {
		t.Fatalf("gh calls %q want %q", got, want)
	}

	// a run that cannot start is reported after the push that landed
	if err := os.WriteFile(filepath.Join(bin, "fail"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"publish"}, strings.NewReader(""), &stdout, &stderr); code != exitFail {
		t.Fatalf("exit %d", code)
	}
	if msg := stderr.String(); !strings.Contains(msg, "pushed to origin/main, but the publish run did not start") || !strings.Contains(msg, "HTTP 404: workflow not found") {
		t.Fatalf("stderr: %s", msg)
	}
}
