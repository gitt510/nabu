// Package config reads the config file, a JSON document with one key.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config is the source of truth.
type Config struct {
	// Root is the notes repository every path is relative to.
	Root string `json:"root"`
}

// Example is the config shown when none is found.
const Example = `{"root": "~/ghq/github.com/you/notes"}
`

// Path is where the config file lives. os.UserConfigDir is avoided because
// on darwin it points at Library/Application Support, not ~/.config.
func Path() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "nabu", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "nabu", "config.json")
	}
	return filepath.Join(home, ".config", "nabu", "config.json")
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
	if err := json.Unmarshal(b, &cfg); err != nil {
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
