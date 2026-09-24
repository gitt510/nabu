package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if got := Path(); got != filepath.Join("/x", "nabu", "config.json") {
		t.Fatalf("got %s", got)
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := Load()
	if err != nil || cfg.Root != "" {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}

func TestLoadExpandsHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "nabu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(Example), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	if want := filepath.Join(home, "ghq/github.com/you/notes"); cfg.Root != want {
		t.Fatalf("got %q want %q", cfg.Root, want)
	}
}
