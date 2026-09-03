package config

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestExampleDecodes(t *testing.T) {
	var cfg Config
	meta, err := toml.Decode(Example, &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) != 0 {
		t.Fatalf("example has keys the source of truth lacks: %v", undecoded)
	}
	if cfg.Root == "" {
		t.Fatal("example must declare root")
	}
}

func TestPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if got := Path(); got != filepath.Join("/x", "nabu", "config.toml") {
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
