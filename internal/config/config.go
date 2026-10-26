// Package config reads the config file, a JSON document.
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
	// BaseURL is the nabu Worker: the viewer and the API behind one Access application.
	BaseURL string `json:"base_url"`
}

// Example is the config shown when none is found.
const Example = `{"base_url": "https://nabu.<you>.workers.dev"}
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
// empty Config and the caller decides whether a BaseURL is required.
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
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return cfg, nil
}
