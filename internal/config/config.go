// Package config reads the config file (TOML). Config is the source of truth;
// config.example.toml documents it and a test keeps the two in step.
package config

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is the source of truth.
type Config struct {
	// Root is the notes repository every path is relative to.
	Root string `toml:"root"`
}

// Example is the bundled config example, embedded so the setup guidance and
// the document in the repo are one and the same.
//
//go:embed config.example.toml
var Example string

// Path is where the config file lives. os.UserConfigDir is avoided because
// on darwin it points at Library/Application Support, not ~/.config.
func Path() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "nabu", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "nabu", "config.toml")
	}
	return filepath.Join(home, ".config", "nabu", "config.toml")
}

// Load reads the config file. A missing file is not an error: it yields an
// empty Config and the caller decides whether root is required.
func Load() (Config, error) {
	var cfg Config
	b, err := os.ReadFile(Path())
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if _, err := toml.Decode(string(b), &cfg); err != nil {
		return cfg, fmt.Errorf("%s: %w", Path(), err)
	}
	cfg.Root = ExpandHome(cfg.Root)
	return cfg, nil
}

// ExpandHome replaces a leading ~ with the home directory.
func ExpandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}
