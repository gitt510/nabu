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
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	var out, errb bytes.Buffer
	if code := run([]string{"doctor", "--root", dir, "--json"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), `"ok": true`) {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestDoctorNotesWarns(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	var out, errb bytes.Buffer
	if code := run([]string{"note", "write", "Bad Name.md", "--root", dir, "--content", "x", "--no-commit"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	out.Reset()
	if code := run([]string{"doctor", "--notes", "--root", dir}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "warn notes") || !strings.Contains(out.String(), "Bad Name.md") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestReplaceAndMvViaCLI(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	var out, errb bytes.Buffer
	if code := run([]string{"note", "replace", "a.md", "--root", dir, "--content", "x"}, strings.NewReader(""), &out, &errb); code != exitFail {
		t.Fatalf("replace of missing note: exit %d", code)
	}
	if code := run([]string{"note", "write", "a.md", "--root", dir, "--content", "one"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if code := run([]string{"note", "replace", "a.md", "--root", dir, "--content", "two", "--json"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"action": "replace"`) {
		t.Fatalf("stdout=%q", out.String())
	}
	out.Reset()
	if code := run([]string{"note", "mv", "a.md", "b.md", "--root", dir, "--json"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"committed": true`) {
		t.Fatalf("stdout=%q", out.String())
	}
	errb.Reset()
	if code := run([]string{"note", "write", "b.md", "--root", dir, "--content", "three", "--force"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "deprecated") {
		t.Fatalf("stderr=%q", errb.String())
	}
}

func TestTodoNewViaCLI(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@x"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	var out, errb bytes.Buffer
	bad := [][]string{
		{"todo", "new", "Bad Slug", "--root", dir, "--content", "# x"},
		{"todo", "new", "a/b", "--root", dir, "--content", "# x"},
		{"todo", "new", "ok", "--root", dir, "--content", "# x", "--scheduled", "2026-09-15T18:00"},
		{"todo", "new", "ok", "--root", dir, "--content", "# x", "--ticket", "#7"},
		{"todo", "new", "ok", "--root", dir, "--content", "# x", "--ticket", "http://example.com/1"},
		{"todo", "new", "ok", "--root", dir, "--content", "---\na: b\n---\n# x", "--ticket", "https://example.com/1"},
	}
	for _, args := range bad {
		if code := run(args, strings.NewReader(""), &out, &errb); code != exitUsage {
			t.Fatalf("%v: exit %d, want %d: %s", args, code, exitUsage, errb.String())
		}
	}
	body := "# Retire legacy domain\n\n- [ ] delete it\n"
	args := []string{"todo", "new", "retire-legacy", "--root", dir, "--json",
		"--scheduled", "2026-09-15T18:00:00+09:00",
		"--ticket", "https://github.com/o/r/issues/7", "--ticket", "https://github.com/o/r/issues/8"}
	if code := run(args, strings.NewReader(body), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"path": "tasks/retire-legacy.md"`) || !strings.Contains(out.String(), `"committed": true`) {
		t.Fatalf("stdout=%q", out.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "tasks", "retire-legacy.md"))
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nscheduled: \"2026-09-15T18:00:00+09:00\"\ntickets:\n  - \"https://github.com/o/r/issues/7\"\n  - \"https://github.com/o/r/issues/8\"\n---\n" + body
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if code := run(args, strings.NewReader(body), &out, &errb); code != exitFail {
		t.Fatalf("overwrite: exit %d", code)
	}
	out.Reset()
	if code := run([]string{"todo", "new", "plain", "--root", dir, "--content", "# Plain\n"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	got, _ = os.ReadFile(filepath.Join(dir, "tasks", "plain.md"))
	if string(got) != "# Plain\n" {
		t.Fatalf("plain todo got frontmatter: %q", got)
	}
	out.Reset()
	if code := run([]string{"note", "ls", "tasks", "--root", dir, "--json"}, strings.NewReader(""), &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"title": "Retire legacy domain"`) {
		t.Fatalf("frontmatter broke title extraction: %s", out.String())
	}
}
