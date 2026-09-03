package main

import (
	"bytes"
	"flag"
	"os/exec"
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
