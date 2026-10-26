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
	if err != nil || cfg.BaseURL != "" {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}

func TestLoadTrimsTrailingSlash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "nabu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(`{"base_url": "https://nabu.example.dev/"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || cfg.BaseURL != "https://nabu.example.dev" {
		t.Fatalf("got %+v, %v", cfg, err)
	}
}
